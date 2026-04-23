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
	MaxDebugRepos     int64 // maks antall repos i debug-modus
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
	var errs []error

	storage := StorageType(os.Getenv("REPO_STORAGE"))

	parallelism := 1
	if pStr := os.Getenv("REPOSNUSERN_PARALL"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			parallelism = p
		} else {
			errs = append(errs, errors.New("REPOSNUSERN_PARALL må være et positivt heltall"))
		}
	}

	maxDebugRepos := int64(10)
	if val := os.Getenv("REPOSNUSER_MAXDEBUGREPOS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			maxDebugRepos = int64(i)
		} else {
			errs = append(errs, errors.New("REPOSNUSER_MAXDEBUGREPOS må være et positivt heltall"))
		}
	}

	featureGitHubApp := os.Getenv("GITHUB_APP_ENABLED") == "true"
	var githubAppConfig *GitHubAppConfig
	if featureGitHubApp {
		var err error
		githubAppConfig, err = LoadGitHubAppConfig()
		if err != nil {
			errs = append(errs, err)
		}
	}

	cfg := Config{
		Org:               os.Getenv("ORG"),
		Token:             os.Getenv("GITHUB_TOKEN"),
		Debug:             os.Getenv("REPOSNUSERDEBUG") == "true",
		MaxDebugRepos:     maxDebugRepos,
		SkipArchived:      os.Getenv("REPOSNUSERARCHIVED") != "true",
		Storage:           storage,
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		BQProjectID:       os.Getenv("GCP_TEAM_PROJECT_ID"),
		BQDataset:         os.Getenv("BQ_DATASET"),
		BQTable:           os.Getenv("BQ_TABLE"),
		BQCredentials:     os.Getenv("BQ_CREDENTIALS"),
		Parallelism:       parallelism,
		Feature_Sbom:      os.Getenv("SBOM") == "true",
		Feature_GitHubApp: featureGitHubApp,
		GitHubAppConfig:   githubAppConfig,
	}

	if cfg.Org == "" {
		errs = append(errs, errors.New("ORG må være satt"))
	}
	if cfg.Token == "" && !cfg.Feature_GitHubApp {
		errs = append(errs, errors.New("GITHUB_TOKEN må være satt, eller GitHub App må være aktivert"))
	}
	if cfg.Storage == "" {
		errs = append(errs, errors.New("REPO_STORAGE må være satt til 'postgres' eller 'bigquery'"))
	}

	switch cfg.Storage {
	case StoragePostgres:
		if cfg.PostgresDSN == "" {
			errs = append(errs, errors.New("POSTGRES_DSN må være satt for postgres-lagring"))
		}
	case StorageBigQuery:
		if cfg.BQProjectID == "" {
			errs = append(errs, errors.New("GCP_TEAM_PROJECT_ID må være satt for bigquery-lagring"))
		}
		if cfg.BQDataset == "" {
			errs = append(errs, errors.New("BQ_DATASET må være satt for bigquery-lagring"))
		}
		if cfg.BQTable == "" {
			errs = append(errs, errors.New("BQ_TABLE må være satt for bigquery-lagring"))
		}
	default:
		if cfg.Storage != "" {
			errs = append(errs, errors.New("ugyldig verdi for REPO_STORAGE – må være 'postgres' eller 'bigquery'"))
		}
	}

	if len(errs) > 0 {
		return Config{}, errors.Join(errs...)
	}

	return cfg, nil
}

func (cfg Config) DebugPrint() string {
	// Printing the raw object reveals GitHub token, use this instead
	return fmt.Sprintf("Org: %v, Token: %v, Debug: %v, MaxDebugRepos: %v, SkipArchived: %v, Storage: %v, Parallelism: %v, Feature_Sbom: %v, Feature_GitHubApp: %v",
		cfg.Org,
		(cfg.Token != ""),
		cfg.Debug,
		cfg.MaxDebugRepos,
		cfg.SkipArchived,
		cfg.Storage,
		cfg.Parallelism,
		cfg.Feature_Sbom,
		cfg.Feature_GitHubApp,
	)
}

func LoadGitHubAppConfig() (*GitHubAppConfig, error) {
	var errs []error

	// Get installation ID from env
	installationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	if installationID == "" {
		errs = append(errs, errors.New("missing required environment variable: GITHUB_APP_INSTALLATION_ID"))
	}

	// Get app ID from env (only required for REST client)
	appID := os.Getenv("GITHUB_APP_ID")
	if appID == "" {
		errs = append(errs, errors.New("missing required environment variable: GITHUB_APP_ID"))
	}

	// Load private key from env
	privateKeyPEM := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if privateKeyPEM == "" {
		errs = append(errs, errors.New("missing required environment variable: GITHUB_APP_PRIVATE_KEY"))
	}

	// Parse installation ID
	var installationIDInt int64
	if installationID != "" {
		parsedInstallationID, err := strconv.ParseInt(installationID, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid GITHUB_APP_INSTALLATION_ID: %w", err))
		} else {
			installationIDInt = parsedInstallationID
		}
	}

	// Parse app ID
	var appIDInt int64
	if appID != "" {
		parsedAppID, err := strconv.ParseInt(appID, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid GITHUB_APP_ID: %w", err))
		} else {
			appIDInt = parsedAppID
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &GitHubAppConfig{
		AppID:          appIDInt,
		InstallationID: installationIDInt,
		PrivateKey:     []byte(privateKeyPEM),
	}, nil
}
