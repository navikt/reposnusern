package config_test

import (
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("LoadConfigWithEnv", func() {
	It("should load config from fake env", func() {
		mockEnv := map[string]string{
			"ORG":                "org",
			"GITHUB_TOKEN":       "abc123",
			"POSTGRES_DSN":       "postgres://...",
			"REPOSNUSERDEBUG":    "true",
			"REPOSNUSERARCHIVED": "true",
		}

		getenv := func(key string) string {
			return mockEnv[key]
		}

		cfg := config.LoadConfigWithEnv(getenv)

		Expect(cfg.Org).To(Equal("org"))
		Expect(cfg.Debug).To(BeTrue())
		Expect(cfg.SkipArchived).To(BeFalse())
	})
})

var _ = Describe("ValidateConfig", func() {
	It("should return error if org is missing", func() {
		cfg := config.Config{Token: "t", PostgresDSN: "dsn"}
		err := config.ValidateConfig(cfg)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("ORG"))
	})

	It("should return error if token is missing", func() {
		cfg := config.Config{Org: "o", PostgresDSN: "dsn"}
		err := config.ValidateConfig(cfg)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GITHUB_TOKEN"))
	})

	It("should return error if DSN is missing", func() {
		cfg := config.Config{Org: "o", Token: "t"}
		err := config.ValidateConfig(cfg)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("POSTGRES_DSN"))
	})

	It("should pass if all fields are valid", func() {
		cfg := config.Config{Org: "o", Token: "t", PostgresDSN: "dsn"}
		err := config.ValidateConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
	})
})
