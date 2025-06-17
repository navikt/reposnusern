package bqwriter_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jonmartinstorm/reposnusern/internal/bqwriter"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

var _ = Describe("Mapping-funksjoner", func() {
	snapshot := time.Date(2025, 6, 17, 12, 0, 0, 0, time.UTC)

	entry := models.RepoEntry{
		Repo: models.RepoMeta{
			ID:           42,
			Name:         "repo",
			FullName:     "org/repo",
			Description:  "desc",
			Stars:        100,
			Forks:        10,
			Archived:     true,
			Private:      false,
			IsFork:       false,
			Language:     "Go",
			Size:         2048,
			UpdatedAt:    "2025-06-17T10:00:00Z",
			PushedAt:     "2025-06-16T10:00:00Z",
			CreatedAt:    "2025-01-01T10:00:00Z",
			HtmlUrl:      "https://github.com/org/repo",
			Topics:       []string{"go", "example"},
			Visibility:   "public",
			License:      &models.License{SpdxID: "MIT"},
			OpenIssues:   5,
			LanguagesURL: "https://github.com/org/repo/languages",
			Readme:       "README content",
			Security: map[string]bool{
				"has_security_md": true,
				"has_dependabot":  true,
				"has_codeql":      false,
			},
		},
		Languages: map[string]int{
			"Go":    1000,
			"Shell": 500,
		},
		Files: map[string][]models.FileEntry{
			"dockerfile": {
				{
					Path:    "Dockerfile",
					Content: "FROM alpine",
				},
			},
		},
		CIConfig: []models.FileEntry{
			{
				Path:    ".github/workflows/ci.yml",
				Content: "name: CI",
			},
		},
		SBOM: map[string]interface{}{
			"sbom": map[string]interface{}{
				"packages": []interface{}{
					map[string]interface{}{
						"name":             "pkg",
						"versionInfo":      "1.0",
						"licenseConcluded": "MIT",
						"externalRefs": []interface{}{
							map[string]interface{}{
								"referenceType":    "purl",
								"referenceLocator": "pkg:golang/pkg@1.0",
							},
						},
					},
				},
			},
		},
	}

	It("konverterer til BGRepoEntry riktig", func() {
		bg := bqwriter.ConvertToBG(entry, snapshot)
		Expect(bg.RepoID).To(Equal(int64(42)))
		Expect(bg.Name).To(Equal("repo"))
		Expect(bg.Stars).To(Equal(int64(100)))
		Expect(bg.HasCodeQL).To(BeFalse())
	})

	It("konverterer språk til BGRepoLanguage", func() {
		langs := bqwriter.ConvertLanguages(entry, snapshot)
		Expect(langs).To(HaveLen(2))
		Expect(langs[0].RepoID).To(Equal(int64(42)))
	})

	It("konverterer dockerfile-features riktig", func() {
		features, _ := bqwriter.ConvertDockerfileFeatures(entry, snapshot)
		Expect(features).To(HaveLen(1))
		Expect(features[0].Content).To(ContainSubstring("FROM alpine"))
	})

	It("konverterer CI-configs riktig", func() {
		ci := bqwriter.ConvertCI(entry, snapshot)
		Expect(ci).To(HaveLen(1))
		Expect(ci[0].Path).To(Equal(".github/workflows/ci.yml"))
	})

	It("konverterer SBOM-pakker riktig", func() {
		pkgs := bqwriter.ConvertSBOMPackages(entry, snapshot)
		Expect(pkgs).To(HaveLen(1))
		Expect(pkgs[0].Name).To(Equal("pkg"))
		Expect(pkgs[0].PURL).To(Equal("pkg:golang/pkg@1.0"))
	})
})
