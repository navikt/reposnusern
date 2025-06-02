package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	_ "github.com/lib/pq"

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

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		slog.Error("❌ POSTGRES_DSN ikke satt")
		os.Exit(1)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("Kunne ikke koble til Postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	queries := storage.New(db)

	data, err := os.ReadFile("data/navikt_analysis_data.json")
	if err != nil {
		slog.Error("Kunne ikke lese JSON", "error", err)
		os.Exit(1)
	}

	var dump Dump
	if err := json.Unmarshal(data, &dump); err != nil {
		slog.Error("Kunne ikke parse JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("🚀 Importerer til PostgreSQL", "antall_repos", len(dump.Repos))

	for _, entry := range dump.Repos {
		r := entry.Repo
		id := int64(r["id"].(float64))

		repo := storage.InsertRepoParams{
			ID:           id,
			Name:         r["name"].(string),
			FullName:     r["full_name"].(string),
			Description:  safeString(r["description"]),
			Stars:        int64(r["stargazers_count"].(float64)),
			Forks:        int64(r["forks_count"].(float64)),
			Archived:     r["archived"].(bool),
			Private:      r["private"].(bool),
			IsFork:       r["fork"].(bool),
			Language:     safeString(r["language"]),
			SizeMb:       float64(r["size"].(float64)) / 1024.0,
			UpdatedAt:    r["updated_at"].(string),
			PushedAt:     r["pushed_at"].(string),
			CreatedAt:    r["created_at"].(string),
			HtmlUrl:      r["html_url"].(string),
			Topics:       joinStrings(r["topics"]),
			Visibility:   r["visibility"].(string),
			License:      extractLicense(r),
			OpenIssues:   int64(r["open_issues_count"].(float64)),
			LanguagesUrl: r["languages_url"].(string),
		}
		if err := queries.InsertRepo(ctx, repo); err != nil {
			slog.Error("Feil ved repo", "repo", repo.FullName, "error", err)
			continue
		}

		for lang, size := range entry.Languages {
			err := queries.InsertRepoLanguage(ctx, storage.InsertRepoLanguageParams{
				RepoID:   id,
				Language: lang,
				Bytes:    int64(size),
			})
			if err != nil {
				slog.Warn("Språkfeil", "repo", repo.FullName, "language", lang, "error", err)
			}
		}

		for name, files := range entry.Files {
			if isDependencyFile(name) {
				for _, f := range files {
					_ = queries.InsertDependencyFile(ctx, storage.InsertDependencyFileParams{
						RepoID:  id,
						Path:    f.Path,
						Content: f.Content,
					})
				}
			}
			if strings.HasPrefix(strings.ToLower(name), "dockerfile") {
				for _, f := range files {
					_ = queries.InsertDockerfile(ctx, storage.InsertDockerfileParams{
						RepoID:   id,
						FullName: repo.FullName,
						Path:     f.Path,
						Content:  f.Content,
					})
				}
			}
		}

		for _, f := range entry.CIConfig {
			_ = queries.InsertCIConfig(ctx, storage.InsertCIConfigParams{
				RepoID:  id,
				Path:    f.Path,
				Content: f.Content,
			})
		}

		if entry.Readme != "" {
			_ = queries.InsertReadme(ctx, storage.InsertReadmeParams{
				RepoID:  id,
				Content: entry.Readme,
			})
		}

		_ = queries.InsertSecurityFeatures(ctx, storage.InsertSecurityFeaturesParams{
			RepoID:        id,
			HasSecurityMd: entry.Security["has_security_md"],
			HasDependabot: entry.Security["has_dependabot"],
			HasCodeql:     entry.Security["has_codeql"],
		})
	}

	slog.Info("✅ Ferdig importert!")
}

func safeString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func joinStrings(arr interface{}) string {
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

func extractLicense(r map[string]interface{}) string {
	if r["license"] == nil {
		return ""
	}
	return r["license"].(map[string]interface{})["spdx_id"].(string)
}

func isDependencyFile(name string) bool {
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
