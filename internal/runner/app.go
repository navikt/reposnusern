package runner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	_ "github.com/lib/pq"
)

const MaxDebugRepos = 10

type DBWriter interface {
	ImportRepo(ctx context.Context, entry models.RepoEntry, index int, snapshotDate time.Time) error
}

type Fetcher interface {
	GetReposPage(ctx context.Context, cfg config.Config, page int) ([]models.RepoMeta, error)
	FetchRepoGraphQL(ctx context.Context, baseRepo models.RepoMeta) (*models.RepoEntry, error)
}

type App struct {
	Cfg     config.Config
	Writer  DBWriter
	Fetcher Fetcher
}

var OpenSQL = sql.Open

func NewApp(cfg config.Config, writer DBWriter, fetcher Fetcher) *App {
	return &App{
		Cfg:     cfg,
		Writer:  writer,
		Fetcher: fetcher,
	}
}

func (a *App) Run(ctx context.Context) error {
	start := time.Now()
	snapshotDate := time.Now().Truncate(24 * time.Hour)
	slog.Info("Starter snapshot", "dato", snapshotDate.Format("2006-01-02"))

	page := 1
	repoIndex := 0

	for {
		repos, err := a.Fetcher.GetReposPage(ctx, a.Cfg, page)
		if err != nil {
			return fmt.Errorf("klarte ikke hente repo-side: %w", err)
		}
		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			if a.Cfg.SkipArchived && repo.Archived {
				slog.Debug("Skipper arkivert repo", "repo", repo.FullName)
				continue
			}

			if a.Cfg.Debug && repoIndex >= MaxDebugRepos {
				slog.Info("Debug-modus: stopper etter 10 repoer")
				return nil
			}

			slog.Info("Henter detaljer via GraphQL", "repo", repo.FullName)
			entry, err := a.Fetcher.FetchRepoGraphQL(ctx, repo)
			if err != nil {
				slog.Error("Kunne ikke hente repo via GraphQL", "repo", repo.FullName, "error", err)
				continue
			}

			repoIndex++
			slog.Info("Behandler repo", "nummer", repoIndex, "navn", repo.FullName)

			if err := a.Writer.ImportRepo(ctx, *entry, repoIndex, snapshotDate); err != nil {
				return fmt.Errorf("import repo: %w", err)
			}

			if repoIndex%25 == 0 {
				runtime.GC()
			}
		}

		page++
	}

	logMemoryStats()
	slog.Info("Ferdig med alle repos!", "varighet", time.Since(start).String())

	return nil
}

func logMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Debug("Minnebruk",
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
