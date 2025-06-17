package postgres_test

import (
	"context"
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
	"github.com/jonmartinstorm/reposnusern/test/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestAppIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App-integrasjon")
}

var _ = Describe("runner.App", Ordered, func() {
	var (
		ctx     context.Context
		testDB  *testutils.TestDB
		cfg     config.Config
		writer  *testutils.RealPostgresWriter
		fetcher *testutils.MockFetcher
		app     *runner.App
	)

	BeforeAll(func() {
		ctx = context.Background()
		testDB = testutils.StartTestPostgresContainer()
		testutils.RunMigrations(testDB.DB)

		writer = testutils.NewRealPostgresWriter(testDB.DB)

		cfg = config.Config{
			Org:         "testorg",
			Token:       "123",
			Debug:       true,
			Parallelism: 2,
			PostgresDSN: "ignored-in-test",
		}

		mockRepos := []models.RepoMeta{
			{ID: 1, Name: "demo", FullName: "testorg/demo"},
			{ID: 2, Name: "lib", FullName: "testorg/lib"},
		}

		fetcher = &testutils.MockFetcher{}
		fetcher.On("GetReposPage", mock.Anything, cfg, 1).Return(mockRepos, nil)
		fetcher.On("GetReposPage", mock.Anything, cfg, 2).Return([]models.RepoMeta{}, nil)

		// Én forventning per repo – tryggere enn dynamisk Return
		for i, repo := range mockRepos {
			entry := &models.RepoEntry{
				Repo: models.RepoMeta{
					ID:       repo.ID,
					Name:     repo.Name,
					FullName: repo.FullName,
					Language: "Go",
					License:  &models.License{SpdxID: "MIT"},
					Topics:   []string{"oss"},
				},
				Languages: map[string]int{"Go": 1000 + i},
			}
			fetcher.On("FetchRepoGraphQL", mock.Anything, repo).Return(entry, nil)
		}

		app = runner.NewApp(cfg, writer, fetcher)
	})

	AfterAll(func() {
		testDB.Close()
	})

	It("kjører hele appen og lagrer repos i databasen", func() {
		err := app.Run(ctx)
		Expect(err).To(BeNil())

		row := testDB.DB.QueryRow(`SELECT COUNT(*) FROM repos WHERE full_name LIKE 'testorg/%'`)
		var count int
		Expect(row.Scan(&count)).To(Succeed())
		Expect(count).To(Equal(2))
	})
})
