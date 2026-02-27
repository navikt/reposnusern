package parser

import (
	"testing"

	"github.com/jonmartinstorm/reposnusern/internal/models"
)

func TestDetectLockfilePairings_SimplePackageJson(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
			{Path: "package-lock.json", Content: "{}"},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing, got %d", len(pairings))
	}

	if pairings[0].Manifest != "package.json" {
		t.Errorf("expected manifest 'package.json', got '%s'", pairings[0].Manifest)
	}

	if pairings[0].Lockfile != "package-lock.json" {
		t.Errorf("expected lockfile 'package-lock.json', got '%s'", pairings[0].Lockfile)
	}
}

func TestDetectLockfilePairings_YarnLock(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
			{Path: "yarn.lock", Content: ""},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing, got %d", len(pairings))
	}

	if pairings[0].Lockfile != "yarn.lock" {
		t.Errorf("expected lockfile 'yarn.lock', got '%s'", pairings[0].Lockfile)
	}
}

func TestDetectLockfilePairings_NoLockfile(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing, got %d", len(pairings))
	}

	if pairings[0].Manifest != "package.json" {
		t.Errorf("expected manifest 'package.json', got '%s'", pairings[0].Manifest)
	}

	if pairings[0].Lockfile != "" {
		t.Errorf("expected empty lockfile, got '%s'", pairings[0].Lockfile)
	}
}

func TestDetectLockfilePairings_MultipleLockfiles(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
			{Path: "package-lock.json", Content: "{}"},
			{Path: "yarn.lock", Content: ""},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing, got %d", len(pairings))
	}

	if pairings[0].Lockfile != "package-lock.json" {
		t.Errorf("expected lockfile 'package-lock.json', got '%s'", pairings[0].Lockfile)
	}
}

func TestDetectLockfilePairings_Subdirectories(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
			{Path: "package-lock.json", Content: "{}"},
			{Path: "frontend/package.json", Content: "{}"},
			{Path: "frontend/yarn.lock", Content: ""},
			{Path: "backend/package.json", Content: "{}"},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 3 {
		t.Fatalf("expected 3 pairings, got %d", len(pairings))
	}

	pairingsMap := make(map[string]string)
	for _, p := range pairings {
		pairingsMap[p.Manifest] = p.Lockfile
	}

	if pairingsMap["package.json"] != "package-lock.json" {
		t.Errorf("root package.json should have package-lock.json, got '%s'", pairingsMap["package.json"])
	}

	if pairingsMap["frontend/package.json"] != "frontend/yarn.lock" {
		t.Errorf("frontend/package.json should have frontend/yarn.lock, got '%s'", pairingsMap["frontend/package.json"])
	}

	if pairingsMap["backend/package.json"] != "" {
		t.Errorf("backend/package.json should have empty lockfile, got '%s'", pairingsMap["backend/package.json"])
	}
}

func TestDetectLockfilePairings_NoManifests(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "README.md", Content: "# Project"},
			{Path: "src/main.js", Content: "console.log('hello')"},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 0 {
		t.Errorf("expected 0 pairings, got %d", len(pairings))
	}
}

func TestHasCompleteLockfiles_AllHaveLockfiles(t *testing.T) {
	pairings := []models.LockfilePairing{
		{Manifest: "package.json", Lockfile: "package-lock.json"},
		{Manifest: "frontend/package.json", Lockfile: "frontend/yarn.lock"},
	}

	result := HasCompleteLockfiles(pairings)

	if !result {
		t.Error("expected true when all manifests have lockfiles")
	}
}

func TestHasCompleteLockfiles_MissingLockfile(t *testing.T) {
	pairings := []models.LockfilePairing{
		{Manifest: "package.json", Lockfile: "package-lock.json"},
		{Manifest: "frontend/package.json", Lockfile: ""},
	}

	result := HasCompleteLockfiles(pairings)

	if result {
		t.Error("expected false when at least one manifest lacks a lockfile")
	}
}

func TestHasCompleteLockfiles_NoManifests(t *testing.T) {
	pairings := []models.LockfilePairing{}

	result := HasCompleteLockfiles(pairings)

	if result {
		t.Error("expected false when there are no manifests")
	}
}

func TestHasCompleteLockfiles_WhitespaceLockfile(t *testing.T) {
	pairings := []models.LockfilePairing{
		{Manifest: "package.json", Lockfile: "   "},
	}

	result := HasCompleteLockfiles(pairings)

	if result {
		t.Error("expected false when lockfile is only whitespace")
	}
}

func TestIsIgnoredPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"package.json", false},
		{"frontend/package.json", false},
		{"node_modules/express/package.json", true},
		{"app/node_modules/lodash/package.json", true},
		{"vendor/autoload.php", true},
		{"lib/vendor/something/composer.json", true},
		{"go.mod", false},
		{"services/api/go.mod", false},
		{"vendor/github.com/pkg/errors/go.mod", true},
		{"site-packages/requests/setup.py", true},
		{".venv/lib/site-packages/flask/setup.py", true},
		{"vendor/bundle/gems/rails/Gemfile", true},
		{"myvendor/bundle/gems/rails/Gemfile", false},
		{"myvendor/package.json", false},
		{"node_modules_extra/package.json", false},
		{"vcpkg_installed/x64-linux/vcpkg.json", true},
		{".dart_tool/package_config.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsIgnoredPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsIgnoredPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDetectLockfilePairings_SkipsIgnoredFiles(t *testing.T) {
	files := map[string][]models.FileEntry{
		"other": {
			{Path: "package.json", Content: "{}"},
			{Path: "package-lock.json", Content: "{}"},
			{Path: "node_modules/express/package.json", Content: "{}"},
			{Path: "node_modules/lodash/package.json", Content: "{}"},
		},
	}

	pairings := DetectLockfilePairings(files)

	if len(pairings) != 1 {
		t.Fatalf("expected 1 pairing (ignored files should be skipped), got %d", len(pairings))
	}

	if pairings[0].Manifest != "package.json" {
		t.Errorf("expected manifest 'package.json', got '%s'", pairings[0].Manifest)
	}
}
