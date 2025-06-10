package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
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

	// Last inn env og legg i config.
	cfg := config.LoadConfig()
	if err := config.ValidateConfig(cfg); err != nil {
		log.Fatal(err)
	}

	if !cfg.SkipArchived {
		slog.Info("📦 Inkluderer arkiverte repositories")
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
	if err := testDB.Close(); err != nil {
		slog.Error("warning: failed to close testDB", "error", err)
		os.Exit(1)
	}
	slog.Info("✅ DB-tilkobling OK")

	if err := runner.RunApp(ctx, cfg); err != nil {
		slog.Error("🚨 Applikasjonen feilet", "error", err)
		os.Exit(1)
	}

}
