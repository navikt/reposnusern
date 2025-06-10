package fetcher

import (
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

type GitHubClient interface {
	GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error)
}
