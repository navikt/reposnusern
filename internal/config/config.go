package config

import (
	"errors"
	"os"
)

type Config struct {
	Org          string
	Token        string
	PostgresDSN  string
	Debug        bool
	SkipArchived bool
}

func LoadConfig() Config {
	return LoadConfigWithEnv(os.Getenv)
}

func LoadConfigWithEnv(getenv func(string) string) Config {
	cfg := Config{
		Org:          getenv("ORG"),
		Token:        getenv("GITHUB_TOKEN"),
		PostgresDSN:  getenv("POSTGRES_DSN"),
		Debug:        getenv("REPOSNUSERDEBUG") == "true",
		SkipArchived: getenv("REPOSNUSERARCHIVED") != "true",
	}
	return cfg
}

func ValidateConfig(cfg Config) error {
	if cfg.Org == "" {
		return errors.New("ORG må være satt")
	}
	if cfg.Token == "" {
		return errors.New("GITHUB_TOKEN må være satt")
	}
	if cfg.PostgresDSN == "" {
		return errors.New("POSTGRES_DSN må være satt")
	}
	return nil
}
