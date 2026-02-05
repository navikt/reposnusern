package parser

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/models"
)

// LockfilePairing represents a manifest file and its corresponding lockfile
type LockfilePairing struct {
	Manifest string `json:"manifest"`
	Lockfile string `json:"lockfile"` // empty string if missing
}

// EcosystemConfig defines manifest and lockfile patterns for a package ecosystem
type EcosystemConfig struct {
	Manifests []string
	Lockfiles []string
}

// Ecosystem configurations - comprehensive language support
var ecosystems = map[string]EcosystemConfig{
	"javascript": {
		Manifests: []string{"package.json"},
		Lockfiles: []string{"package-lock.json", "npm-shrinkwrap.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "bun.lock", "deno.lock"},
	},
	"deno": {
		Manifests: []string{"deno.json"},
		Lockfiles: []string{"deno.lock"},
	},
	"python": {
		Manifests: []string{"Pipfile", "pyproject.toml", "requirements.txt", "setup.py"},
		Lockfiles: []string{"Pipfile.lock", "poetry.lock", "pdm.lock", "uv.lock"},
	},
	"ruby": {
		Manifests: []string{"Gemfile"},
		Lockfiles: []string{"Gemfile.lock"},
	},
	"php": {
		Manifests: []string{"composer.json"},
		Lockfiles: []string{"composer.lock"},
	},
	"rust": {
		Manifests: []string{"Cargo.toml"},
		Lockfiles: []string{"Cargo.lock"},
	},
	"go": {
		Manifests: []string{"go.mod"},
		Lockfiles: []string{"go.sum"},
	},
	"java": {
		Manifests: []string{"pom.xml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"},
		Lockfiles: []string{"gradle.lockfile"},
	},
	"dotnet": {
		Manifests: []string{"packages.config"},
		Lockfiles: []string{"packages.lock.json"},
	},
	"swift": {
		Manifests: []string{"Package.swift"},
		Lockfiles: []string{"Package.resolved"},
	},
	"dart": {
		Manifests: []string{"pubspec.yaml"},
		Lockfiles: []string{"pubspec.lock"},
	},
	"elixir": {
		Manifests: []string{"mix.exs"},
		Lockfiles: []string{"mix.lock"},
	},
	"scala": {
		Manifests: []string{"build.sbt"},
		Lockfiles: []string{"coursier.lock"},
	},
	"clojure": {
		Manifests: []string{"project.clj", "deps.edn"},
		Lockfiles: []string{},
	},
	"r": {
		Manifests: []string{"DESCRIPTION"},
		Lockfiles: []string{"renv.lock"},
	},
	"perl": {
		Manifests: []string{"cpanfile"},
		Lockfiles: []string{"cpanfile.snapshot"},
	},
	"haskell": {
		Manifests: []string{"stack.yaml", "package.yaml", "cabal.project"},
		Lockfiles: []string{"stack.yaml.lock", "cabal.project.freeze"},
	},
	"cpp": {
		Manifests: []string{"conanfile.txt", "conanfile.py", "vcpkg.json"},
		Lockfiles: []string{"conan.lock", "vcpkg-lock.json"},
	},
	"nim": {
		Manifests: []string{},
		Lockfiles: []string{"nimble.lock"},
	},
	"crystal": {
		Manifests: []string{"shard.yml"},
		Lockfiles: []string{"shard.lock"},
	},
}

// DetectLockfilePairings analyzes repository files and returns manifest-lockfile pairings
func DetectLockfilePairings(files map[string][]models.FileEntry) []LockfilePairing {
	var pairings []LockfilePairing

	// Extract just paths for logging (without content)
	filePaths := make(map[string][]string)
	for fileType, entries := range files {
		paths := make([]string, len(entries))
		for i, entry := range entries {
			paths[i] = entry.Path
		}
		filePaths[fileType] = paths
	}
	slog.Debug("Detecting lockfile pairings", "file_structure", filePaths)

	// Build a set of all file paths for quick lookup
	filePathSet := make(map[string]bool)
	for _, fileEntries := range files {
		for _, entry := range fileEntries {
			filePathSet[entry.Path] = true
		}
	}

	// For each enabled ecosystem, detect manifest-lockfile pairings
	for _, config := range ecosystems {
		pairings = append(pairings, detectPairingsForEcosystem(config, filePathSet)...)
	}

	return pairings
}

// detectPairingsForEcosystem finds all manifest files for an ecosystem and checks for lockfiles
func detectPairingsForEcosystem(config EcosystemConfig, filePaths map[string]bool) []LockfilePairing {
	var pairings []LockfilePairing

	// Find all manifest files in the repository
	manifests := findFiles(filePaths, config.Manifests)

	if len(manifests) > 0 {
		slog.Debug("Found manifests", "count", len(manifests), "manifests", manifests)
	}

	// For each manifest, look for corresponding lockfiles
	for _, manifestPath := range manifests {
		slog.Debug("Checking manifest for lockfile", "manifest", manifestPath)
		// Get the directory containing the manifest
		manifestDir := filepath.Dir(manifestPath)
		if manifestDir == "." {
			manifestDir = ""
		}

		// Check for each possible lockfile in the same directory
		foundLockfile := ""
		for _, lockfileName := range config.Lockfiles {
			lockfilePath := lockfileName
			if manifestDir != "" {
				lockfilePath = filepath.Join(manifestDir, lockfileName)
			}

			if filePaths[lockfilePath] {
				foundLockfile = lockfilePath
				slog.Debug("Found lockfile for manifest", "manifest", manifestPath, "lockfile", foundLockfile)
				break // Found a lockfile, stop looking
			}
		}

		// Create pairing (lockfile will be empty string if not found)
		if foundLockfile == "" {
			slog.Debug("No lockfile found for manifest", "manifest", manifestPath)
		}

		pairings = append(pairings, LockfilePairing{
			Manifest: manifestPath,
			Lockfile: foundLockfile,
		})
	}

	return pairings
}

// findFiles returns all file paths matching any of the given filenames
func findFiles(filePaths map[string]bool, filenames []string) []string {
	var matches []string

	for path := range filePaths {
		basename := filepath.Base(path)
		for _, filename := range filenames {
			if basename == filename {
				matches = append(matches, path)
				break
			}
		}
	}

	return matches
}

// HasProperLockfiles returns true if all manifests have corresponding lockfiles
func HasProperLockfiles(pairings []LockfilePairing) bool {
	if len(pairings) == 0 {
		return false // No manifests found, so no proper lockfiles
	}

	for _, pairing := range pairings {
		if pairing.Lockfile == "" || strings.TrimSpace(pairing.Lockfile) == "" {
			return false // Found a manifest without a lockfile
		}
	}

	return true // All manifests have lockfiles
}
