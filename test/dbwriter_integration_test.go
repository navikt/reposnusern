package test

import (
	"context"
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/test/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDBWriterIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DBWriter Integrasjon")
}

var _ = Describe("dbwriter.ImportRepo", Ordered, func() {
	var testDB *testutils.TestDB
	var ctx context.Context

	BeforeAll(func() {
		ctx = context.Background()
		testDB = testutils.StartTestPostgresContainer()
		testutils.RunMigrations(testDB.DB)
	})

	AfterAll(func() {
		testDB.Close()
	})

	It("skriver inn repo og språkinformasjon", func() {
		entry := models.RepoEntry{
			Repo: models.RepoMeta{
				ID:        42,
				Name:      "demo",
				FullName:  "test/demo",
				Language:  "Go",
				Topics:    []string{"fun", "oss"},
				License:   &models.License{SpdxID: "MIT"},
				UpdatedAt: "2024-01-01T00:00:00Z",
			},
			Languages: map[string]int{"Go": 12345},
		}

		err := dbwriter.ImportRepo(ctx, testDB.DB, entry, 1)
		Expect(err).To(BeNil())

		row := testDB.DB.QueryRow(`SELECT COUNT(*) FROM repos WHERE full_name = 'test/demo'`)
		var count int
		Expect(row.Scan(&count)).To(Succeed())
		Expect(count).To(Equal(1))
	})
})
