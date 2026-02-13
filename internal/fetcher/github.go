package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/jonmartinstorm/reposnusern/internal/config"
	"github.com/jonmartinstorm/reposnusern/internal/models"
	"github.com/jonmartinstorm/reposnusern/internal/parser"
)

// apiCallCounter tracks the total number of external API calls made
var apiCallCounter atomic.Int64

type RepoFetcher struct {
	Cfg config.Config
}

// TreeEntry represents a single entry in a Git tree from GitHub's API
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int    `json:"size"`
	URL  string `json:"url"`
}

type TreeFile struct {
	Path string `json:"path"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// Injecter en klient (for testbarhet)
var HttpClient = http.DefaultClient

func NewRepoFetcher(cfg config.Config) *RepoFetcher {
	return &RepoFetcher{
		Cfg: cfg,
	}
}

// GetAPICallCount returns the total number of external API calls made
func GetAPICallCount() int64 {
	return apiCallCounter.Load()
}

// CreateGitHubAppTransport creates a transport for the GitHub App
func CreateGitHubAppTransport(config *config.GitHubAppConfig) (http.RoundTripper, error) {
	tr, err := ghinstallation.New(
		http.DefaultTransport,
		config.AppID,
		config.InstallationID,
		config.PrivateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub installation transport: %w", err)
	}
	return tr, nil
}

// CreateGitHubAppToken creates an access token for GitHub API using the provided transport
func CreateGitHubAppToken(ctx context.Context, config *config.GitHubAppConfig) (string, error) {
	tr := http.DefaultTransport
	// Fix: Use AppID instead of InstallationID for the second parameter
	appsTransport, err := ghinstallation.NewAppsTransport(tr, config.AppID, config.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create GitHub apps transport: %w", err)
	}
	itr := ghinstallation.NewFromAppsTransport(appsTransport, config.InstallationID)
	token, err := itr.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}
	return token, nil
}

func (r *RepoFetcher) GetReposPage(ctx context.Context, cfg config.Config, page int) ([]models.RepoMeta, error) {
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all&page=%d", cfg.Org, page)
	var pageRepos []models.RepoMeta
	slog.Info("Henter repos", "page", page)

	token, err := r.GetAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	err = DoRequestWithRateLimit(ctx, "GET", url, token, nil, &pageRepos)
	if err != nil {
		return nil, err
	}

	return pageRepos, nil
}

func (r *RepoFetcher) FetchRepoGraphQL(ctx context.Context, baseRepo models.RepoMeta) (*models.RepoEntry, error) {
	query := BuildRepoQuery(r.Cfg.Org, baseRepo.Name)

	reqBody := map[string]string{"query": query}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Kunne ikke serialisere GraphQL-request", "repo", r.Cfg.Org+"/"+baseRepo.Name, "error", err)
		return nil, err
	}

	token, err := r.GetAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	var result map[string]interface{}
	err = DoRequestWithRateLimit(ctx, "POST", "https://api.github.com/graphql", token, bodyBytes, &result)
	if err != nil {
		slog.Error("GraphQL-kall feilet", "repo", r.Cfg.Org+"/"+baseRepo.Name, "error", err)
		return nil, err
	}

	if errs, ok := result["errors"]; ok {
		slog.Warn("GraphQL-resultat har feil", "repo", r.Cfg.Org+"/"+baseRepo.Name, "errors", errs)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok || data["repository"] == nil {
		slog.Warn("Ingen repository-data fra GraphQL", "repo", r.Cfg.Org+"/"+baseRepo.Name)
		return nil, fmt.Errorf("ingen repository-data for %s/%s", r.Cfg.Org, baseRepo.Name)
	}

	entry := ParseRepoData(data, baseRepo)

	// Hent SBOM hvis feature_sbom er true
	if r.Cfg.Feature_Sbom {
		sbom := r.fetchSBOM(ctx, r.Cfg.Org, baseRepo.Name)
		entry.SBOM = sbom
	}

	// Fetch tree once if we need to search deeper for files
	var treeEntries []TreeEntry
	var treeErr error

	if IsMonorepoCandidate(entry) {
		slog.Info("Monorepo-kandidat – henter dype Dockerfiles og dependencyfiles", "repo", baseRepo.FullName)

		treeEntries, treeErr = r.fetchRepoTreeREST(ctx, r.Cfg.Org, baseRepo.Name)
		if treeErr != nil {
			slog.Warn("Klarte ikke hente repo-tree", "repo", baseRepo.FullName, "error", treeErr)
		}

		if treeEntries != nil {
			files := r.FetchDockerfilesFromTree(ctx, r.Cfg.Org, baseRepo.Name, treeEntries)
			entry.Files["dockerfile"] = append(entry.Files["dockerfile"], files...)

			manifests := r.FetchDependencyfilesFromTree(ctx, r.Cfg.Org, baseRepo.Name, treeEntries)
			entry.Files["dependencies"] = append(entry.Files["dependencies"], manifests...)
		}
	} else if shouldSearchDeepForManifests(entry) {
		slog.Info("Ikke monorepo, men ingen rot-manifester funnet, søker i underkataloger", "repo", baseRepo.FullName)

		treeEntries, treeErr = r.fetchRepoTreeREST(ctx, r.Cfg.Org, baseRepo.Name)
		if treeErr != nil {
			slog.Warn("Klarte ikke hente repo-tree", "repo", baseRepo.FullName, "error", treeErr)
		}

		if treeEntries != nil {
			manifests := r.FetchDependencyfilesFromTree(ctx, r.Cfg.Org, baseRepo.Name, treeEntries)
			entry.Files["dependencies"] = append(entry.Files["dependencies"], manifests...)
		}
	}

	// Analyze lockfile pairings now that all files have been fetched
	entry.Repo.LockfilePairings = parser.DetectLockfilePairings(entry.Files)
	entry.Repo.HasCompleteLockfiles = parser.HasCompleteLockfiles(entry.Repo.LockfilePairings)
	entry.Repo.Lockfile_pair_count = len(entry.Repo.LockfilePairings)

	return entry, nil
}

func DoRequestWithRateLimit(ctx context.Context, method, url, token string, body []byte, out interface{}) error {
	for {
		slog.Info("Henter URL", "url", url)

		apiCallCounter.Add(1)

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("advarsel: klarte ikke å lukke body: %v", err)
			}
		}()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				wait := time.Until(time.Unix(ts, 0)) + time.Second
				slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
				time.Sleep(wait)
				continue
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			slog.Error("GitHub-feil", "status", resp.StatusCode, "body", string(bodyBytes))
			return fmt.Errorf("GitHub API-feil: status %d – %s", resp.StatusCode, string(bodyBytes))
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}

func (r *RepoFetcher) fetchSBOM(ctx context.Context, owner, repo string) map[string]interface{} {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/dependency-graph/sbom", owner, repo)

	token, err := r.GetAuthToken(ctx)
	if err != nil {
		slog.Warn("Kunne ikke hente auth token for SBOM", "repo", owner+"/"+repo, "error", err)
		return nil
	}

	var sbom map[string]interface{}
	err = doRequestWithRateLimitAndOptional404(ctx, "GET", url, token, nil, &sbom)
	if err != nil {
		slog.Warn("SBOM-kall feilet", "repo", owner+"/"+repo, "error", err)
		return nil
	}
	return sbom
}

func doRequestWithRateLimitAndOptional404(ctx context.Context, method, url, token string, body []byte, out interface{}) error {
	for {
		apiCallCounter.Add(1)

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("advarsel: klarte ikke å lukke body: %v", err)
			}
		}()

		if rl := resp.Header.Get("X-RateLimit-Remaining"); rl == "0" {
			reset := resp.Header.Get("X-RateLimit-Reset")
			if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
				wait := time.Until(time.Unix(ts, 0)) + time.Second
				slog.Warn("Rate limit nådd", "venter", wait.Truncate(time.Second))
				time.Sleep(wait)
				continue
			}
		}

		// 404 is acceptable for SBOM - it just means no SBOM is available
		if resp.StatusCode == 404 {
			slog.Info("SBOM ikke tilgjengelig (404)", "url", url)
			return nil
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			slog.Error("GitHub-feil", "status", resp.StatusCode, "body", string(bodyBytes))
			return fmt.Errorf("GitHub API-feil: status %d – %s", resp.StatusCode, string(bodyBytes))
		}

		return json.NewDecoder(resp.Body).Decode(out)
	}
}

func ParseRepoData(data map[string]interface{}, baseRepo models.RepoMeta) *models.RepoEntry {
	repoData, ok := data["repository"].(map[string]interface{})
	if !ok {
		slog.Warn("Mangler 'repository'-data i GraphQL-response")
		return nil
	}

	updatedRepo := baseRepo
	updatedRepo.Readme = ExtractReadme(repoData)
	updatedRepo.Security = ExtractSecurity(repoData)

	return &models.RepoEntry{
		Repo:      updatedRepo,
		Languages: ExtractLanguages(repoData),
		Files:     ExtractFiles(repoData),
		CIConfig:  ExtractCI(repoData),
	}
}

func IsMonorepoCandidate(entry *models.RepoEntry) bool {
	langs := 0
	for lang := range entry.Languages {
		switch lang {
		case "Go", "Java", "Python", "JavaScript", "TypeScript", "Rust":
			langs++
		}
	}

	hasMatrix := false
	for _, ci := range entry.CIConfig {
		if strings.Contains(ci.Content, "matrix:") {
			hasMatrix = true
			break
		}
	}

	hasSecuritySignals := entry.Repo.Security["has_codeql"] || entry.Repo.Security["has_dependabot"]

	noDockerfiles := len(entry.Files["dockerfile"]) == 0

	return noDockerfiles && (langs > 0 || hasMatrix || hasSecuritySignals)
}

// shouldSearchDeepForManifests implements the hybrid approach:
// Only search subdirectories if no root-level manifests exist but the repo has code
func shouldSearchDeepForManifests(entry *models.RepoEntry) bool {
	// Check if we have any root-level dependency files
	hasRootManifests := len(entry.Files["dependencies"]) > 0

	// If we have root manifests, no need to search deeper
	if hasRootManifests {
		return false
	}

	// No root manifests - check if repo has code
	hasCode := len(entry.Languages) > 0

	// Search deeper only if there's code but no root manifests
	return hasCode
}

func ExtractLanguages(data map[string]interface{}) map[string]int {
	langs := map[string]int{}

	if langsData, ok := data["languages"].(map[string]interface{}); ok {
		if edges, ok := langsData["edges"].([]interface{}); ok {
			for _, edgeRaw := range edges {
				edge, ok := edgeRaw.(map[string]interface{})
				if !ok {
					continue
				}

				// node["name"]
				var name string
				if node, ok := edge["node"].(map[string]interface{}); ok {
					name, _ = node["name"].(string)
				}

				// size
				var size int
				if s, ok := edge["size"].(float64); ok {
					size = int(s)
				}

				if name != "" && size > 0 {
					langs[name] = size
				}
			}
		}
	}
	return langs
}

func ExtractFiles(data map[string]interface{}) map[string][]models.FileEntry {
	files := map[string][]map[string]string{}

	// Dependency files
	if deps, ok := data["dependencies"].(map[string]interface{}); ok {
		if entries, ok := deps["entries"].([]interface{}); ok {
			for _, raw := range entries {
				entry, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := entry["name"].(string)
				lowerName := strings.ToLower(name)

				// Check if this is a file we want to extract
				var fileType string
				if strings.Contains(lowerName, "dockerfile") {
					fileType = lowerName
				} else if isDependencyfile(name) {
					fileType = "dependencies"
				} else {
					continue
				}

				var content string
				if obj, ok := entry["object"].(map[string]interface{}); ok {
					if text, ok := obj["text"].(string); ok {
						content = text
					}
				}

				if content != "" {
					files[fileType] = append(files[fileType], map[string]string{
						"path":    name,
						"content": content,
					})
				}
			}
		}
	}
	return ConvertFiles(files)
}

// isDependencyfile checks if a filename is a dependency file we care about
// Uses the parser package's ecosystem definitions as single source of truth
func isDependencyfile(filename string) bool {
	files := parser.GetAllDependencyfileNames()
	for _, f := range files {
		if filename == f {
			return true
		}
	}
	return false
}

func ExtractCI(data map[string]interface{}) []models.FileEntry {
	ci := []map[string]string{}
	// CI config
	if workflows, ok := data["workflows"].(map[string]interface{}); ok {
		if entries, ok := workflows["entries"].([]interface{}); ok {
			for _, raw := range entries {
				entry, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := entry["name"].(string)

				// Hent .object.text hvis det finnes og er string
				var content string
				if obj, ok := entry["object"].(map[string]interface{}); ok {
					if text, ok := obj["text"].(string); ok {
						content = text
					}
				}

				// Bare legg til hvis det finnes
				if content != "" {
					ci = append(ci, map[string]string{
						"path":    ".github/workflows/" + name,
						"content": content,
					})
				}
			}
		}
	}
	return ConvertToFileEntries(ci)
}

func ExtractSecurity(data map[string]interface{}) map[string]bool {
	security := map[string]bool{}
	security["has_security_md"] = data["SECURITY"] != nil
	security["has_dependabot"] = data["dependabot"] != nil
	security["has_codeql"] = data["codeql"] != nil
	return security
}

func ExtractReadme(data map[string]interface{}) string {
	if val, ok := data["README"].(map[string]interface{}); ok {
		if text, ok := val["text"].(string); ok {
			return text
		}
	}
	return ""
}

func BuildRepoQuery(owner string, name string) string {
	query := fmt.Sprintf(`
	{
		repository(owner: "%s", name: "%s") {
			defaultBranchRef {
				name
			}
			README: object(expression: "HEAD:README.md") {
				... on Blob {
					text
				}
			}
			SECURITY: object(expression: "HEAD:SECURITY.md") {
				... on Blob {
					text
				}
			}
			dependabot: object(expression: "HEAD:.github/dependabot.yml") {
				... on Blob {
					text
				}
			}
			codeql: object(expression: "HEAD:.github/codeql.yml") {
				... on Blob {
					text
				}
			}
			workflows: object(expression: "HEAD:.github/workflows") {
				... on Tree {
					entries {
						name
						object {
							... on Blob {
								text
							}
						}
					}
				}
			}
			dependencies: object(expression: "HEAD:") {
				... on Tree {
					entries {
						name
						object {
							... on Blob {
								text
							}
						}
					}
				}
			}
			languages(first: 10) {
				edges {
					size
					node {
						name
					}
				}
			}
		}
	}`, owner, name)
	return query
}

func ConvertToFileEntries(entries []map[string]string) []models.FileEntry {
	var result []models.FileEntry
	for _, e := range entries {
		result = append(result, models.FileEntry{
			Path:    e["path"],
			Content: e["content"],
		})
	}
	return result
}

func ConvertFiles(input map[string][]map[string]string) map[string][]models.FileEntry {
	out := map[string][]models.FileEntry{}
	for k, v := range input {
		out[k] = ConvertToFileEntries(v)
	}
	return out
}

// fetchRepoTreeREST fetches the git tree structure from GitHub REST API
func (r *RepoFetcher) fetchRepoTreeREST(ctx context.Context, owner, repo string) ([]TreeEntry, error) {
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)

	var tree struct {
		Tree      []TreeEntry `json:"tree"`
		Truncated bool        `json:"truncated"`
	}

	token, err := r.GetAuthToken(ctx)
	if err != nil {
		slog.Warn("Kunne ikke hente auth token for git tree", "repo", owner+"/"+repo)
		return nil, fmt.Errorf("could not get auth token: %w", err)
	}

	err = DoRequestWithRateLimit(ctx, "GET", treeURL, token, nil, &tree)
	if err != nil {
		slog.Warn("Kunne ikke hente repo tree", "repo", owner+"/"+repo)
		return nil, fmt.Errorf("could not fetch repo tree: %w", err)
	}

	if tree.Truncated {
		slog.Warn("Git tree was truncated (too large)", "repo", owner+"/"+repo)
	}

	return tree.Tree, nil
}

// FetchDockerfilesFromTree extracts Dockerfiles from a provided git tree structure
func (r *RepoFetcher) FetchDockerfilesFromTree(ctx context.Context, owner, repo string, treeEntries []TreeEntry) []models.FileEntry {
	var results []models.FileEntry

	for _, entry := range treeEntries {
		if entry.Type != "blob" {
			continue
		}
		if !strings.Contains(strings.ToLower(entry.Path), "dockerfile") {
			continue
		}
		if entry.Size == 0 {
			continue
		}
		content := r.fetchFileContent(ctx, owner, repo, entry.Path)
		if content != "" {
			results = append(results, models.FileEntry{
				Path:    entry.Path,
				Content: content,
			})
		}
	}
	return results
}

// FetchDependencyfilesFromTree extracts dependency files from a provided git tree structure
func (r *RepoFetcher) FetchDependencyfilesFromTree(ctx context.Context, owner, repo string, treeEntries []TreeEntry) []models.FileEntry {
	var results []models.FileEntry

	dependencyfileNames := parser.GetAllDependencyfileNames()
	fileMap := make(map[string]bool)
	for _, name := range dependencyfileNames {
		fileMap[name] = true
	}

	for _, entry := range treeEntries {
		if entry.Type != "blob" {
			continue
		}

		// Extract just the filename from the path
		pathParts := strings.Split(entry.Path, "/")
		filename := pathParts[len(pathParts)-1]

		// Check if this filename is a dependency file we care about
		if !fileMap[filename] {
			continue
		}

		// Skip root-level files (they were already fetched via GraphQL)
		if len(pathParts) == 1 {
			continue
		}

		// Skip empty files, should they exist in the tree
		if entry.Size == 0 {
			continue
		}
		results = append(results, models.FileEntry{
			Path:    entry.Path,
			Content: "",
			// Content is not interesting for now, maybe in a later feature.
		})
		slog.Debug("Fant dependency file i underkatalog", "repo", owner+"/"+repo, "path", entry.Path)
	}

	slog.Info("Hentet dependency files fra underkataloger", "repo", owner+"/"+repo, "count", len(results))
	return results
}

func (r *RepoFetcher) fetchFileContent(ctx context.Context, owner, repo, path string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)

	token, err := r.GetAuthToken(ctx)
	if err != nil {
		slog.Warn("Kunne ikke hente auth token for filinnhold", "repo", owner+"/"+repo, "path", path, "error", err)
		return ""
	}

	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	err = DoRequestWithRateLimit(ctx, "GET", url, token, nil, &file)
	if err != nil {
		slog.Warn("Klarte ikke hente filinnhold", "repo", owner+"/"+repo, "path", path, "error", err)
		return ""
	}

	if file.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(file.Content)
		if err == nil {
			return string(decoded)
		}
	}
	return ""
}

// GetAuthToken returns the appropriate authentication token based on configuration
func (r *RepoFetcher) GetAuthToken(ctx context.Context) (string, error) {
	if r.Cfg.Feature_GitHubApp && r.Cfg.GitHubAppConfig != nil {
		// Use GitHub App authentication
		token, err := CreateGitHubAppToken(ctx, r.Cfg.GitHubAppConfig)
		if err != nil {
			return "", fmt.Errorf("failed to create GitHub App token: %w", err)
		}
		return token, nil
	}

	// Use personal access token
	if r.Cfg.Token == "" {
		return "", fmt.Errorf("no authentication token available")
	}
	return r.Cfg.Token, nil
}
