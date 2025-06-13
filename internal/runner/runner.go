package runner

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
)

const MaxDebugRepos = 10

func Run(ctx context.Context, cfg config.Config, deps RunnerDeps) error {
	slog.Info("Starter repo-import én og én")

	db, err := deps.OpenDB(cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("DB-feil: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Warn("Klarte ikke å lukke databaseforbindelsen", "error", err)
		}
	}()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(10 * time.Minute)

	page := 1
	repoIndex := 0

	for {
		repos, err := deps.GetRepoPage(cfg, page)
		if err != nil {
			return fmt.Errorf("klarte ikke hente repo-side: %w", err)
		}
		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			if cfg.SkipArchived && repo.Archived {
				if cfg.Debug {
					slog.Info("Skipper arkivert repo", "repo", repo.FullName)
				}
				continue
			}

			if cfg.Debug && repoIndex >= MaxDebugRepos {
				slog.Info("Debug-modus: stopper etter 10 repoer")
				return nil
			}

			slog.Info("Henter detaljer via GraphQL", "repo", repo.FullName)
			entry := deps.Fetcher().Fetch(cfg.Org, repo.Name, cfg.Token, repo)
			if entry == nil {
				slog.Warn("Hopper over tomt repo", "repo", repo.FullName)
				continue
			}

			repoIndex++
			slog.Info("Behandler repo", "nummer", repoIndex, "navn", repo.FullName)

			if err := deps.ImportRepo(ctx, db, *entry, repoIndex); err != nil {
				return fmt.Errorf("import repo: %w", err)
			}

			if repoIndex%25 == 0 {
				runtime.GC()
			}
		}

		page++
	}

	slog.Info("Ferdig med alle repos!")
	return nil
}
