package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/dbwriter"
	"github.com/jonmartinstorm/reposnusern/internal/storage"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
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

	var dump dbwriter.Dump
	if err := json.Unmarshal(data, &dump); err != nil {
		slog.Error("Kunne ikke parse JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("🚀 Importerer til PostgreSQL", "org", dump.Org, "antall_repos", len(dump.Repos))

	for i, entry := range dump.Repos {
		r := entry.Repo
		id := int64(r["id"].(float64))
		name := r["full_name"].(string)
		slog.Info("⏳ Behandler repo", "nummer", i+1, "navn", name)

		repo := storage.InsertRepoParams{
			ID:           id,
			Name:         r["name"].(string),
			FullName:     name,
			Description:  dbwriter.SafeString(r["description"]),
			Stars:        int64(r["stargazers_count"].(float64)),
			Forks:        int64(r["forks_count"].(float64)),
			Archived:     r["archived"].(bool),
			Private:      r["private"].(bool),
			IsFork:       r["fork"].(bool),
			Language:     dbwriter.SafeString(r["language"]),
			SizeMb:       float32(r["size"].(float64)) / 1024.0,
			UpdatedAt:    r["updated_at"].(string),
			PushedAt:     r["pushed_at"].(string),
			CreatedAt:    r["created_at"].(string),
			HtmlUrl:      r["html_url"].(string),
			Topics:       dbwriter.JoinStrings(r["topics"]),
			Visibility:   r["visibility"].(string),
			License:      dbwriter.ExtractLicense(r),
			OpenIssues:   int64(r["open_issues_count"].(float64)),
			LanguagesUrl: r["languages_url"].(string),
		}
		if err := queries.InsertRepo(ctx, repo); err != nil {
			slog.Error("Feil ved repo", "repo", name, "error", err)
			continue
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
			if dbwriter.IsDependencyFile(filetype) {
				for _, f := range files {
					if err := queries.InsertDependencyFile(ctx, storage.InsertDependencyFileParams{
						RepoID:  id,
						Path:    f.Path,
						Content: f.Content,
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

	slog.Info("✅ Ferdig importert!")
}
