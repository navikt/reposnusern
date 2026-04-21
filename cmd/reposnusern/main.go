package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jonmartinstorm/reposnusern/internal/bqwriter"
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/logger"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
)

func main() {
	processingCtx, stopProcessing := context.WithCancel(context.Background())
	defer stopProcessing()

	shutdownCtx, stopShutdown := context.WithCancel(context.Background())
	defer stopShutdown()

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	defer signal.Stop(signals)

	go func() {
		sig := <-signals
		slog.Warn("Mottok stoppsignal, stopper nye repositories og fullfører pågående arbeid", "signal", sig.String())
		stopShutdown()

		sig = <-signals
		slog.Warn("Mottok nytt stoppsignal, avbryter pågående arbeid", "signal", sig.String())
		stopProcessing()
	}()

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

	var writer runner.DBWriter
	// Velger lagringsmetode basert på konfigurasjon
	switch cfg.Storage {
	case config.StoragePostgres:
		slog.Info("Setter opp writer for PostgreSQL-database")
		pgWriter, err := dbwriter.NewPostgresWriter(cfg.PostgresDSN)
		if err != nil {
			slog.Error("Kunne ikke opprette databaseforbindelse til PostgreSQL", "error", err)
			os.Exit(1)
		}
		writer = pgWriter
		defer func() {
			if err := pgWriter.DB.Close(); err != nil {
				slog.Warn("Klarte ikke å lukke PostgreSQL-tilkoblingen", "error", err)
			}
		}()

	case config.StorageBigQuery:
		slog.Info("Setter opp writer for BigQuery")
		bqWriter, err := bqwriter.NewBigQueryWriter(processingCtx, &cfg)
		if err != nil {
			slog.Error("Kunne ikke opprette BigQuery-klient", "error", err)
			os.Exit(1)
		}
		writer = bqWriter

	default:
		slog.Error("Ugyldig lagringstype angitt", "storage", cfg.Storage)
		os.Exit(1)
	}

	// Initialiserer fetcher for GitHub API
	slog.Info("Setter opp fetcher med GitHub API for å hente repositories")
	getter := fetcher.NewRepoFetcher(cfg)

	app := runner.NewApp(cfg, writer, getter)

	if err := app.Run(processingCtx, shutdownCtx); err != nil {
		if errors.Is(err, context.Canceled) && processingCtx.Err() != nil {
			slog.Error("Applikasjonen ble avbrutt før pågående arbeid ble ferdig", "error", err)
			os.Exit(1)
		}
		slog.Error("Applikasjonen feilet", "error", err)
		os.Exit(1)
	}

}
