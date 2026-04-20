package config

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewConfig", func() {
	var trackedEnvVars = []string{
		"ORG",
		"GITHUB_TOKEN",
		"REPO_STORAGE",
		"POSTGRES_DSN",
		"GCP_TEAM_PROJECT_ID",
		"BQ_DATASET",
		"BQ_TABLE",
		"BQ_CREDENTIALS",
		"REPOSNUSERN_PARALL",
		"REPOSNUSER_MAXDEBUGREPOS",
		"REPOSNUSERDEBUG",
		"REPOSNUSERARCHIVED",
		"SBOM",
		"GITHUB_APP_ENABLED",
		"GITHUB_APP_ID",
		"GITHUB_APP_INSTALLATION_ID",
		"GITHUB_APP_PRIVATE_KEY",
	}

	BeforeEach(func() {
		for _, key := range trackedEnvVars {
			originalValue, hadValue := os.LookupEnv(key)
			Expect(os.Unsetenv(key)).To(Succeed())
			DeferCleanup(func(envKey, envValue string, envWasSet bool) func() {
				return func() {
					var err error
					if envWasSet {
						err = os.Setenv(envKey, envValue)
					} else {
						err = os.Unsetenv(envKey)
					}
					Expect(err).NotTo(HaveOccurred())
				}
			}(key, originalValue, hadValue))
		}
	})

	It("aggregates setup validation errors instead of returning the first one", func() {
		Expect(os.Setenv("GITHUB_APP_ENABLED", "true")).To(Succeed())
		Expect(os.Setenv("REPOSNUSERN_PARALL", "0")).To(Succeed())

		_, err := NewConfig()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("REPOSNUSERN_PARALL må være et positivt heltall"))
		Expect(err.Error()).To(ContainSubstring("missing required environment variable: GITHUB_APP_INSTALLATION_ID"))
		Expect(err.Error()).To(ContainSubstring("missing required environment variable: GITHUB_APP_ID"))
		Expect(err.Error()).To(ContainSubstring("missing required environment variable: GITHUB_APP_PRIVATE_KEY"))
		Expect(err.Error()).To(ContainSubstring("ORG må være satt"))
		Expect(err.Error()).To(ContainSubstring("REPO_STORAGE må være satt til 'postgres' eller 'bigquery'"))
	})

	It("reports all missing BigQuery settings together", func() {
		Expect(os.Setenv("ORG", "navikt")).To(Succeed())
		Expect(os.Setenv("GITHUB_TOKEN", "token")).To(Succeed())
		Expect(os.Setenv("REPO_STORAGE", string(StorageBigQuery))).To(Succeed())

		_, err := NewConfig()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GCP_TEAM_PROJECT_ID må være satt for bigquery-lagring"))
		Expect(err.Error()).To(ContainSubstring("BQ_DATASET må være satt for bigquery-lagring"))
		Expect(err.Error()).To(ContainSubstring("BQ_TABLE må være satt for bigquery-lagring"))
	})
})

var _ = Describe("LoadGitHubAppConfig", func() {
	It("reports all invalid numeric GitHub App values together", func() {
		Expect(os.Setenv("GITHUB_APP_INSTALLATION_ID", "not-a-number")).To(Succeed())
		Expect(os.Setenv("GITHUB_APP_ID", "still-not-a-number")).To(Succeed())
		Expect(os.Setenv("GITHUB_APP_PRIVATE_KEY", "pem")).To(Succeed())
		DeferCleanup(func() {
			Expect(os.Unsetenv("GITHUB_APP_INSTALLATION_ID")).To(Succeed())
			Expect(os.Unsetenv("GITHUB_APP_ID")).To(Succeed())
			Expect(os.Unsetenv("GITHUB_APP_PRIVATE_KEY")).To(Succeed())
		})

		_, err := LoadGitHubAppConfig()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invalid GITHUB_APP_INSTALLATION_ID"))
		Expect(err.Error()).To(ContainSubstring("invalid GITHUB_APP_ID"))
		Expect(err.Error()).NotTo(ContainSubstring("missing required environment variable"))
	})
})
