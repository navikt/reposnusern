package bqwriter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/parser"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

type BigQueryWriter struct {
	Client  *bigquery.Client
	Dataset string
}

func NewBigQueryWriter(ctx context.Context, cfg *config.Config) (*BigQueryWriter, error) {
	var client *bigquery.Client

	client, err := bigquery.NewClient(ctx, cfg.BQProjectID, option.WithCredentialsFile(cfg.BQCredentials))
	if err != nil {
		return nil, fmt.Errorf("kan ikke opprette BigQuery-klient: %w", err)
	}

	// Sørg for at hver tabell finnes
	tables := map[string]any{
		"repos":               BGRepoEntry{},
		"repo_languages":      BGRepoLanguage{},
		"dockerfile_features": BGDockerfileFeatures{},
		"dockerfile_stages":   BGDockerStageMeta{},
		"ci_config":           BGCIConfig{},
		"sbom_packages":       BGSBOMPackages{},
	}

	for tableName, schemaExample := range tables {
		if err := ensureTableExists(ctx, client, cfg.BQDataset, tableName, schemaExample); err != nil {
			return nil, fmt.Errorf("kunne ikke sikre tabell %s: %w", tableName, err)
		}
	}

	return &BigQueryWriter{
		Client:  client,
		Dataset: cfg.BQDataset,
	}, nil
}

func (w *BigQueryWriter) ImportRepo(ctx context.Context, entry models.RepoEntry, snapshot time.Time) error {
	repo := ConvertToBG(entry, snapshot)
	langs := ConvertLanguages(entry, snapshot)
	dockerfileFeatures, dockerfileStages := ConvertDockerfileFeatures(entry, snapshot)
	ciconfig := ConvertCI(entry, snapshot)
	sbom := ConvertSBOMPackages(entry, snapshot)

	if err := insert(ctx, w.Client, w.Dataset, "repos", []BGRepoEntry{repo}); err != nil {
		return fmt.Errorf("repos insert failed: %w", err)
	}
	if err := insert(ctx, w.Client, w.Dataset, "repo_languages", langs); err != nil {
		return fmt.Errorf("repo_languages insert failed: %w", err)
	}
	if err := insert(ctx, w.Client, w.Dataset, "dockerfile_features", dockerfileFeatures); err != nil {
		return fmt.Errorf("dockerfile_features insert failed: %w", err)
	}
	if err := insert(ctx, w.Client, w.Dataset, "dockerfile_stages", dockerfileStages); err != nil {
		return fmt.Errorf("dockerfile_stagesinsert failed: %w", err)
	}
	if err := insert(ctx, w.Client, w.Dataset, "ci_config", ciconfig); err != nil {
		return fmt.Errorf("ci_config insert failed: %w", err)
	}
	if err := insert(ctx, w.Client, w.Dataset, "sbom_packages", sbom); err != nil {
		return fmt.Errorf("sbom insert failed: %w", err)
	}

	return nil
}

func insert[T any](ctx context.Context, client *bigquery.Client, dataset, table string, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	inserter := client.Dataset(dataset).Table(table).Inserter()
	return inserter.Put(ctx, rows)
}

// ==== Data-strukturer ====

type BGRepoEntry struct {
	RepoID        int64     `bigquery:"repo_id"`
	WhenCollected time.Time `bigquery:"when_collected"`
	Name          string    `bigquery:"name"`
	FullName      string    `bigquery:"full_name"`
	Description   string    `bigquery:"description"`
	Stars         int64     `bigquery:"stars"`
	Forks         int64     `bigquery:"forks"`
	Archived      bool      `bigquery:"archived"`
	Private       bool      `bigquery:"private"`
	IsFork        bool      `bigquery:"is_fork"`
	Language      string    `bigquery:"language"`
	SizeMB        float32   `bigquery:"size_mb"`
	UpdatedAt     time.Time `bigquery:"updated_at"`
	PushedAt      time.Time `bigquery:"pushed_at"`
	CreatedAt     time.Time `bigquery:"created_at"`
	HtmlUrl       string    `bigquery:"html_url"`
	Topics        string    `bigquery:"topics"`
	Visibility    string    `bigquery:"visibility"`
	License       string    `bigquery:"license"`
	OpenIssues    int64     `bigquery:"open_issues"`
	LanguagesUrl  string    `bigquery:"languages_url"`
	ReadmeContent string    `bigquery:"readme_content"`
	HasSecurityMD bool      `bigquery:"has_security_md"`
	HasDependabot bool      `bigquery:"has_dependabot"`
	HasCodeQL     bool      `bigquery:"has_codeql"`
}

