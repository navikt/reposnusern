package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/logger"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
)

func main() {
	ctx := context.Background()

	logger.SetupLogger()

	cfg, err := config.NewConfig()
	if err != nil {
		slog.Error("Ugyldig konfigurasjon:", "error", err)
		os.Exit(1)
	}

	logger.SetDebug(cfg.Debug)

	if !cfg.SkipArchived {
		slog.Info("Inkluderer arkiverte repositories")
	}

	slog.Info("Starter reposnusern...", "org", cfg.Org)

	// Initialiser writer for PostgreSQL
	slog.Info("Setter opp writer for PostgreSQL-database")
	writer, err := dbwriter.NewPostgresWriter(cfg.PostgresDSN)
	if err != nil {
		slog.Error("Kunne ikke opprette databaseforbindelse", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := writer.DB.Close(); err != nil {
			slog.Warn("Klarte ikke å lukke databaseforbindelsen", "error", err)
		}
	}()

	// Initialiserer fetcher for GitHub API
	slog.Info("Setter opp fetcher med GitHub API for å hente repositories")
	getter := fetcher.NewRepoFetcher(cfg)

	app := runner.NewApp(cfg, writer, getter)

	if err := app.Run(ctx); err != nil {
		slog.Error("Applikasjonen feilet", "error", err)
		os.Exit(1)
	}

}
