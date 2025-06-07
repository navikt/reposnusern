package main

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: false,
	}))
	slog.SetDefault(logger)

	org := os.Getenv("ORG")
	if org == "" {
		slog.Error("Du må angi organisasjon via ORG=<orgnavn>")
		os.Exit(1)
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		slog.Error("Mangler GITHUB_TOKEN i environment")
		os.Exit(1)
	}

	// Hent full repo-metadata som map
	repos := fetcher.GetAllRepos(org, token)

	if err := fetcher.StoreRepoDumpJSON("data", org, repos); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

	allData := fetcher.OrgRepos{
		Org:   org,
		Repos: []map[string]interface{}{},
	}

	for i, r := range repos {
		fullName := r["full_name"].(string)
		if r["archived"].(bool) {
			continue
		}

		slog.Info("Bearbeider repo", "index", i+1, "total", len(repos), "repo", fullName)

		result := map[string]interface{}{
			"repo":      r,
			"languages": fetcher.GetJSONMap(fmt.Sprintf("https://api.github.com/repos/%s/languages", fullName), token),
			"files":     map[string][]map[string]string{},
			"security":  map[string]bool{},
		}
		ciConfig := []map[string]string{}

		tree := fetcher.GetJSONMap(fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", fullName, r["default_branch"].(string)), token)
		treeFiles := fetcher.ParseTree(tree)

		for _, tf := range treeFiles {
			lpath := strings.ToLower(tf.Path)
			switch {
			case fetcher.IsDependencyFile(lpath):
				fetcher.AppendFile(result["files"].(map[string][]map[string]string), path.Base(tf.Path), tf, fullName, token)
			case strings.HasPrefix(path.Base(lpath), "dockerfile"):
				fetcher.AppendFile(result["files"].(map[string][]map[string]string), path.Base(tf.Path), tf, fullName, token)
			case strings.HasPrefix(tf.Path, ".github/workflows/"):
				fetcher.AppendCI(&ciConfig, tf, fullName, token)
			case tf.Path == "SECURITY.md":
				result["security"].(map[string]bool)["has_security_md"] = true
			case tf.Path == ".github/dependabot.yml":
				result["security"].(map[string]bool)["has_dependabot"] = true
			case tf.Path == ".github/codeql.yml":
				result["security"].(map[string]bool)["has_codeql"] = true
			}
		}

		result["ci_config"] = ciConfig

		if readme := fetcher.GetReadme(fullName, token); readme != "" {
			result["readme"] = readme
		}

		allData.Repos = append(allData.Repos, result)
	}

	if err := fetcher.StoreRepoDetailedJSON("data", org, allData); err != nil {
		slog.Error("Feil under lagring av repo-dump", "error", err)
		os.Exit(1)
	}

}
