package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type StorageType string

const (
	StoragePostgres StorageType = "postgres"
	StorageBigQuery StorageType = "bigquery"
)

type Config struct {
	Org               string
	Token             string
	Debug             bool
	SkipArchived      bool
	Storage           StorageType
	PostgresDSN       string
	BQProjectID       string
	BQDataset         string
	BQTable           string
	BQCredentials     string           // Valgfritt hvis GCP auth skjer automatisk
	Parallelism       int              // maks antall samtidige repo-prosesser
	Feature_Sbom      bool             // Om SBOM-funksjonalitet er aktivert
	Feature_GitHubApp bool             // Om GitHub App autentisering er aktivert
	GitHubAppConfig   *GitHubAppConfig // Valgfritt, for GitHub App autentisering
}

type GitHubAppConfig struct {
	AppID          int64
	InstallationID int64
	PrivateKey     []byte
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

	var githubAppConfig, _ = LoadGitHubAppConfig()

	cfg := Config{
		Org:               os.Getenv("ORG"),
		Token:             os.Getenv("GITHUB_TOKEN"),
		Debug:             os.Getenv("REPOSNUSERDEBUG") == "true",
		SkipArchived:      os.Getenv("REPOSNUSERARCHIVED") != "true",
		Storage:           storage,
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		BQProjectID:       os.Getenv("GCP_TEAM_PROJECT_ID"),
		BQDataset:         os.Getenv("BQ_DATASET"),
		BQTable:           os.Getenv("BQ_TABLE"),
		BQCredentials:     os.Getenv("BQ_CREDENTIALS"),
		Parallelism:       parallelism,
		Feature_Sbom:      os.Getenv("SBOM") == "true",
		Feature_GitHubApp: os.Getenv("GITHUB_APP_ENABLED") == "true",
		GitHubAppConfig:   githubAppConfig,
	}

	if cfg.Org == "" {
		return Config{}, errors.New("ORG må være satt")
	}
	if cfg.Token == "" && !cfg.Feature_GitHubApp {
		return Config{}, errors.New("GITHUB_TOKEN må være satt, eller GitHub App må være aktivert")
	}
	if cfg.Feature_GitHubApp && cfg.GitHubAppConfig == nil {
		return Config{}, errors.New("GitHub App er aktivert, men konfigurasjon mangler")
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

func LoadGitHubAppConfig() (*GitHubAppConfig, error) {
	// Get installation ID from env
	installationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	if installationID == "" {
		return nil, fmt.Errorf("missing required environment variable: GITHUB_APP_INSTALLATION_ID")
	}

	// Get app ID from env (only required for REST client)
	appID := os.Getenv("GITHUB_APP_ID")
	if appID == "" {
		return nil, fmt.Errorf("missing required environment variable: GITHUB_APP_ID")
	}

	// Load private key from env
	privateKeyPEM := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if privateKeyPEM == "" {
		return nil, fmt.Errorf("missing required environment variable: GITHUB_APP_PRIVATE_KEY")
	}
	privateKey := []byte(privateKeyPEM)

	// Parse installation ID
	installationIDInt, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_INSTALLATION_ID: %w", err)
	}

	// Parse app ID
	appIDInt, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	return &GitHubAppConfig{
		AppID:          appIDInt,
		InstallationID: installationIDInt,
		PrivateKey:     privateKey,
	}, nil
}