type BGRepoLanguage struct {
	RepoID        int64     `bigquery:"repo_id"`
	WhenCollected time.Time `bigquery:"when_collected"`
	Language      string    `bigquery:"language"`
	Bytes         int64     `bigquery:"bytes"`
}

type BGDockerfileFeatures struct {
	RepoID               int64     `bigquery:"repo_id"`
	WhenCollected        time.Time `bigquery:"when_collected"`
	FileType             string    `bigquery:"file_type"`
	Content              string    `bigquery:"content"`
	Path                 string    `bigquery:"path"`
	UsesLatestTag        bool      `bigquery:"uses_latest_tag"`
	HasUserInstruction   bool      `bigquery:"has_user_instruction"`
	HasCopySensitive     bool      `bigquery:"has_copy_sensitive"`
	HasPackageInstalls   bool      `bigquery:"has_package_installs"`
	UsesMultistage       bool      `bigquery:"uses_multistage"`
	HasHealthcheck       bool      `bigquery:"has_healthcheck"`
	UsesAddInstruction   bool      `bigquery:"uses_add_instruction"`
	HasLabelMetadata     bool      `bigquery:"has_label_metadata"`
	HasExpose            bool      `bigquery:"has_expose"`
	HasEntrypointOrCmd   bool      `bigquery:"has_entrypoint_or_cmd"`
	InstallsCurlOrWget   bool      `bigquery:"installs_curl_or_wget"`
	InstallsBuildTools   bool      `bigquery:"installs_build_tools"`
	HasAptGetClean       bool      `bigquery:"has_apt_get_clean"`
	WorldWritable        bool      `bigquery:"world_writable"`
	HasSecretsInEnvOrArg bool      `bigquery:"has_secrets_in_env_or_arg"`
}

type BGDockerStageMeta struct {
	RepoID        int64     `bigquery:"repo_id"`
	WhenCollected time.Time `bigquery:"when_collected"`
	Path          string    `bigquery:"path"`
	StageIndex    int       `bigquery:"stage_index"`
	BaseImage     string    `bigquery:"base_image"`
	BaseTag       string    `bigquery:"base_tag"`
}

type BGCIConfig struct {
	RepoID        int64     `bigquery:"repo_id"`
	WhenCollected time.Time `bigquery:"when_collected"`
	Path          string    `bigquery:"path"`
	Content       string    `bigquery:"content"`
}

type BGSBOMPackages struct {
	RepoID        int64     `bigquery:"repo_id"`
	WhenCollected time.Time `bigquery:"when_collected"`
	Name          string    `bigquery:"name"`
	Version       string    `bigquery:"version"`
	License       string    `bigquery:"license"`
	PURL          string    `bigquery:"purl"`
}

// ==== Mapping-funksjoner ====

func ConvertToBG(entry models.RepoEntry, snapshot time.Time) BGRepoEntry {
	r := entry.Repo
	return BGRepoEntry{
		RepoID:        r.ID,
		WhenCollected: snapshot,
		Name:          r.Name,
		FullName:      r.FullName,
		Description:   r.Description,
		Stars:         r.Stars,
		Forks:         r.Forks,
		Archived:      r.Archived,
		Private:       r.Private,
		IsFork:        r.IsFork,
		Language:      r.Language,
		SizeMB:        float32(r.Size) / 1024.0,
		UpdatedAt:     parseTime(r.UpdatedAt),
		PushedAt:      parseTime(r.PushedAt),
		CreatedAt:     parseTime(r.CreatedAt),
		HtmlUrl:       r.HtmlUrl,
		Topics:        strings.Join(r.Topics, ","),
		Visibility:    r.Visibility,
		License:       safeLicense(r.License),
		OpenIssues:    r.OpenIssues,
		LanguagesUrl:  r.LanguagesURL,
		ReadmeContent: r.Readme,
		HasSecurityMD: r.Security["has_security_md"],
		HasDependabot: r.Security["has_dependabot"],
		HasCodeQL:     r.Security["has_codeql"],
	}
}

func ConvertLanguages(entry models.RepoEntry, snapshot time.Time) []BGRepoLanguage {
	var result []BGRepoLanguage
	for lang, size := range entry.Languages {
		result = append(result, BGRepoLanguage{
			RepoID:        entry.Repo.ID,
			WhenCollected: snapshot,
			Language:      lang,
			Bytes:         int64(size),
		})
	}
	return result
}

