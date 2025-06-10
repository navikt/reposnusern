package runner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

const MaxDebugRepos = 10

func Run(ctx context.Context, cfg config.Config, deps RunnerDeps) error {
	slog.Info("🔁 Starter repo-import én og én")

	db, err := sql.Open("postgres", cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("DB-feil: %w", err)
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("klarte ikke å close db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)

	page := 1
	repoIndex := 0

	for {
		repos, err := fetcher.GetRepoPage(cfg, page)
		if err != nil {
			return fmt.Errorf("klarte ikke hente repo-side: %w", err)
		}
		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			if cfg.SkipArchived && repo.Archived {
				if cfg.Debug {
					slog.Info("⏭️ Skipper arkivert repo", "repo", repo.FullName)
				}
				continue
			}

			if cfg.Debug && repoIndex >= MaxDebugRepos {
				slog.Info("🛑 Debug-modus: stopper etter 10 repoer")
				return nil
			}

			slog.Info("📦 Henter detaljer via GraphQL", "repo", repo.FullName)
			entry := fetcher.FetchRepoGraphQL(cfg.Org, repo.Name, cfg.Token, repo)
			if entry == nil {
				slog.Warn("⚠️ Hopper over tomt repo", "repo", repo.FullName)
				continue
			}

			repoIndex++
			slog.Info("⏳ Behandler repo", "nummer", repoIndex, "navn", repo.FullName)

			if err := dbwriter.ImportRepo(ctx, db, *entry, repoIndex); err != nil {
				return fmt.Errorf("import repo: %w", err)
			}

			if repoIndex%25 == 0 {
				runtime.GC()
			}
		}

		page++
	}

	slog.Info("✅ Ferdig med alle repos!")
	return nil
}
