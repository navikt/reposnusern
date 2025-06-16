package runner_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/mocks"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRunner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Runner Suite")
}

var _ = Describe("App.Run", func() {
	var (
		ctx     context.Context
		cfg     config.Config
		writer  *mocks.MockDBWriter
		fetcher *mocks.MockFetcher
		app     *runner.App
	)

	BeforeEach(func() {
		ctx = context.Background()
		cfg = config.Config{
			Org:         "testorg",
			Token:       "fake-token",
			PostgresDSN: "mockdsn",
			Debug:       true, // så vi stopper etter 10 repo
		}

		writer = &mocks.MockDBWriter{}
		fetcher = &mocks.MockFetcher{}
		app = runner.NewApp(cfg, writer, fetcher)
	})

	It("returnerer feil hvis GetReposPage feiler", func() {
		fetcher.On("GetReposPage", ctx, cfg, 1).
			Return(nil, errors.New("API-feil"))

		err := app.Run(ctx)
		Expect(err).To(MatchError(ContainSubstring("API-feil")))
	})

	It("hopper over arkiverte repo hvis SkipArchived er true", func() {
		cfg.SkipArchived = true
		app = runner.NewApp(cfg, writer, fetcher)

		archived := models.RepoMeta{FullName: "repo1", Archived: true}
		fetcher.On("GetReposPage", ctx, cfg, 1).Return([]models.RepoMeta{archived}, nil)
		fetcher.On("GetReposPage", ctx, cfg, 2).Return([]models.RepoMeta{}, nil)

		err := app.Run(ctx)
		Expect(err).To(BeNil())
	})

	It("stopper etter maks 10 repo i debug-modus", func() {
		// Returner 10 ikke-arkiverte repos
		var repos []models.RepoMeta
		for i := 0; i < 10; i++ {
			repos = append(repos, models.RepoMeta{FullName: "repo", Name: "name"})
		}
		fetcher.On("GetReposPage", ctx, cfg, 1).Return(repos, nil)

		// Return en tom page etterpå
		fetcher.On("GetReposPage", ctx, cfg, 2).Return([]models.RepoMeta{}, nil)

		// Returner dummy data for GraphQL og ImportRepo
		for i := 0; i < 10; i++ {
			entry := &models.RepoEntry{}
			fetcher.On("FetchRepoGraphQL", ctx, repos[i]).Return(entry, nil)
			writer.On("ImportRepo", ctx, *entry, i+1, mock.AnythingOfType("time.Time")).Return(nil)
		}

		err := app.Run(ctx)
		Expect(err).To(BeNil())
	})
})
