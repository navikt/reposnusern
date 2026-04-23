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
		processingCtx context.Context
		shutdownCtx   context.Context
		cfg           config.Config
		writer        *mocks.MockDBWriter
		fetcher       *mocks.MockFetcher
		app           *runner.App
	)

	BeforeEach(func() {
		processingCtx = context.Background()
		shutdownCtx = context.Background()
		cfg = config.Config{
			Org:           "testorg",
			Token:         "fake-token",
			PostgresDSN:   "mockdsn",
			Debug:         true,
			Parallelism:   2,
			MaxDebugRepos: 10,
		}

		writer = &mocks.MockDBWriter{}
		fetcher = &mocks.MockFetcher{}
		app = runner.NewApp(cfg, writer, fetcher)
	})

	It("returnerer feil hvis GetReposPage feiler", func() {
		fetcher.On("GetReposPage", mock.Anything, cfg, 1).
			Return(nil, errors.New("API-feil"))

		err := app.Run(processingCtx, shutdownCtx)
		Expect(err).To(MatchError(ContainSubstring("API-feil")))
	})

	It("hopper over arkiverte repo hvis SkipArchived er true", func() {
		cfg.SkipArchived = true
		app = runner.NewApp(cfg, writer, fetcher)

		archived := models.RepoMeta{FullName: "repo1", Archived: true}
		fetcher.On("GetReposPage", mock.Anything, cfg, 1).Return([]models.RepoMeta{archived}, nil)
		fetcher.On("GetReposPage", mock.Anything, cfg, 2).Return([]models.RepoMeta{}, nil)

		err := app.Run(processingCtx, shutdownCtx)
		Expect(err).To(BeNil())
	})

	It("stopper etter maks 10 repo i debug-modus", func() {
		app = runner.NewApp(cfg, writer, fetcher)

		var repos []models.RepoMeta
		for i := 0; i < 10; i++ {
			repos = append(repos, models.RepoMeta{FullName: "repo", Name: "name"})
		}
		fetcher.On("GetReposPage", mock.Anything, cfg, 1).Return(repos, nil)

		// Vi forventer at side 2 aldri blir hentet
		fetcher.On("GetReposPage", mock.Anything, cfg, 2).Return([]models.RepoMeta{}, nil)

		for i := 0; i < 10; i++ {
			entry := &models.RepoEntry{}
			fetcher.On("FetchRepoGraphQL", mock.Anything, repos[i]).Return(entry, nil)
			writer.On("ImportRepo", mock.Anything, *entry, mock.AnythingOfType("time.Time")).Return(nil)

		}

		err := app.Run(processingCtx, shutdownCtx)
		Expect(err).To(BeNil())
		Expect(writer.Calls).To(HaveLen(10))
	})

	It("dispatcher ikke flere repos enn debug-grensen selv med parallelle workere", func() {
		cfg.MaxDebugRepos = 1
		cfg.Parallelism = 2
		app = runner.NewApp(cfg, writer, fetcher)

		repo1 := models.RepoMeta{FullName: "testorg/repo1", Name: "repo1"}
		repo2 := models.RepoMeta{FullName: "testorg/repo2", Name: "repo2"}
		repos := []models.RepoMeta{repo1, repo2}
		entry := &models.RepoEntry{}
		repoStarted := make(chan struct{})
		releaseRepo := make(chan struct{})

		fetcher.On("GetReposPage", mock.Anything, cfg, 1).Return(repos, nil)
		fetcher.On("FetchRepoGraphQL", mock.Anything, repo1).Run(func(mock.Arguments) {
			close(repoStarted)
			<-releaseRepo
		}).Return(entry, nil).Once()
		writer.On("ImportRepo", mock.Anything, *entry, mock.AnythingOfType("time.Time")).Return(nil).Once()

		done := make(chan error, 1)
		go func() {
			done <- app.Run(processingCtx, shutdownCtx)
		}()

		<-repoStarted
		close(releaseRepo)

		Expect(<-done).To(Succeed())
		fetcher.AssertNotCalled(GinkgoT(), "FetchRepoGraphQL", mock.Anything, repo2)
		writer.AssertNumberOfCalls(GinkgoT(), "ImportRepo", 1)
	})

	It("hopper over repo der GraphQL feiler og fortsetter", func() {
		repo := models.RepoMeta{FullName: "testorg/fails", Name: "fails"}
		fetcher.On("GetReposPage", mock.Anything, cfg, 1).Return([]models.RepoMeta{repo}, nil)
		fetcher.On("GetReposPage", mock.Anything, cfg, 2).Return([]models.RepoMeta{}, nil)
		fetcher.On("FetchRepoGraphQL", mock.Anything, repo).Return(nil, errors.New("graphql error"))

		err := app.Run(processingCtx, shutdownCtx)
		Expect(err).To(BeNil())
		writer.AssertNotCalled(GinkgoT(), "ImportRepo")
	})

	It("stopper nye repos ved shutdown men fullfører pågående arbeid", func() {
		cfg.Debug = false
		cfg.Parallelism = 1
		app = runner.NewApp(cfg, writer, fetcher)

		shutdownCtx, stopShutdown := context.WithCancel(context.Background())
		defer stopShutdown()

		repo1 := models.RepoMeta{FullName: "testorg/repo1", Name: "repo1"}
		repo2 := models.RepoMeta{FullName: "testorg/repo2", Name: "repo2"}
		entry := &models.RepoEntry{}
		repoStarted := make(chan struct{})
		releaseRepo := make(chan struct{})

		fetcher.On("GetReposPage", mock.MatchedBy(func(ctx context.Context) bool {
			return ctx == shutdownCtx
		}), cfg, 1).Return([]models.RepoMeta{repo1, repo2}, nil)
		fetcher.On("FetchRepoGraphQL", mock.Anything, repo1).Run(func(mock.Arguments) {
			close(repoStarted)
			<-releaseRepo
		}).Return(entry, nil)
		writer.On("ImportRepo", mock.Anything, *entry, mock.AnythingOfType("time.Time")).Return(nil)

		done := make(chan error, 1)
		go func() {
			done <- app.Run(processingCtx, shutdownCtx)
		}()

		<-repoStarted
		stopShutdown()
		close(releaseRepo)

		Expect(<-done).To(Succeed())
		writer.AssertNumberOfCalls(GinkgoT(), "ImportRepo", 1)
		fetcher.AssertNotCalled(GinkgoT(), "FetchRepoGraphQL", mock.Anything, repo2)
	})
})
