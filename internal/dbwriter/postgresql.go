package dbwriter

import "strings"

type FileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type RepoEntry struct {
	Repo      map[string]interface{} `json:"repo"`
	Languages map[string]int         `json:"languages"`
	Files     map[string][]FileEntry `json:"files"`
	CIConfig  []FileEntry            `json:"ci_config"`
	Readme    string                 `json:"readme"`
	Security  map[string]bool        `json:"security"`
}

type Dump struct {
	Org   string      `json:"org"`
	Repos []RepoEntry `json:"repos"`
}

func SafeString(v interface{}) string {
	if v == nil {
		return ""
	}
	return v.(string)
}

func JoinStrings(arr interface{}) string {
	if arr == nil {
		return ""
	}
	raw := arr.([]interface{})
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, v.(string))
	}
	return strings.Join(out, ",")
}

func ExtractLicense(r map[string]interface{}) string {
	if r["license"] == nil {
		return ""
	}
	return r["license"].(map[string]interface{})["spdx_id"].(string)
}

func IsDependencyFile(name string) bool {
	known := []string{
		"package.json", "pom.xml", "build.gradle", "build.gradle.kts",
		"go.mod", "cargo.toml", "requirements.txt", "pyproject.toml",
		"composer.json", ".csproj", "gemfile", "gemfile.lock",
		"yarn.lock", "pnpm-lock.yaml", "package-lock.json",
	}
	name = strings.ToLower(name)
	for _, k := range known {
		if k == name {
			return true
		}
	}
	return false
}
