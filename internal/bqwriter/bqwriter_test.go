package bqwriter_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jonmartinstorm/reposnusern/internal/bqwriter"
	"github.com/jonmartinstorm/reposnusern/internal/models"
)

var updateSchemaFile = flag.Bool("update-schema", false, "Regenerate schema/bigquery_schema.json")

func TestBQWriter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BQWriter Suite")
}

type fieldSpec struct {
	Name  string
	Type  string
	BQTag string
}

func extractFields(t reflect.Type) []fieldSpec {
	var fields []fieldSpec
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields = append(fields, fieldSpec{
			Name:  f.Name,
			Type:  f.Type.String(),
			BQTag: f.Tag.Get("bigquery"),
		})
	}
	return fields
}

var _ = Describe("BigQuery schema contract", func() {
	DescribeTable("struct fields must match expected schema exactly",
		func(structInstance any, expected []fieldSpec) {
			actual := extractFields(reflect.TypeOf(structInstance))
			Expect(actual).To(Equal(expected),
				"Schema mismatch! If this is intentional, update the expected fields in this test.")
		},

		Entry("BGRepoEntry", bqwriter.BGRepoEntry{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"Name", "string", "name"},
			{"FullName", "string", "full_name"},
			{"Description", "string", "description"},
			{"Stars", "int64", "stars"},
			{"Forks", "int64", "forks"},
			{"Archived", "bool", "archived"},
			{"Private", "bool", "private"},
			{"IsFork", "bool", "is_fork"},
			{"Language", "string", "language"},
			{"SizeMB", "float32", "size_mb"},
			{"UpdatedAt", "time.Time", "updated_at"},
			{"PushedAt", "time.Time", "pushed_at"},
			{"CreatedAt", "time.Time", "created_at"},
			{"HtmlUrl", "string", "html_url"},
			{"Topics", "string", "topics"},
			{"Visibility", "string", "visibility"},
			{"License", "string", "license"},
			{"OpenIssues", "int64", "open_issues"},
			{"LanguagesUrl", "string", "languages_url"},
			{"ReadmeContent", "string", "readme_content"},
			{"HasSecurityMD", "bool", "has_security_md"},
			{"HasDependabot", "bool", "has_dependabot"},
			{"HasCodeQL", "bool", "has_codeql"},
			{"HasCompleteLockfiles", "bool", "has_complete_lockfiles"},
			{"LockfilePairings", "string", "lockfile_pairings"},
			{"LockfilePairCount", "int", "lockfile_pair_count"},
		}),

		Entry("BGRepoLanguage", bqwriter.BGRepoLanguage{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"Language", "string", "language"},
			{"Bytes", "int64", "bytes"},
		}),

		Entry("BGDockerfileFeatures", bqwriter.BGDockerfileFeatures{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"FileType", "string", "file_type"},
			{"Content", "string", "content"},
			{"Path", "string", "path"},
			{"UsesLatestTag", "bool", "uses_latest_tag"},
			{"HasUserInstruction", "bool", "has_user_instruction"},
			{"HasCopySensitive", "bool", "has_copy_sensitive"},
			{"HasPackageInstalls", "bool", "has_package_installs"},
			{"UsesMultistage", "bool", "uses_multistage"},
			{"HasHealthcheck", "bool", "has_healthcheck"},
			{"UsesAddInstruction", "bool", "uses_add_instruction"},
			{"HasLabelMetadata", "bool", "has_label_metadata"},
			{"HasExpose", "bool", "has_expose"},
			{"HasEntrypointOrCmd", "bool", "has_entrypoint_or_cmd"},
			{"InstallsCurlOrWget", "bool", "installs_curl_or_wget"},
			{"InstallsBuildTools", "bool", "installs_build_tools"},
			{"HasAptGetClean", "bool", "has_apt_get_clean"},
			{"WorldWritable", "bool", "world_writable"},
			{"HasSecretsInEnvOrArg", "bool", "has_secrets_in_env_or_arg"},
			{"UsesNpmInstall", "bool", "uses_npm_install"},
			{"UsesNpmCiWithoutIgnoreScripts", "bool", "uses_npm_ci_without_ignore_scripts"},
			{"UsesYarnInstallWithoutFrozen", "bool", "uses_yarn_install_without_frozen"},
			{"UsesPipInstallWithoutNoCache", "bool", "uses_pip_install_without_no_cache"},
			{"UsesPipInstallWithoutHashes", "bool", "uses_pip_install_without_hashes"},
			{"UsesCurlBashPipe", "bool", "uses_curl_bash_pipe"},
		}),

		Entry("BGDockerStageMeta", bqwriter.BGDockerStageMeta{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"Path", "string", "path"},
			{"StageIndex", "int", "stage_index"},
			{"BaseImage", "string", "base_image"},
			{"BaseTag", "string", "base_tag"},
		}),

		Entry("BGCIConfig", bqwriter.BGCIConfig{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"Path", "string", "path"},
			{"Content", "string", "content"},
			{"UsesNpmInstall", "bool", "uses_npm_install"},
			{"UsesNpmCiWithoutIgnoreScripts", "bool", "uses_npm_ci_without_ignore_scripts"},
			{"UsesYarnInstallWithoutFrozen", "bool", "uses_yarn_install_without_frozen"},
			{"UsesPipInstallWithoutNoCache", "bool", "uses_pip_install_without_no_cache"},
			{"UsesPipInstallWithoutHashes", "bool", "uses_pip_install_without_hashes"},
			{"UsesCurlBashPipe", "bool", "uses_curl_bash_pipe"},
			{"UsesSudo", "bool", "uses_sudo"},
		}),

		Entry("BGSBOMPackages", bqwriter.BGSBOMPackages{}, []fieldSpec{
			{"RepoID", "int64", "repo_id"},
			{"WhenCollected", "time.Time", "when_collected"},
			{"Name", "string", "name"},
			{"Version", "string", "version"},
			{"License", "string", "license"},
			{"PURL", "string", "purl"},
		}),
	)
})

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

