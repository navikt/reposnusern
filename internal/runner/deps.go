package runner

import (
	"context"
	"database/sql"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

type RunnerDeps interface {
	OpenDB(dsn string) (*sql.DB, error)
	GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error)
	FetchRepoGraphQL(org, name, token string, base models.RepoMeta) *models.RepoEntry
	ImportRepo(ctx context.Context, db *sql.DB, entry models.RepoEntry, index int) error
}

type RealDeps struct {
	GitHub fetcher.GitHubClient
}

func (RealDeps) OpenDB(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}
func (r RealDeps) GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error) {
	return r.GitHub.GetRepoPage(cfg, page)
}
func (RealDeps) FetchRepoGraphQL(org, name, token string, base models.RepoMeta) *models.RepoEntry {
	return fetcher.FetchRepoGraphQL(org, name, token, base)
}
func (RealDeps) ImportRepo(ctx context.Context, db *sql.DB, entry models.RepoEntry, index int) error {
	return dbwriter.ImportRepo(ctx, db, entry, index)
}
