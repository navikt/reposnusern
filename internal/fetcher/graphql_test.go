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
