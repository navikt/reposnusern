package runner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

func Run(ctx context.Context, cfg config.Config) error {
	slog.Info("🔁 Starter repo-import per page")

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("DB-feil: %w", err)
	}
	defer db.Close()

	page := 1
	for {
		repos, err := fetcher.GetRepoPage(cfg, page)
		if err != nil {
			return err
		}
		if len(repos) == 0 {
			break
		}

		slog.Info("📦 Henter detaljer via GraphQL", "antall", len(repos))
		data := fetcher.GetDetailsActiveReposGraphQL(cfg.Org, cfg.Token, repos)

		slog.Info("📝 Skriver til DB", "antall_repos", len(data.Repos))
		if err := dbwriter.ImportToPostgreSQLDB(data, db); err != nil {
			return err
		}

		// FLUSH DATA
		data = models.OrgRepos{} // tom struct
		repos = nil              // slice nulles
		page++
		if cfg.Debug {
			break // for å ikke gå uendelig i test
		}

		// Hint til GC ved høy minnebruk
		if page%5 == 0 {
			runtime.GC()
		}
	}
	slog.Info("✅ Ferdig med all import")
	return nil
}
