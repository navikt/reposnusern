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

// NewConfig oppretter en ny Config basert på miljøvariabler.
// Den validerer også konfigurasjonen og returnerer en feil hvis noe mangler.
// Denne funksjonen bør kalles i main.go for å sette opp konfigurasjonen før applikasjonen starter.
func NewConfig() (Config, error) {
	cfg := Config{
		Org:          os.Getenv("ORG"),
		Token:        os.Getenv("GITHUB_TOKEN"),
		PostgresDSN:  os.Getenv("POSTGRES_DSN"),
		Debug:        os.Getenv("REPOSNUSERDEBUG") == "true",
		SkipArchived: os.Getenv("REPOSNUSERARCHIVED") != "true",
	}

	if cfg.Org == "" {
		return Config{}, errors.New("ORG må være satt")
	}
	if cfg.Token == "" {
		return Config{}, errors.New("GITHUB_TOKEN må være satt")
	}
	if cfg.PostgresDSN == "" {
		return Config{}, errors.New("POSTGRES_DSN må være satt")
	}
	return cfg, nil
}