func readGoldenFile(name string) []byte {
	data, err := os.ReadFile(filepath.Join(testdataDir(), name))
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to read golden file: "+name)
	return data
}

func toJSON(v any) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	return data
}

var _ = Describe("Convert* golden file tests", func() {
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
				{Path: "Dockerfile", Content: "FROM alpine"},
			},
		},
		CIConfig: []models.FileEntry{
			{Path: ".github/workflows/ci.yml", Content: "name: CI"},
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

	It("ConvertToBG matches golden file", func() {
		result := bqwriter.ConvertToBG(entry, snapshot)
		actual := toJSON(result)
		expected := readGoldenFile("golden_repo.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})

	It("ConvertLanguages matches golden file", func() {
		result := bqwriter.ConvertLanguages(entry, snapshot)
		sort.Slice(result, func(i, j int) bool {
			return result[i].Language < result[j].Language
		})
		actual := toJSON(result)
		expected := readGoldenFile("golden_languages.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})

	It("ConvertDockerfileFeatures matches golden file", func() {
		features, _ := bqwriter.ConvertDockerfileFeatures(entry, snapshot)
		actual := toJSON(features)
		expected := readGoldenFile("golden_dockerfile_features.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})

	It("ConvertDockerfileFeatures stages match golden file", func() {
		_, stages := bqwriter.ConvertDockerfileFeatures(entry, snapshot)
		actual := toJSON(stages)
		expected := readGoldenFile("golden_dockerfile_stages.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})

	It("ConvertCI matches golden file", func() {
		result := bqwriter.ConvertCI(entry, snapshot)
		actual := toJSON(result)
		expected := readGoldenFile("golden_ci_config.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})

	It("ConvertSBOMPackages matches golden file", func() {
		result := bqwriter.ConvertSBOMPackages(entry, snapshot)
		actual := toJSON(result)
		expected := readGoldenFile("golden_sbom_packages.json")
		Expect(string(actual)).To(MatchJSON(string(expected)))
	})
})

type tableSchema struct {
	Table   string        `json:"table"`
	Columns []columnEntry `json:"columns"`
}

type columnEntry struct {
	Field  string `json:"field"`
	GoType string `json:"go_type"`
	BQName string `json:"bq_name"`
}

func schemaFilePath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "schema", "bigquery_schema.json")
}

func generateSchema() []tableSchema {
	tables := []struct {
		name    string
		example any
	}{
		{"repos", bqwriter.BGRepoEntry{}},
		{"repo_languages", bqwriter.BGRepoLanguage{}},
		{"dockerfile_features", bqwriter.BGDockerfileFeatures{}},
		{"dockerfile_stages", bqwriter.BGDockerStageMeta{}},
		{"ci_config", bqwriter.BGCIConfig{}},
		{"sbom_packages", bqwriter.BGSBOMPackages{}},
	}

	var schema []tableSchema
	for _, t := range tables {
		rt := reflect.TypeOf(t.example)
		var cols []columnEntry
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			cols = append(cols, columnEntry{
				Field:  f.Name,
				GoType: f.Type.String(),
				BQName: f.Tag.Get("bigquery"),
			})
		}
		schema = append(schema, tableSchema{Table: t.name, Columns: cols})
	}
	return schema
}

var _ = Describe("Schema file sync", func() {
	It("schema/bigquery_schema.json must match current BG structs", func() {
		generated := generateSchema()
		data, err := json.MarshalIndent(generated, "", "  ")
		Expect(err).NotTo(HaveOccurred())
		data = append(data, '\n')

		path := schemaFilePath()

		if *updateSchemaFile {
			err := os.MkdirAll(filepath.Dir(path), 0o755)
			Expect(err).NotTo(HaveOccurred())
			err = os.WriteFile(path, data, 0o644)
			Expect(err).NotTo(HaveOccurred())
			return
		}

		committed, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred(),
			"schema/bigquery_schema.json not found. Generate it with:\n"+
				"  go test ./internal/bqwriter/ -run 'Schema file sync' -update-schema")

		Expect(string(data)).To(Equal(string(committed)),
			"schema/bigquery_schema.json is stale. Regenerate with:\n"+
				"  go test ./internal/bqwriter/ -run 'Schema file sync' -update-schema")
	})
})
