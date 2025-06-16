package fetcher_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

// Ginkgo sin test-runner. Denne trengs for at "go test" skal vite hvor den skal starte.
func TestFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fetcher – GraphQL-funksjoner")
}

var _ = Describe("GraphQL-relaterte hjelpefunksjoner", func() {

	Describe("convertToFileEntries", func() {
		It("skal konvertere liste med path/content til FileEntry-struktur", func() {
			input := []map[string]string{
				{"path": "Dockerfile", "content": "FROM alpine"},
				{"path": "build.sh", "content": "#!/bin/sh"},
			}
			forventet := []models.FileEntry{
				{Path: "Dockerfile", Content: "FROM alpine"},
				{Path: "build.sh", Content: "#!/bin/sh"},
			}
			Expect(fetcher.ConvertToFileEntries(input)).To(Equal(forventet))
		})
	})

	Describe("convertFiles", func() {
		It("skal konvertere nested map til map[string][]FileEntry", func() {
			input := map[string][]map[string]string{
				"dockerfile": {{"path": "Dockerfile", "content": "FROM alpine"}},
				"scripts":    {{"path": "build.sh", "content": "#!/bin/sh"}},
			}
			forventet := map[string][]models.FileEntry{
				"dockerfile": {{Path: "Dockerfile", Content: "FROM alpine"}},
				"scripts":    {{Path: "build.sh", Content: "#!/bin/sh"}},
			}
			Expect(fetcher.ConvertFiles(input)).To(Equal(forventet))
		})
	})

	Describe("buildRepoQuery", func() {
		It("skal bygge en GraphQL-spørring som inneholder riktig owner og repo", func() {
			query := fetcher.BuildRepoQuery("navikt", "arbeidsgiver")
			Expect(query).To(ContainSubstring(`repository(owner: "navikt", name: "arbeidsgiver")`))
			Expect(query).To(ContainSubstring("defaultBranchRef"))
		})
	})

	Describe("parseRepoData", func() {
		It("skal returnere strukturert RepoEntry fra minimal GraphQL-respons", func() {
			data := map[string]interface{}{
				"repository": map[string]interface{}{
					"languages": map[string]interface{}{
						"edges": []interface{}{
							map[string]interface{}{
								"size": float64(100),
								"node": map[string]interface{}{"name": "Go"},
							},
						},
					},
					"README":     map[string]interface{}{"text": "Hello world"},
					"SECURITY":   map[string]interface{}{},
					"dependabot": nil,
					"codeql":     map[string]interface{}{},
				},
			}
			base := models.RepoMeta{Name: "arbeidsgiver"}
			entry := fetcher.ParseRepoData(data, base)

			Expect(entry).NotTo(BeNil())
			Expect(entry.Repo.Name).To(Equal("arbeidsgiver"))
			Expect(entry.Repo.Readme).To(Equal("Hello world"))
			Expect(entry.Languages["Go"]).To(Equal(100))
			Expect(entry.Repo.Security["has_security_md"]).To(BeTrue())
			Expect(entry.Repo.Security["has_dependabot"]).To(BeFalse())
		})
	})

	Describe("extractLanguages", func() {
		It("skal håndtere både gyldige og ugyldige strukturer", func() {
			testcases := map[string]struct {
				data map[string]interface{}
				want map[string]int
			}{
				"gyldige språk": {
					data: map[string]interface{}{
						"languages": map[string]interface{}{
							"edges": []interface{}{
								map[string]interface{}{
									"size": float64(1234),
									"node": map[string]interface{}{"name": "Go"},
								},
								map[string]interface{}{
									"size": float64(567),
									"node": map[string]interface{}{"name": "Python"},
								},
							},
						},
					},
					want: map[string]int{"Go": 1234, "Python": 567},
				},
				"mangler node": {
					data: map[string]interface{}{
						"languages": map[string]interface{}{
							"edges": []interface{}{
								map[string]interface{}{"size": float64(100)},
							},
						},
					},
					want: map[string]int{},
				},
				"edge er ikke et map": {
					data: map[string]interface{}{
						"languages": map[string]interface{}{
							"edges": []interface{}{
								"not-a-map",
								map[string]interface{}{
									"size": float64(200),
									"node": map[string]interface{}{"name": "Java"},
								},
							},
						},
					},
					want: map[string]int{"Java": 200},
				},
			}

			for navn, tc := range testcases {
				got := fetcher.ExtractLanguages(tc.data)
				Expect(got).To(Equal(tc.want), "feilet for test: %s", navn)
			}
		})
	})

	Describe("extractCI", func() {
		It("skal hente ut CI-workflows med korrekt filsti og innhold", func() {
			data := map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"name": "build.yml",
							"object": map[string]interface{}{
								"text": "workflow-innhold",
							},
						},
					},
				},
			}
			got := fetcher.ExtractCI(data)
			Expect(got).To(HaveLen(1))
			Expect(got[0].Path).To(Equal(".github/workflows/build.yml"))
			Expect(got[0].Content).To(Equal("workflow-innhold"))
		})
		It("skal ignorere CI-entries som ikke er maps", func() {
			data := map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						"bare en streng",
						42,
						true,
						nil,
						map[string]interface{}{ // eneste gyldige entry
							"name": "bygge.yml",
							"object": map[string]interface{}{
								"text": "CI workflow",
							},
						},
					},
				},
			}
			got := fetcher.ExtractCI(data)
			Expect(got).To(HaveLen(1))
			Expect(got[0].Path).To(Equal(".github/workflows/bygge.yml"))
			Expect(got[0].Content).To(Equal("CI workflow"))
		})
	})

	Describe("extractReadme", func() {
		It("skal returnere README-tekst hvis den finnes", func() {
			Expect(fetcher.ExtractReadme(map[string]interface{}{
				"README": map[string]interface{}{"text": "Min README"},
			})).To(Equal("Min README"))

			Expect(fetcher.ExtractReadme(map[string]interface{}{})).To(Equal(""))
			Expect(fetcher.ExtractReadme(map[string]interface{}{
				"README": map[string]interface{}{},
			})).To(Equal(""))
		})
	})

	Describe("extractSecurity", func() {
		It("skal detektere sikkerhetsmetadata fra GraphQL-responsen", func() {
			data := map[string]interface{}{
				"SECURITY":   map[string]interface{}{},
				"dependabot": nil,
				"codeql":     map[string]interface{}{},
			}
			got := fetcher.ExtractSecurity(data)
			Expect(got["has_security_md"]).To(BeTrue())
			Expect(got["has_dependabot"]).To(BeFalse())
			Expect(got["has_codeql"]).To(BeTrue())
		})
	})

	Describe("extractFiles", func() {
		It("skal hente ut kun gyldige Dockerfile-objekter med innhold", func() {
			data := map[string]interface{}{
				"dependencies": map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"name": "Dockerfile",
							"object": map[string]interface{}{
								"text": "FROM alpine",
							},
						},
						map[string]interface{}{
							"name": "README.md",
							"object": map[string]interface{}{
								"text": "irrelevant",
							},
						},
						map[string]interface{}{
							"name":   "Dockerfile.empty",
							"object": map[string]interface{}{},
						},
						"not-a-map",
					},
				},
			}
			got := fetcher.ExtractFiles(data)
			Expect(got).To(HaveKey("dockerfile"))
			Expect(got["dockerfile"]).To(HaveLen(1))
			Expect(got["dockerfile"][0].Path).To(Equal("Dockerfile"))
			Expect(got["dockerfile"][0].Content).To(Equal("FROM alpine"))
		})
	})
})

