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
	SBOM      map[string]interface{} `json:"sbom"`
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

	for i, entry := range dump.Repos {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("start tx: %w", err)
		}

		queries := storage.New(tx)
		if err := importRepo(ctx, queries, entry, i); err != nil {
			tx.Rollback()
			return fmt.Errorf("import repo: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit failed: %w", err)
		}
	}
	return nil
}

func importRepo(ctx context.Context, queries *storage.Queries, entry RepoEntry, index int) error {
	r := entry.Repo
	id := int64(r["id"].(float64))
	name := r["full_name"].(string)
	slog.Info("⏳ Behandler repo", "nummer", index+1, "navn", name)

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

	insertLanguages(ctx, queries, id, name, entry.Languages)
	insertDependencyFiles(ctx, queries, id, name, entry.Files)
	insertDockerfiles(ctx, queries, id, name, entry.Files)
	insertCIConfig(ctx, queries, id, name, entry.CIConfig)
	insertReadme(ctx, queries, id, name, entry.Readme)
	insertSecurityFeatures(ctx, queries, id, name, entry.Security)
	insertSBOMPackagesGithub(ctx, queries, id, name, entry.SBOM)

	return nil
}

func insertLanguages(ctx context.Context, queries *storage.Queries, repoID int64, name string, langs map[string]int) {
	for lang, size := range langs {
		err := queries.InsertRepoLanguage(ctx, storage.InsertRepoLanguageParams{
			RepoID:   repoID,
			Language: lang,
			Bytes:    int64(size),
		})
		if err != nil {
			slog.Warn("❗️Språkfeil", "repo", name, "language", lang, "error", err)
		}
	}
}

func insertDependencyFiles(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	files map[string][]FileEntry,
) {
	for filetype, fileEntries := range files {
		if !IsDependencyFile(filetype) {
			continue
		}
		for _, f := range fileEntries {
			if err := queries.InsertDependencyFile(ctx, storage.InsertDependencyFileParams{
				RepoID: repoID,
				Path:   f.Path,
			}); err != nil {
				slog.Warn("Dependency-feil", "repo", name, "fil", f.Path, "error", err)
			}
		}
	}
}

func insertDockerfiles(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	files map[string][]FileEntry,
) {
	for filetype, fileEntries := range files {
		if !strings.HasPrefix(strings.ToLower(filetype), "dockerfile") {
			continue
		}
		for _, f := range fileEntries {
			if err := queries.InsertDockerfile(ctx, storage.InsertDockerfileParams{
				RepoID:   repoID,
				FullName: name,
				Path:     f.Path,
				Content:  f.Content,
			}); err != nil {
				slog.Warn("Dockerfile-feil", "repo", name, "fil", f.Path, "error", err)
			}
		}
	}
}

func insertCIConfig(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	files []FileEntry,
) {
	for _, f := range files {
		if err := queries.InsertCIConfig(ctx, storage.InsertCIConfigParams{
			RepoID:  repoID,
			Path:    f.Path,
			Content: f.Content,
		}); err != nil {
			slog.Warn("CI-feil", "repo", name, "fil", f.Path, "error", err)
		}
	}
}

// func insertDockerfiles(...)

func insertReadme(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	content string,
) {
	if content == "" {
		return
	}

	if err := queries.InsertReadme(ctx, storage.InsertReadmeParams{
		RepoID:  repoID,
		Content: content,
	}); err != nil {
		slog.Warn("README-feil", "repo", name, "error", err)
	}
}

func insertSecurityFeatures(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	security map[string]bool,
) {
	if err := queries.InsertSecurityFeatures(ctx, storage.InsertSecurityFeaturesParams{
		RepoID:        repoID,
		HasSecurityMd: security["has_security_md"],
		HasDependabot: security["has_dependabot"],
		HasCodeql:     security["has_codeql"],
	}); err != nil {
		slog.Warn("Security-feil", "repo", name, "error", err)
	}
}

func insertSBOMPackagesGithub(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	sbomRaw map[string]interface{},
) {
	if sbomRaw == nil {
		return
	}

	sbomInner, ok := sbomRaw["sbom"].(map[string]interface{})
	if !ok {
		slog.Warn("❗️Ugyldig sbom-format", "repo", name)
		return
	}

	packages, ok := sbomInner["packages"].([]interface{})
	if !ok {
		slog.Warn("❗️Ingen pakker i sbom", "repo", name)
		return
	}

	for _, p := range packages {
		pkg, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		nameVal := SafeString(pkg["name"])
		version := SafeString(pkg["versionInfo"])
		license := SafeString(pkg["licenseConcluded"])

		// Prøv å hente ut PURL (Package URL) fra externalRefs
		var purl string
		if refs, ok := pkg["externalRefs"].([]interface{}); ok {
			for _, ref := range refs {
				refMap, ok := ref.(map[string]interface{})
				if !ok {
					continue
				}
				if refMap["referenceType"] == "purl" {
					purl = SafeString(refMap["referenceLocator"])
					break
				}
			}
		}

		err := queries.InsertGithubSBOM(ctx, storage.InsertGithubSBOMParams{
			RepoID:  repoID,
			Name:    nameVal,
			Version: sql.NullString{String: version, Valid: version != ""},
			License: sql.NullString{String: license, Valid: license != ""},
			Purl:    sql.NullString{String: purl, Valid: purl != ""},
		})
		if err != nil {
			slog.Warn("🚨 SBOM-insert-feil", "repo", name, "package", nameVal, "error", err)
		}
	}
}
