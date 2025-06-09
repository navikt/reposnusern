package models

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
	SBOM      map[string]interface{} `json:"sbom"`
}

type OrgRepos struct {
	Org   string      `json:"org"`
	Repos []RepoEntry `json:"repos"`
}