var _ = Describe("doRequestWithRateLimit", func() {
	var originalClient *http.Client

	BeforeEach(func() {
		originalClient = fetcher.HttpClient
	})

	AfterEach(func() {
		fetcher.HttpClient = originalClient
	})

	It("skal håndtere rate limit og retry riktig", func() {
		callCount := 0
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Simuler at vi har truffet rate limit
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprint(time.Now().Add(50*time.Millisecond).Unix()))
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintln(w, `{}`)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"message": "ok"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()
		ctx := context.Background()
		var result struct{ Message string }
		err := fetcher.DoRequestWithRateLimit(ctx, "GET", ts.URL, "dummy-token", nil, &result)
		Expect(err).To(BeNil())
		Expect(result.Message).To(Equal("ok"))
		Expect(callCount).To(BeNumerically(">=", 2))
	})

	It("skal sette Content-Type header for POST", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal("POST"))
			Expect(r.Header.Get("Content-Type")).To(Equal("application/json"))
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintln(w, `{"message": "ok"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()
		ctx := context.Background()

		var result struct{ Message string }
		err := fetcher.DoRequestWithRateLimit(ctx, "POST", ts.URL, "token", []byte(`{}`), &result)
		Expect(err).To(BeNil())
		Expect(result.Message).To(Equal("ok"))
	})

	It("skal feile på ugyldig URL (DNS-feil)", func() {
		var result any
		ctx := context.Background()
		err := fetcher.DoRequestWithRateLimit(ctx, "GET", "http://invalid-url", "token", nil, &result)
		Expect(err).To(HaveOccurred())
	})

	It("skal feile på ugyldig request-format (syntax)", func() {
		var result any
		ctx := context.Background()
		err := fetcher.DoRequestWithRateLimit(ctx, "GET", ":", "token", nil, &result)
		Expect(err).To(HaveOccurred())
	})

	It("skal returnere feil hvis GitHub API svarer med status != 2xx", func() {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden) // 403
			_, _ = fmt.Fprint(w, `{"message":"access denied"}`)
		}))
		defer ts.Close()

		fetcher.HttpClient = ts.Client()
		ctx := context.Background()

		var result any
		err := fetcher.DoRequestWithRateLimit(ctx, "GET", ts.URL, "token", nil, &result)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("GitHub API-feil"))
		Expect(err.Error()).To(ContainSubstring("403"))
		Expect(err.Error()).To(ContainSubstring("access denied"))
	})
})