func ConvertDockerfileFeatures(entry models.RepoEntry, snapshot time.Time) ([]BGDockerfileFeatures, []BGDockerStageMeta) {
	var dff []BGDockerfileFeatures
	var dsm []BGDockerStageMeta

	for typ, list := range entry.Files {
		if !strings.HasPrefix(strings.ToLower(typ), "dockerfile") {
			continue
		}
		for _, f := range list {
			features, stages := parser.ParseDockerfile(f.Content)

			dff = append(dff, BGDockerfileFeatures{
				RepoID:               entry.Repo.ID,
				WhenCollected:        snapshot,
				FileType:             typ,
				Path:                 f.Path,
				Content:              f.Content,
				UsesLatestTag:        features.UsesLatestTag,
				HasUserInstruction:   features.HasUserInstruction,
				HasCopySensitive:     features.HasCopySensitive,
				HasPackageInstalls:   features.HasPackageInstalls,
				UsesMultistage:       features.UsesMultistage,
				HasHealthcheck:       features.HasHealthcheck,
				UsesAddInstruction:   features.UsesAddInstruction,
				HasLabelMetadata:     features.HasLabelMetadata,
				HasExpose:            features.HasExpose,
				HasEntrypointOrCmd:   features.HasEntrypointOrCmd,
				InstallsCurlOrWget:   features.InstallsCurlOrWget,
				InstallsBuildTools:   features.InstallsBuildTools,
				HasAptGetClean:       features.HasAptGetClean,
				WorldWritable:        features.WorldWritable,
				HasSecretsInEnvOrArg: features.HasSecretsInEnvOrArg,
			})

			for _, stage := range stages {
				dsm = append(dsm, BGDockerStageMeta{
					RepoID:        entry.Repo.ID,
					WhenCollected: snapshot,
					Path:          f.Path,
					StageIndex:    stage.StageIndex,
					BaseImage:     stage.BaseImage,
					BaseTag:       stage.BaseTag,
				})
			}
		}
	}

	return dff, dsm
}

func ConvertCI(entry models.RepoEntry, snapshot time.Time) []BGCIConfig {
	var result []BGCIConfig
	for _, f := range entry.CIConfig {
		result = append(result, BGCIConfig{
			RepoID:        entry.Repo.ID,
			WhenCollected: snapshot,
			Path:          f.Path,
			Content:       f.Content,
		})
	}
	return result
}

func ConvertSBOMPackages(entry models.RepoEntry, snapshot time.Time) []BGSBOMPackages {
	raw := entry.SBOM
	var result []BGSBOMPackages

	sbomInner, ok := raw["sbom"].(map[string]interface{})
	if !ok {
		return result
	}
	pkgs, ok := sbomInner["packages"].([]interface{})
	if !ok {
		return result
	}

	for _, p := range pkgs {
		pkg, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, BGSBOMPackages{
			RepoID:        entry.Repo.ID,
			WhenCollected: snapshot,
			Name:          safeString(pkg["name"]),
			Version:       safeString(pkg["versionInfo"]),
			License:       safeString(pkg["licenseConcluded"]),
			PURL:          extractPURL(pkg),
		})
	}

	return result
}

// ==== Hjelpefunksjoner ====

func safeLicense(lic *models.License) string {
	if lic == nil {
		return ""
	}
	return lic.SpdxID
}

func safeString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func extractPURL(pkg map[string]interface{}) string {
	refs, ok := pkg["externalRefs"].([]interface{})
	if !ok {
		return ""
	}
	for _, ref := range refs {
		refMap, ok := ref.(map[string]interface{})
		if ok && refMap["referenceType"] == "purl" {
			return safeString(refMap["referenceLocator"])
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func ensureTableExists(ctx context.Context, client *bigquery.Client, dataset, table string, exampleStruct any) error {
	tbl := client.Dataset(dataset).Table(table)
	_, err := tbl.Metadata(ctx)
	if err == nil {
		return nil // tabellen finnes
	}

	if gErr, ok := err.(*googleapi.Error); !ok || gErr.Code != 404 {
		return fmt.Errorf("feil ved henting av tabell-metadata: %w", err)
	}

	schema, err := bigquery.InferSchema(exampleStruct)
	if err != nil {
		return fmt.Errorf("klarte ikke å generere schema for %s: %w", table, err)
	}

	if err := tbl.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil {
		return fmt.Errorf("klarte ikke å opprette tabell %s: %w", table, err)
	}

	return nil
}
