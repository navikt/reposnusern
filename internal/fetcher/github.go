package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TreeFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// GetJSONWithRateLimit henter JSON fra en URL og respekterer GitHub rate-limiting.
func GetJSONWithRateLimit(url, token string, out interface{}) error {
	for {
		slog.Info("Henter URL", "url", url)
		req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if reset != "" {
				if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
					wait := time.Until(time.Unix(ts, 0)) + time.Second
					slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
					time.Sleep(wait)
					continue
				}
			}
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}

func GetJSONMap(url, token string) map[string]interface{} {
	var out map[string]interface{}
	err := GetJSONWithRateLimit(url, token, &out)
	if err != nil {
		slog.Error("Feil ved henting", "url", url, "error", err)
		return nil
	}
	return out
}

func GetReadme(fullName, token string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/readme", fullName)
	var payload map[string]interface{}
	if err := GetJSONWithRateLimit(url, token, &payload); err != nil {
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
	if err := GetJSONWithRateLimit(url, token, &result); err != nil {
		return ""
	}
	if content, ok := result["content"].(string); ok {
		d, _ := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\n", ""))
		return string(d)
	}
	return ""
}

func AppendFile(files map[string][]map[string]string, key string, tf TreeFile, repo, token string) {
	content := getGitBlob(tf.URL, token)
	files[key] = append(files[key], map[string]string{
		"path":    tf.Path,
		"content": content,
	})
}

func AppendCI(ciList *[]map[string]string, tf TreeFile, repo, token string) {
	content := getGitBlob(tf.URL, token)
	*ciList = append(*ciList, map[string]string{
		"path":    tf.Path,
		"content": content,
	})
}

func ParseTree(tree map[string]interface{}) []TreeFile {
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

func IsDependencyFile(p string) bool {
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
