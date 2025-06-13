package dbwriter

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/parser"
	"github.com/jonmartinstorm/reposnusern/internal/storage"
)

func SafeLicense(lic *struct{ SpdxID string }) string {
	if lic == nil {
		return ""
	}
	return lic.SpdxID
}

func SafeString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func ImportRepo(ctx context.Context, db *sql.DB, entry models.RepoEntry, index int) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start tx: %w", err)
	}

	queries := storage.New(tx)

	r := entry.Repo
	id := int64(r.ID)
	name := r.FullName

	repo := storage.InsertRepoParams{
		ID:           id,
		Name:         r.Name,
		FullName:     name,
		Description:  r.Description,
		Stars:        r.Stars,
		Forks:        r.Forks,
		Archived:     r.Archived,
		Private:      r.Private,
		IsFork:       r.IsFork,
		Language:     r.Language,
		SizeMb:       float32(r.Size) / 1024.0,
		UpdatedAt:    r.UpdatedAt,
		PushedAt:     r.PushedAt,
		CreatedAt:    r.CreatedAt,
		HtmlUrl:      r.HtmlUrl,
		Topics:       strings.Join(r.Topics, ","),
		Visibility:   r.Visibility,
		License:      SafeLicense((*struct{ SpdxID string })(r.License)),
		OpenIssues:   r.OpenIssues,
		LanguagesUrl: r.LanguagesURL,
	}

	if err := queries.InsertRepo(ctx, repo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("InsertRepo feilet: %v (rollback feilet: %w)", err, rbErr)
		}
		return fmt.Errorf("InsertRepo feilet: %w", err)
	}

	insertLanguages(ctx, queries, id, name, entry.Languages)
	insertDockerfiles(ctx, queries, id, name, entry.Files)
	insertCIConfig(ctx, queries, id, name, entry.CIConfig)
	insertReadme(ctx, queries, id, name, entry.Readme)
	insertSecurityFeatures(ctx, queries, id, name, entry.Security)
	insertSBOMPackagesGithub(ctx, queries, id, name, entry.SBOM)

	if err := tx.Commit(); err != nil {
		slog.Error("Commit-feil – ruller tilbake", "repo", name, "error", err)
		return fmt.Errorf("commit failed: %w", err)
	}

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
			slog.Warn("Språkfeil", "repo", name, "language", lang, "error", err)
		}
	}
}

func insertDockerfiles(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	files map[string][]models.FileEntry,
) {
	for filetype, fileEntries := range files {
		if !strings.HasPrefix(strings.ToLower(filetype), "dockerfile") {
			continue
		}
		for _, f := range fileEntries {
			dockerfileID, err := queries.InsertDockerfile(ctx, storage.InsertDockerfileParams{
				RepoID:   repoID,
				FullName: name,
				Path:     f.Path,
				Content:  f.Content,
			})
			if err != nil {
				slog.Warn("Dockerfile-feil", "repo", name, "fil", f.Path, "error", err)
				continue
			}

			features := parser.ParseDockerfile(f.Content)

			err = queries.InsertDockerfileFeatures(ctx, storage.InsertDockerfileFeaturesParams{
				DockerfileID:         dockerfileID,
				BaseImage:            sql.NullString{String: features.BaseImage, Valid: features.BaseImage != ""},
				BaseTag:              sql.NullString{String: features.BaseTag, Valid: features.BaseTag != ""},
				UsesLatestTag:        sql.NullBool{Bool: features.UsesLatestTag, Valid: true},
				HasUserInstruction:   sql.NullBool{Bool: features.HasUserInstruction, Valid: true},
				HasCopySensitive:     sql.NullBool{Bool: features.HasCopySensitive, Valid: true},
				HasPackageInstalls:   sql.NullBool{Bool: features.HasPackageInstalls, Valid: true},
				UsesMultistage:       sql.NullBool{Bool: features.UsesMultistage, Valid: true},
				HasHealthcheck:       sql.NullBool{Bool: features.HasHealthcheck, Valid: true},
				UsesAddInstruction:   sql.NullBool{Bool: features.UsesAddInstruction, Valid: true},
				HasLabelMetadata:     sql.NullBool{Bool: features.HasLabelMetadata, Valid: true},
				HasExpose:            sql.NullBool{Bool: features.HasExpose, Valid: true},
				HasEntrypointOrCmd:   sql.NullBool{Bool: features.HasEntrypointOrCmd, Valid: true},
				InstallsCurlOrWget:   sql.NullBool{Bool: features.InstallsCurlOrWget, Valid: true},
				InstallsBuildTools:   sql.NullBool{Bool: features.InstallsBuildTools, Valid: true},
				HasAptGetClean:       sql.NullBool{Bool: features.HasAptGetClean, Valid: true},
				WorldWritable:        sql.NullBool{Bool: features.WorldWritable, Valid: true},
				HasSecretsInEnvOrArg: sql.NullBool{Bool: features.HasSecretsInEnvOrArg, Valid: true},
			})
			if err != nil {
				slog.Warn("Dockerfile-feature-feil", "repo", name, "fil", f.Path, "error", err)
			}
		}
	}
}

func insertCIConfig(
	ctx context.Context,
	queries *storage.Queries,
	repoID int64,
	name string,
	files []models.FileEntry,
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
		slog.Warn("Ugyldig sbom-format", "repo", name)
		return
	}

	packages, ok := sbomInner["packages"].([]interface{})
	if !ok {
		slog.Warn("Ingen pakker i sbom", "repo", name)
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
			slog.Warn("SBOM-insert-feil", "repo", name, "package", nameVal, "error", err)
		}
	}
}
