package runner

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

func Run(ctx context.Context, cfg config.Config) error {
	slog.Info("🔍 Henter oversikt over alle repos (paged 100 og 100)")
	allRepos := fetcher.GetAllRepos(cfg)

	for i := 0; i < len(allRepos); i += 100 {
		end := i + 100
		if end > len(allRepos) {
			end = len(allRepos)
		}
		batch := allRepos[i:end]

		slog.Info("📦 Henter detaljert info for repos", "batch_start", i, "batch_end", end)
		batchData := fetcher.GetDetailsActiveReposGraphQL(cfg.Org, cfg.Token, batch)

		db, err := sql.Open("postgres", cfg.PostgresDSN)
		if err != nil {
			slog.Error("Kunne ikke åpne DB-forbindelse", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		slog.Info("📝 Skriver batch til DB", "antall_repos", len(batchData.Repos))
		if err := dbwriter.ImportToPostgreSQLDB(batchData, db); err != nil {
			return err
		}
	}

	slog.Info("✅ Ferdig importert hele organisasjonen!")
	return nil
}
