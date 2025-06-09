package config

import (
	"errors"
	"os"
)

type Config struct {
	Org         string
	Token       string
	PostgresDSN string
	Debug       bool
}

func LoadConfig() Config {
	return Config{
		Org:         os.Getenv("ORG"),
		Token:       os.Getenv("GITHUB_TOKEN"),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
		Debug:       os.Getenv("REPOSNUSERDEBUG") == "false",
	}
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
