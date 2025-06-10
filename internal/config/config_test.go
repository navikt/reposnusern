package config_test

import (
	"os"
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/config"
)

func TestLoadConfig(t *testing.T) {
	os.Setenv("ORG", "navikt")
	os.Setenv("GITHUB_TOKEN", "abc123")
	os.Setenv("POSTGRES_DSN", "postgres://...")
	os.Setenv("REPOSNUSERDEBUG", "true")
	os.Setenv("REPOSNUSERARCHIVED", "true")

	cfg := config.LoadConfig()

	if cfg.Org != "navikt" {
		t.Errorf("expected ORG to be navikt, got %s", cfg.Org)
	}
	if !cfg.Debug {
		t.Errorf("expected Debug to be true")
	}
	if cfg.SkipArchived {
		t.Errorf("expected SkipArchived to be false")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr string
	}{
		{
			name:    "missing org",
			cfg:     config.Config{Token: "t", PostgresDSN: "dsn"},
			wantErr: "ORG",
		},
		{
			name:    "missing token",
			cfg:     config.Config{Org: "o", PostgresDSN: "dsn"},
			wantErr: "GITHUB_TOKEN",
		},
		{
			name:    "missing DSN",
			cfg:     config.Config{Org: "o", Token: "t"},
			wantErr: "POSTGRES_DSN",
		},
		{
			name:    "all valid",
			cfg:     config.Config{Org: "o", Token: "t", PostgresDSN: "dsn"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateConfig(tt.cfg)
			if tt.wantErr == "" && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr != "" && (err == nil || !contains(err.Error(), tt.wantErr)) {
				t.Errorf("expected error to contain %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return s != "" && substr != "" && (s == substr || len(s) >= len(substr) && (s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr || contains(s[1:], substr)))
}
