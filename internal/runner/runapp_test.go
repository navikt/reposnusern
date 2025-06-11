package runner_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/mocks"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/runner"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestRunApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RunApp Suite")
}

var _ = Describe("RunAppSafe", func() {
	var (
		ctx   context.Context
		cfg   config.Config
		deps  *mocks.MockRunnerDeps
		db    *sql.DB
		smock sqlmock.Sqlmock
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		cfg = config.Config{
			Org:         "test",
			Token:       "123",
			PostgresDSN: "mockdb",
		}
		db, smock, err = sqlmock.New()
		Expect(err).To(BeNil())

		deps = mocks.NewMockRunnerDeps(GinkgoT())
	})

	AfterEach(func() {
		if db != nil {
			err := smock.ExpectationsWereMet()
			Expect(err).To(BeNil())
		}
	})

	It("returnerer nil når Run lykkes", func() {
		mockFetcher := mocks.NewMockGraphQLFetcher(GinkgoT())

		deps.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(db, nil)

		deps.EXPECT().
			GetRepoPage(cfg, 1).
			Return([]models.RepoMeta{
				{FullName: "test/repo", Name: "repo"},
			}, nil)

		deps.EXPECT().
			GetRepoPage(cfg, 2).
			Return([]models.RepoMeta{}, nil)

		deps.EXPECT().
			Fetcher().
			Return(mockFetcher)

		mockFetcher.EXPECT().
			Fetch(cfg.Org, "repo", cfg.Token, mock.Anything).
			Return(&models.RepoEntry{})

		deps.EXPECT().
			ImportRepo(ctx, db, mock.AnythingOfType("models.RepoEntry"), 1).
			Return(nil)

		smock.ExpectClose()
		err := runner.RunAppSafe(ctx, cfg, deps)
		Expect(err).To(BeNil())

		// Verifiser at alle forventninger ble møtt
		err = smock.ExpectationsWereMet()
		Expect(err).To(BeNil())
	})

	It("returnerer feil når GetRepoPage feiler", func() {
		deps.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(db, nil)

		deps.EXPECT().
			GetRepoPage(cfg, 1).
			Return(nil, errors.New("API fail"))

		err := runner.RunAppSafe(ctx, cfg, deps)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("API fail"))
	})

	It("returnerer feil når OpenDB feiler", func() {
		deps.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(nil, errors.New("DB nede"))

		err := runner.RunAppSafe(ctx, cfg, deps)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DB nede"))
	})
})

var _ = Describe("CheckDatabaseConnection", func() {
	It("returnerer nil for en vellykket tilkobling", func() {
		db, smock, err := sqlmock.New()
		Expect(err).To(BeNil())

		smock.ExpectPing()
		smock.ExpectClose()

		originalOpenSQL := runner.OpenSQL
		runner.OpenSQL = func(driver, dsn string) (*sql.DB, error) {
			return db, nil
		}
		defer func() { runner.OpenSQL = originalOpenSQL }()

		err = runner.CheckDatabaseConnection(context.Background(), "mock-dsn")
		Expect(err).To(BeNil())

		Expect(smock.ExpectationsWereMet()).To(Succeed())
	})

	It("returnerer feil ved åpningsfeil", func() {
		originalOpenSQL := runner.OpenSQL
		runner.OpenSQL = func(driver, dsn string) (*sql.DB, error) {
			return nil, errors.New("kan ikke åpne DB")
		}
		defer func() { runner.OpenSQL = originalOpenSQL }()

		err := runner.CheckDatabaseConnection(context.Background(), "mock-dsn")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DB open-feil"))
	})
})
