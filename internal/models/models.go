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

// result := map[string]interface{}{
// 	"repo":      r,
// 	"languages": data["languages"],
// 	"files":     data["files"],
// 	"security":  data["security"],
// 	"ci_config": data["ci_config"],
// 	"readme":    data["readme"],
// 	"sbom":      data["sbom"],
// }

type OrgRepos struct {
	Org   string      `json:"org"`
	Repos []RepoEntry `json:"repos"`
}
