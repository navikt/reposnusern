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
)

func TestRunApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RunApp Suite")
}

var _ = Describe("RunAppSafe", func() {
	var (
		ctx  context.Context
		cfg  config.Config
		mock *mocks.MockRunnerDeps
		db   *sql.DB
	)

	BeforeEach(func() {
		var err error
		ctx = context.Background()
		cfg = config.Config{
			Org:         "test",
			Token:       "123",
			PostgresDSN: "mockdb",
		}
		db, _, err = sqlmock.New()
		Expect(err).To(BeNil())

		mock = mocks.NewMockRunnerDeps(GinkgoT())
	})

	AfterEach(func() {
		db.Close()
	})

	It("returnerer nil når Run lykkes", func() {
		mock.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(db, nil)

		mock.EXPECT().
			GetRepoPage(cfg, 1).
			Return([]models.RepoMeta{}, nil)

		err := runner.RunAppSafe(ctx, cfg, mock)
		Expect(err).To(BeNil())
	})

	It("returnerer feil når GetRepoPage feiler", func() {
		mock.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(db, nil)

		mock.EXPECT().
			GetRepoPage(cfg, 1).
			Return(nil, errors.New("API fail"))

		err := runner.RunAppSafe(ctx, cfg, mock)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("API fail"))
	})

	It("returnerer feil når OpenDB feiler", func() {
		mock.EXPECT().
			OpenDB(cfg.PostgresDSN).
			Return(nil, errors.New("DB nede"))

		err := runner.RunAppSafe(ctx, cfg, mock)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("DB nede"))
	})
})
