package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
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
	}()

	org := os.Getenv("ORG")
	if org == "" {
		slog.Error("Du må angi organisasjon via ORG=<orgnavn>")
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		slog.Error("Mangler GITHUB_TOKEN i environment")
		os.Exit(1)
	}

	start := time.Now()

	// Hent repo-liste fra GitHub
	slog.Info("🔍 Henter oversikt over alle repos")
	repos := fetcher.GetAllRepos(org, token)

	// Lagre oversikt som JSON
	if err := fetcher.StoreRepoDumpJSON("data", org, repos); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

	// Hent detaljer via GraphQL
	slog.Info("📦 Henter detaljert info for aktive repos")
	allData := fetcher.GetDetailsActiveReposGraphQL(org, token, repos)

	// Lagre detaljert info som JSON
	if err := fetcher.StoreRepoDetailedJSON("data", org, allData); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

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
