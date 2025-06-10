package fetcher

import (
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

type GitHubAPIClient struct{}

type GitHubClient interface {
	GetRepoPage(cfg config.Config, page int) ([]models.RepoMeta, error)
}

type GraphQLFetcher interface {
	Fetch(owner, name, token string, baseRepo models.RepoMeta) *models.RepoEntry
}

// Kombinerer begge i én (valgfritt, men ryddig hvis du bruker begge via én struct)
type GitHubAPI interface {
	GitHubClient
	GraphQLFetcher
}
