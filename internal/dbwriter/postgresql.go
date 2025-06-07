package dbwriter

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/storage"
)

type FileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RepoEntry struct {
	Repo      map[string]interface{} `json:"repo"`
	Languages map[string]int         `json:"languages"`
	Files     map[string][]FileEntry `json:"files"`
	CIConfig  []FileEntry            `json:"ci_config"`
	Readme    string                 `json:"readme"`
	Security  map[string]bool        `json:"security"`
}

type Dump struct {
	Org   string      `json:"org"`
	Repos []RepoEntry `json:"repos"`
}

func SafeString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func JoinStrings(arr interface{}) string {
	if arr == nil {
		return ""
	}
	raw := arr.([]interface{})
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, v.(string))
	}
	return strings.Join(out, ",")
}

func ExtractLicense(r map[string]interface{}) string {
	if r["license"] == nil {
		return ""
	}
	return r["license"].(map[string]interface{})["spdx_id"].(string)
}

func IsDependencyFile(name string) bool {
	known := []string{
		"package.json", "pom.xml", "build.gradle", "build.gradle.kts",
		"go.mod", "cargo.toml", "requirements.txt", "pyproject.toml",
		"composer.json", ".csproj", "gemfile", "gemfile.lock",
		"yarn.lock", "pnpm-lock.yaml", "package-lock.json",
	}
	name = strings.ToLower(name)
	for _, k := range known {
		if k == name {
			return true
		}
	}
	return false
}

func ImportToPostgreSQLDB(dump Dump, db *sql.DB) error {
	ctx := context.Background()
	queries := storage.New(db)

	for i, entry := range dump.Repos {
		r := entry.Repo
		id := int64(r["id"].(float64))
		name := r["full_name"].(string)
		slog.Info("⏳ Behandler repo", "nummer", i+1, "navn", name)

		repo := storage.InsertRepoParams{
			ID:           id,
			Name:         r["name"].(string),
			FullName:     name,
			Description:  SafeString(r["description"]),
			Stars:        int64(r["stargazers_count"].(float64)),
			Forks:        int64(r["forks_count"].(float64)),
			Archived:     r["archived"].(bool),
			Private:      r["private"].(bool),
			IsFork:       r["fork"].(bool),
			Language:     SafeString(r["language"]),
			SizeMb:       float32(r["size"].(float64)) / 1024.0,
			UpdatedAt:    r["updated_at"].(string),
			PushedAt:     r["pushed_at"].(string),
			CreatedAt:    r["created_at"].(string),
			HtmlUrl:      r["html_url"].(string),
			Topics:       JoinStrings(r["topics"]),
			Visibility:   r["visibility"].(string),
			License:      ExtractLicense(r),
			OpenIssues:   int64(r["open_issues_count"].(float64)),
			LanguagesUrl: r["languages_url"].(string),
		}
		if err := queries.InsertRepo(ctx, repo); err != nil {
			slog.Error("🚨 Feil ved InsertRepo – avbryter import", "repo", name, "error", err)
			return fmt.Errorf("insert repo failed: %w", err)
		}

		for lang, size := range entry.Languages {
			err := queries.InsertRepoLanguage(ctx, storage.InsertRepoLanguageParams{
				RepoID:   id,
				Language: lang,
				Bytes:    int64(size),
			})
			if err != nil {
				slog.Warn("❗️Språkfeil", "repo", name, "language", lang, "error", err)
			}
		}

		for filetype, files := range entry.Files {
			if IsDependencyFile(filetype) {
				for _, f := range files {
					if err := queries.InsertDependencyFile(ctx, storage.InsertDependencyFileParams{
						RepoID: id,
						Path:   f.Path,
					}); err != nil {
						slog.Warn("Dependency-feil", "repo", name, "fil", f.Path, "error", err)
					}
				}
			}
			if strings.HasPrefix(strings.ToLower(filetype), "dockerfile") {
				for _, f := range files {
					if err := queries.InsertDockerfile(ctx, storage.InsertDockerfileParams{
						RepoID:   id,
						FullName: name,
						Path:     f.Path,
						Content:  f.Content,
					}); err != nil {
						slog.Warn("Dockerfile-feil", "repo", name, "fil", f.Path, "error", err)
					}
				}
			}
		}

		for _, f := range entry.CIConfig {
			if err := queries.InsertCIConfig(ctx, storage.InsertCIConfigParams{
				RepoID:  id,
				Path:    f.Path,
				Content: f.Content,
			}); err != nil {
				slog.Warn("CI-feil", "repo", name, "fil", f.Path, "error", err)
			}
		}

		if entry.Readme != "" {
			if err := queries.InsertReadme(ctx, storage.InsertReadmeParams{
				RepoID:  id,
				Content: entry.Readme,
			}); err != nil {
				slog.Warn("README-feil", "repo", name, "error", err)
			}
		}

		if err := queries.InsertSecurityFeatures(ctx, storage.InsertSecurityFeaturesParams{
			RepoID:        id,
			HasSecurityMd: entry.Security["has_security_md"],
			HasDependabot: entry.Security["has_dependabot"],
			HasCodeql:     entry.Security["has_codeql"],
		}); err != nil {
			slog.Warn("Security-feil", "repo", name, "error", err)
		}
	}

	return nil
}
