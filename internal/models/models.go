package models

type FileEntry struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type License struct {
	SpdxID string `json:"spdx_id"`
}

type RepoMeta struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	FullName     string          `json:"full_name"`
	Description  string          `json:"description"`
	Stars        int64           `json:"stargazers_count"`
	Forks        int64           `json:"forks_count"`
	Archived     bool            `json:"archived"`
	Private      bool            `json:"private"`
	IsFork       bool            `json:"fork"`
	Language     string          `json:"language"`
	Size         int64           `json:"size"`
	UpdatedAt    string          `json:"updated_at"`
	PushedAt     string          `json:"pushed_at"`
	CreatedAt    string          `json:"created_at"`
	HtmlUrl      string          `json:"html_url"`
	Topics       []string        `json:"topics"`
	Visibility   string          `json:"visibility"`
	OpenIssues   int64           `json:"open_issues_count"`
	LanguagesURL string          `json:"languages_url"`
	License      *License        `json:"license"`
	Readme       string          `json:"readme"`
	Security     map[string]bool `json:"security"`
}

type RepoEntry struct {
	Repo      RepoMeta               `json:"repo"`
	Languages map[string]int         `json:"languages"`
	Files     map[string][]FileEntry `json:"files"`
	CIConfig  []FileEntry            `json:"ci_config"`
	SBOM      map[string]interface{} `json:"sbom"`
}

type OrgRepos struct {
	Org   string      `json:"org"`
	Repos []RepoEntry `json:"repos"`
}
