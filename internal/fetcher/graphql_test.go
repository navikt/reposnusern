package fetcher

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/models"
)

func TestConvertToFileEntries(t *testing.T) {
	input := []map[string]string{
		{"path": "Dockerfile", "content": "FROM alpine"},
		{"path": "build.sh", "content": "#!/bin/sh"},
	}
	expected := []models.FileEntry{
		{Path: "Dockerfile", Content: "FROM alpine"},
		{Path: "build.sh", Content: "#!/bin/sh"},
	}

	result := convertToFileEntries(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestConvertFiles(t *testing.T) {
	input := map[string][]map[string]string{
		"dockerfile": {
			{"path": "Dockerfile", "content": "FROM alpine"},
		},
		"scripts": {
			{"path": "build.sh", "content": "#!/bin/sh"},
		},
	}
	expected := map[string][]models.FileEntry{
		"dockerfile": {
			{Path: "Dockerfile", Content: "FROM alpine"},
		},
		"scripts": {
			{Path: "build.sh", Content: "#!/bin/sh"},
		},
	}

	result := convertFiles(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %+v, got %+v", expected, result)
	}
}

func TestBuildRepoQuery(t *testing.T) {
	owner := "navikt"
	repo := "arbeidsgiver"
	query := buildRepoQuery(owner, repo)

	if !strings.Contains(query, `repository(owner: "navikt", name: "arbeidsgiver")`) {
		t.Errorf("buildRepoQuery() mangler korrekt owner/repo: %s", query)
	}
	if !strings.Contains(query, "defaultBranchRef") {
		t.Errorf("buildRepoQuery() ser ikke ut til å inkludere forventet GraphQL-innhold")
	}
}

func TestParseRepoData_Minimal(t *testing.T) {
	data := map[string]interface{}{
		"languages": map[string]interface{}{
			"edges": []interface{}{
				map[string]interface{}{
					"size": float64(100),
					"node": map[string]interface{}{"name": "Go"},
				},
			},
		},
		"README": map[string]interface{}{
			"text": "Hello world",
		},
		"SECURITY":   map[string]interface{}{},
		"dependabot": nil,
		"codeql":     map[string]interface{}{},
	}

	base := models.RepoMeta{Name: "arbeidsgiver"}
	entry := parseRepoData(data, base)

	if entry == nil {
		t.Fatal("parseRepoData() returnerte nil")
	}
	if entry.Repo.Name != "arbeidsgiver" {
		t.Errorf("Repo.Name = %s, vil ha arbeidsgiver", entry.Repo.Name)
	}
	if entry.Readme != "Hello world" {
		t.Errorf("Readme = %q, vil ha 'Hello world'", entry.Readme)
	}
	if entry.Languages["Go"] != 100 {
		t.Errorf("Languages[Go] = %d, vil ha 100", entry.Languages["Go"])
	}
	if !entry.Security["has_security_md"] || entry.Security["has_dependabot"] {
		t.Errorf("Security metadata feil: %+v", entry.Security)
	}
}

func TestExtractLanguages(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want map[string]int
	}{
		{
			name: "valid languages",
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
		{
			name: "missing node",
			data: map[string]interface{}{
				"languages": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"size": float64(100),
						},
					},
				},
			},
			want: map[string]int{},
		},
		{
			name: "missing name",
			data: map[string]interface{}{
				"languages": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"size": float64(100),
							"node": map[string]interface{}{},
						},
					},
				},
			},
			want: map[string]int{},
		},
		{
			name: "missing size",
			data: map[string]interface{}{
				"languages": map[string]interface{}{
					"edges": []interface{}{
						map[string]interface{}{
							"node": map[string]interface{}{"name": "Rust"},
						},
					},
				},
			},
			want: map[string]int{},
		},
		{
			name: "invalid edges type",
			data: map[string]interface{}{
				"languages": map[string]interface{}{
					"edges": "not-a-list",
				},
			},
			want: map[string]int{},
		},
		{
			name: "missing languages field",
			data: map[string]interface{}{},
			want: map[string]int{},
		},
		{
			name: "edge is not a map",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLanguages(tt.data)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractLanguages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCI(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want []models.FileEntry
	}{
		{
			name: "valid CI entry",
			data: map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"name": "build.yml",
							"object": map[string]interface{}{
								"text": "name: Build\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest",
							},
						},
					},
				},
			},
			want: []models.FileEntry{
				{
					Path:    ".github/workflows/build.yml",
					Content: "name: Build\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest",
				},
			},
		},
		{
			name: "missing object",
			data: map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"name": "test.yml",
						},
					},
				},
			},
			want: []models.FileEntry{},
		},
		{
			name: "object without text",
			data: map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						map[string]interface{}{
							"name": "deploy.yml",
							"object": map[string]interface{}{
								"notText": "something else",
							},
						},
					},
				},
			},
			want: []models.FileEntry{},
		},
		{
			name: "entry is not a map",
			data: map[string]interface{}{
				"workflows": map[string]interface{}{
					"entries": []interface{}{
						"not-a-map",
					},
				},
			},
			want: []models.FileEntry{},
		},
		{
			name: "missing workflows",
			data: map[string]interface{}{},
			want: []models.FileEntry{},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			got := extractCI(tt.data)
			if got == nil {
				got = []models.FileEntry{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractCI() = %#v, want %#v", got, tt.want)
			}
		})

	}
}

func TestExtractReadme(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{
			name: "README present with text",
			data: map[string]interface{}{
				"README": map[string]interface{}{
					"text": "This is a README",
				},
			},
			want: "This is a README",
		},
		{
			name: "README missing",
			data: map[string]interface{}{},
			want: "",
		},
		{
			name: "README present but no text",
			data: map[string]interface{}{
				"README": map[string]interface{}{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractReadme(tt.data)
			if got != tt.want {
				t.Errorf("extractReadme() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractSecurity(t *testing.T) {
	data := map[string]interface{}{
		"SECURITY":   map[string]interface{}{},
		"dependabot": nil,
		"codeql":     map[string]interface{}{},
	}

	got := extractSecurity(data)
	want := map[string]bool{
		"has_security_md": true,
		"has_dependabot":  false,
		"has_codeql":      true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("extractSecurity() = %v, want %v", got, want)
	}
}

func TestExtractFiles(t *testing.T) {
	data := map[string]interface{}{
		"dependencies": map[string]interface{}{
			"entries": []interface{}{
				// ✅ Gyldig Dockerfile med innhold
				map[string]interface{}{
					"name": "Dockerfile",
					"object": map[string]interface{}{
						"text": "FROM alpine",
					},
				},
				// ❌ Ikke en dockerfil
				map[string]interface{}{
					"name": "README.md",
					"object": map[string]interface{}{
						"text": "This is not a Dockerfile",
					},
				},
				// ❌ Dockerfile uten innhold
				map[string]interface{}{
					"name":   "Dockerfile.empty",
					"object": map[string]interface{}{},
				},
				// ❌ Ugyldig struktur
				"not-a-map",
			},
		},
	}

	got := extractFiles(data)

	if len(got) != 1 {
		t.Fatalf("expected 1 dockerfile entry, got %d", len(got))
	}

	dockerfiles, ok := got["dockerfile"]
	if !ok {
		t.Fatalf("expected key 'dockerfile' in result")
	}

	if len(dockerfiles) != 1 {
		t.Errorf("expected 1 dockerfile entry, got %d", len(dockerfiles))
	}

	if dockerfiles[0].Path != "Dockerfile" || dockerfiles[0].Content != "FROM alpine" {
		t.Errorf("unexpected dockerfile content: %+v", dockerfiles[0])
	}
}
