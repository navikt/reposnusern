package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/fetcher"
)

type TreeFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

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
	debug := os.Getenv("REPOSNUSERDEBUG") == "true"
	repos := []map[string]interface{}{}
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", org, page)
		var pageRepos []map[string]interface{}
		slog.Info("Henter repos", "page", page)
		err := fetcher.GetJSONWithRateLimit(url, token, &pageRepos)
		if err != nil {
			slog.Error("Kunne ikke hente repo-metadata", "error", err)
			os.Exit(1)
		}
		if len(pageRepos) == 0 {
			break
		}

		if debug {
			// Shuffle og velg 3 tilfeldig
			rand.Shuffle(len(pageRepos), func(i, j int) {
				pageRepos[i], pageRepos[j] = pageRepos[j], pageRepos[i]
			})
			repos = append(repos, pageRepos[:min(3, len(pageRepos))]...)
			break
		} else {
			repos = append(repos, pageRepos...)
		}

		page++
	}

	_ = os.MkdirAll("data", 0755)
	rawOut, _ := json.MarshalIndent(repos, "", "  ")
	rawFile := fmt.Sprintf("data/%s_repos_raw_dump.json", org)
	_ = os.WriteFile(rawFile, rawOut, 0644)
	slog.Info("Lagret full repo-metadata", "count", len(repos), "file", rawFile)

	allData := map[string]interface{}{
		"org":   org,
		"repos": []map[string]interface{}{},
	}

	for i, r := range repos {
		fullName := r["full_name"].(string)
		if r["archived"].(bool) {
			continue
		}

		slog.Info("Bearbeider repo", "index", i+1, "total", len(repos), "repo", fullName)

		result := map[string]interface{}{
			"repo":      r,
			"languages": getJSONMap(fmt.Sprintf("https://api.github.com/repos/%s/languages", fullName), token),
			"files":     map[string][]map[string]string{},
			"security":  map[string]bool{},
		}
		ciConfig := []map[string]string{}

		tree := getJSONMap(fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", fullName, r["default_branch"].(string)), token)
		treeFiles := parseTree(tree)

		for _, tf := range treeFiles {
			lpath := strings.ToLower(tf.Path)
			switch {
			case isDependencyFile(lpath):
				appendFile(result["files"].(map[string][]map[string]string), path.Base(tf.Path), tf, fullName, token)
			case strings.HasPrefix(path.Base(lpath), "dockerfile"):
				appendFile(result["files"].(map[string][]map[string]string), path.Base(tf.Path), tf, fullName, token)
			case strings.HasPrefix(tf.Path, ".github/workflows/"):
				appendCI(&ciConfig, tf, fullName, token)
			case tf.Path == "SECURITY.md":
				result["security"].(map[string]bool)["has_security_md"] = true
			case tf.Path == ".github/dependabot.yml":
				result["security"].(map[string]bool)["has_dependabot"] = true
			case tf.Path == ".github/codeql.yml":
				result["security"].(map[string]bool)["has_codeql"] = true
			}
		}

		result["ci_config"] = ciConfig

		if readme := getReadme(fullName, token); readme != "" {
			result["readme"] = readme
		}

		allData["repos"] = append(allData["repos"].([]map[string]interface{}), result)
	}

	outputFile := fmt.Sprintf("data/%s_analysis_data.json", org)
	allBytes, _ := json.MarshalIndent(allData, "", "  ")
	_ = os.WriteFile(outputFile, allBytes, 0644)
	slog.Info("Lagret samlet analyse", "file", outputFile)
}

func getJSONMap(url, token string) map[string]interface{} {
	var out map[string]interface{}
	err := fetcher.GetJSONWithRateLimit(url, token, &out)
	if err != nil {
		slog.Error("Feil ved henting", "url", url, "error", err)
		return nil
	}
	return out
}

func getReadme(fullName, token string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/readme", fullName)
	var payload map[string]interface{}
	if err := fetcher.GetJSONWithRateLimit(url, token, &payload); err != nil {
		return ""
	}
	if content, ok := payload["content"].(string); ok {
		decoded, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		return string(decoded)
	}
	return ""
}

func getGitBlob(url, token string) string {
	var result map[string]interface{}
	if err := fetcher.GetJSONWithRateLimit(url, token, &result); err != nil {
		return ""
	}
	if content, ok := result["content"].(string); ok {
		d, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		return string(d)
	}
	return ""
}

func appendFile(files map[string][]map[string]string, key string, tf TreeFile, repo, token string) {
	content := getGitBlob(tf.URL, token)
	files[key] = append(files[key], map[string]string{
		"path":    tf.Path,
		"content": content,
	})
}

func appendCI(ciList *[]map[string]string, tf TreeFile, repo, token string) {
	content := getGitBlob(tf.URL, token)
	*ciList = append(*ciList, map[string]string{
		"path":    tf.Path,
		"content": content,
	})
}

func parseTree(tree map[string]interface{}) []TreeFile {
	files := []TreeFile{}
	if tree == nil {
		return files
	}
	if arr, ok := tree["tree"].([]interface{}); ok {
		for _, item := range arr {
			entry := item.(map[string]interface{})
			if entry["type"] == "blob" {
				files = append(files, TreeFile{
					Path: entry["path"].(string),
					URL:  entry["url"].(string),
					Type: entry["type"].(string),
				})
			}
		}
	}
	return files
}

func isDependencyFile(p string) bool {
	files := []string{
		"package.json", "pom.xml", "build.gradle", "build.gradle.kts",
		"go.mod", "cargo.toml", "requirements.txt", "pyproject.toml",
		"composer.json", ".csproj", "gemfile", "gemfile.lock",
		"yarn.lock", "pnpm-lock.yaml", "package-lock.json",
	}
	for _, f := range files {
		if strings.HasSuffix(p, f) {
			return true
		}
	}
	return false
}
