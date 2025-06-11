package runner

import (
	"context"
	"database/sql"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

type RunnerDeps interface {
	OpenDB(dsn string) (*sql.DB, error)
	GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error)
	ImportRepo(ctx context.Context, db *sql.DB, entry models.RepoEntry, index int) error
	Fetcher() fetcher.GraphQLFetcher
}
