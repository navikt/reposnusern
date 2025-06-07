package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		slog.Error("❌ POSTGRES_DSN ikke satt")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("Kunne ikke koble til Postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	data, err := os.ReadFile("data/navikt_analysis_data.json")
	if err != nil {
		slog.Error("Kunne ikke lese JSON", "error", err)
		os.Exit(1)
	}

	var dump dbwriter.Dump
	if err := json.Unmarshal(data, &dump); err != nil {
		slog.Error("Kunne ikke parse JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("🚀 Importerer til PostgreSQL", "org", dump.Org, "antall_repos", len(dump.Repos))

	err = dbwriter.ImportToPostgreSQLDB(dump, db)
	if err != nil {
		slog.Error("Kunne ikke skrive til DB", "error", err)
		os.Exit(1)
	}

	slog.Info("✅ Ferdig importert!")
}
