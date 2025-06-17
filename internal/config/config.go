package config

import (
	"errors"
	"os"
	"strconv"
)

type StorageType string

const (
	StoragePostgres StorageType = "postgres"
	StorageBigQuery StorageType = "bigquery"
)

type Config struct {
	Org           string
	Token         string
	Debug         bool
	SkipArchived  bool
	Storage       StorageType
	PostgresDSN   string
	BQProjectID   string
	BQDataset     string
	BQTable       string
	BQCredentials string // Valgfritt hvis GCP auth skjer automatisk
	Parallelism   int    // maks antall samtidige repo-prosesser
}

// NewConfig oppretter en ny konfigurasjon basert på miljøvariabler
func NewConfig() (Config, error) {
	storage := StorageType(os.Getenv("REPO_STORAGE"))

	parallelism := 1
	if pStr := os.Getenv("REPOSNUSERN_PARALL"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallelism = p
		} else {
			return Config{}, errors.New("REPOSNUSERN_PARALL må være et positivt heltall")
		}
	}

	cfg := Config{
		Org:           os.Getenv("ORG"),
		Token:         os.Getenv("GITHUB_TOKEN"),
		Debug:         os.Getenv("REPOSNUSERDEBUG") == "true",
		SkipArchived:  os.Getenv("REPOSNUSERARCHIVED") != "true",
		Storage:       storage,
		PostgresDSN:   os.Getenv("POSTGRES_DSN"),
		BQProjectID:   os.Getenv("BQ_PROJECT_ID"),
		BQDataset:     os.Getenv("BQ_DATASET"),
		BQTable:       os.Getenv("BQ_TABLE"),
		BQCredentials: os.Getenv("BQ_CREDENTIALS"),
		Parallelism:   parallelism,
	}

	if cfg.Org == "" {
		return Config{}, errors.New("ORG må være satt")
	}
	if cfg.Token == "" {
		return Config{}, errors.New("GITHUB_TOKEN må være satt")
	}
	if cfg.Storage == "" {
		return Config{}, errors.New("REPO_STORAGE må være satt til 'postgres' eller 'bigquery'")
	}

	switch cfg.Storage {
	case StoragePostgres:
		if cfg.PostgresDSN == "" {
			return Config{}, errors.New("POSTGRES_DSN må være satt for postgres-lagring")
		}
	case StorageBigQuery:
		if cfg.BQProjectID == "" || cfg.BQDataset == "" || cfg.BQTable == "" {
			return Config{}, errors.New("BQ_PROJECT_ID, BQ_DATASET og BQ_TABLE må være satt for bigquery-lagring")
		}
	default:
		return Config{}, errors.New("ugyldig verdi for REPO_STORAGE – må være 'postgres' eller 'bigquery'")
	}

	return cfg, nil
}
