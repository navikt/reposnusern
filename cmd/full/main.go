package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)

	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go func() {
		<-ctx.Done()
		slog.Info("SIGTERM mottatt – rydder opp...")
		// Her kan vi legge til ekstra rydding om vi trenger det
		// TODO sende context til dbcall og skriving av filer.
	}()

	cfg := config.LoadConfig()
	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatal(err)
	}

	// Test tidlig
	testDB, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		slog.Error("Kunne ikke åpne DB-forbindelse", "error", err)
		os.Exit(1)
	}
	if err := testDB.PingContext(ctx); err != nil {
		slog.Error("❌ Klarte ikke å nå databasen", "error", err)
		os.Exit(1)
	}
	testDB.Close()
	slog.Info("✅ DB-tilkobling OK")

	start := time.Now()

	// Hent repo-liste fra GitHub
	slog.Info("🔍 Henter oversikt over alle repos")
	repos := fetcher.GetAllRepos(cfg)

	// Hent detaljer via GraphQL
	slog.Info("📦 Henter detaljert info for aktive repos")
	allData := fetcher.GetDetailsActiveReposGraphQL(cfg.Org, cfg.Token, repos)

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		slog.Error("Kunne ikke åpne DB-forbindelse", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// skriv til postgresql.
	slog.Info("🚀 Importerer til PostgreSQL", "cfg.Org", allData.Org, "antall_repos", len(allData.Repos))

	err = dbwriter.ImportToPostgreSQLDB(allData, db)
	if err != nil {
		slog.Error("Kunne ikke skrive til DB", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Ferdig importert!")

	logMemoryStats()

	elapsed := time.Since(start)
	slog.Info("✅ Ferdig!", "varighet", elapsed.String())
}

// Logger topp minnebruk
func logMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("📊 Minnebruk",
		"alloc", byteSize(m.Alloc),
		"totalAlloc", byteSize(m.TotalAlloc),
		"sys", byteSize(m.Sys),
		"numGC", m.NumGC)
}

func byteSize(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
