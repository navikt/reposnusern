package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
	_ "github.com/lib/pq"
)

type AppDeps struct {
	GitHub fetcher.GitHubClient
}

func (AppDeps) OpenDB(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}

func (a AppDeps) GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error) {
	return a.GitHub.GetRepoPage(cfg, page)
}

func (AppDeps) FetchRepoGraphQL(org, name, token string, base models.RepoMeta) *models.RepoEntry {
	return fetcher.FetchRepoGraphQL(org, name, token, base)
}

func (AppDeps) ImportRepo(ctx context.Context, db *sql.DB, entry models.RepoEntry, index int) error {
	return dbwriter.ImportRepo(ctx, db, entry, index)
}

func main() {
	// Context for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	go func() {
		<-ctx.Done()
		slog.Info("SIGTERM mottatt – rydder opp...")
		// Her kan vi legge til ekstra rydding om vi trenger det
		// TODO sende context til dbcall og skriving av filer.
	}()

	runner.SetupLogger()
	cfg := config.LoadAndValidateConfig()

	runner.CheckDatabaseConnection(ctx, cfg.PostgresDSN)

	if !cfg.SkipArchived {
		slog.Info("📦 Inkluderer arkiverte repositories")
	}

	deps := AppDeps{
		GitHub: &fetcher.GitHubAPI{},
	}

	if err := runner.RunApp(ctx, cfg, deps); err != nil {
		slog.Error("🚨 Applikasjonen feilet", "error", err)
		os.Exit(1)
	}

}
