package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	DB        *sql.DB
	container testcontainers.Container
}

type MockFetcher struct {
	mock.Mock
}

func (m *MockFetcher) GetReposPage(ctx context.Context, cfg config.Config, page int) ([]models.RepoMeta, error) {
	args := m.Called(ctx, cfg, page)
	return args.Get(0).([]models.RepoMeta), args.Error(1)
}

func (m *MockFetcher) FetchRepoGraphQL(ctx context.Context, base models.RepoMeta) (*models.RepoEntry, error) {
	args := m.Called(ctx, base)
	return args.Get(0).(*models.RepoEntry), args.Error(1)
}

type RealPostgresWriter struct {
	db *sql.DB
}

func NewRealPostgresWriter(db *sql.DB) *RealPostgresWriter {
	return &RealPostgresWriter{db: db}
}

func (r *RealPostgresWriter) ImportRepo(ctx context.Context, entry models.RepoEntry, snapshot time.Time) error {
	pw := &dbwriter.PostgresWriter{DB: r.db}
	return pw.ImportRepo(ctx, entry, snapshot)
}

func StartTestPostgresContainer() *TestDB {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:      "postgres:15",
		SkipReaper: true, // Unngå problemer med Ryuk på macOS/Podman
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("Kunne ikke starte testcontainer: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("Klarte ikke hente host fra container: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("Klarte ikke hente port fra container: %v", err)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	var db *sql.DB
	for retries := 0; retries < 10; retries++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil && db.PingContext(ctx) == nil {
			log.Println("Databasen er klar")
			break
		}
		log.Println("Venter på at databasen skal bli klar...")
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatalf("Klarte ikke koble til databasen: %v", err)
	}

	return &TestDB{
		DB:        db,
		container: container,
	}
}

func (t *TestDB) Close() {
	ctx := context.Background()

	if err := t.DB.Close(); err != nil {
		log.Printf("Kunne ikke lukke databaseforbindelsen: %v", err)
	}
	if err := t.container.Terminate(ctx); err != nil {
		log.Printf("Kunne ikke stoppe testcontaineren: %v", err)
	}
}

func RunMigrations(db *sql.DB) {
	root, err := os.Getwd()
	if err != nil {
		log.Fatalf("Kunne ikke hente arbeidskatalog: %v", err)
	}

	schemaPath := root + "/db/schema.sql"
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		schemaPath = root + "/../../db/schema.sql"
	}

	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Kunne ikke lese schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatalf("Klarte ikke å kjøre migrering: %v", err)
	}
}
