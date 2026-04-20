package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// GraphQLEndpoint is the GitHub GraphQL API URL. Can be overridden in tests.
var GraphQLEndpoint = "https://api.github.com/graphql"

// HttpClient is the HTTP client used for all GitHub API requests.
// It can be overridden in tests. The default client has a 30-second timeout
// to prevent requests from hanging indefinitely.
var HttpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// RetryBackoff returns the wait duration before retry attempt n (1-indexed).
// Can be overridden in tests to speed up retry logic.
var RetryBackoff = func(attempt int) time.Duration {
	return time.Duration(1<<uint(attempt-1)) * time.Second
}

func formatWaitForLog(wait time.Duration) string {
	if wait >= time.Second {
		return wait.Truncate(time.Second).String()
	}

	return wait.String()
}

func formatResetAtForLog(resetAt time.Time) string {
	if resetAt.IsZero() {
		return ""
	}

	return resetAt.Local().Format(time.RFC3339)
}

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

// FetchRepoGraphQL fetches and enriches one repo through the GraphQL resource bucket.
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

	for rateLimitAttempt := 1; ; rateLimitAttempt++ {
		var result map[string]interface{}
		headers, err := doRequestWithHeaders(ctx, RateLimitResourceGraphQL, "POST", GraphQLEndpoint, token, bodyBytes, &result, false)
		if err != nil {
			slog.Error("GraphQL-kall feilet", "repo", r.Cfg.Org+"/"+baseRepo.Name, "error", err)
			return nil, err
		}

		if errs, ok := result["errors"]; ok {
			if isGraphQLRateLimitError(errs) {
				wait := graphQLRateLimitWait(headers, rateLimitAttempt)
				blockResult := SharedRateLimiter.BlockFor(RateLimitResourceGraphQL, wait)
				switch {
				case blockResult.StartedNewBlock:
					slog.Warn("GraphQL rate limit nådd", "repo", r.Cfg.Org+"/"+baseRepo.Name, "venter", formatWaitForLog(blockResult.RemainingCooldown), "reset_at", formatResetAtForLog(blockResult.BlockedUntil))
				case blockResult.ExtendedBlock:
					slog.Warn("GraphQL rate limit forlenget", "repo", r.Cfg.Org+"/"+baseRepo.Name, "venter", formatWaitForLog(blockResult.RemainingCooldown), "reset_at", formatResetAtForLog(blockResult.BlockedUntil))
				}
				continue
			}
			return nil, fmt.Errorf("GraphQL returnerte feil for %s/%s: %v", r.Cfg.Org, baseRepo.Name, errs)
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

		entry = r.needsFiletreeFetching(ctx, baseRepo, entry)

		// Analyze lockfile pairings now that all files have been fetched
		entry.Repo.LockfilePairings = parser.DetectLockfilePairings(entry.Files)
		entry.Repo.HasCompleteLockfiles = parser.HasCompleteLockfiles(entry.Repo.LockfilePairings)
		entry.Repo.Lockfile_pair_count = len(entry.Repo.LockfilePairings)

		return entry, nil
	}
}

// Checks if we should fetch the entire filetree
func (r *RepoFetcher) needsFiletreeFetching(ctx context.Context, baseRepo models.RepoMeta, entry *models.RepoEntry) *models.RepoEntry {

	if IsMonorepoCandidate(entry) {
		slog.Debug("Monorepo-kandidat – henter dype Dockerfiles og dependencyfiles", "repo", baseRepo.FullName)
		updatedEntry := r.fetchAndParseFiletree(ctx, baseRepo, entry)
		return updatedEntry

	} else if shouldSearchDeepForManifests(entry) {
		slog.Debug("Ikke monorepo, men ingen rot-manifester funnet, søker i underkataloger", "repo", baseRepo.FullName)
		updatedEntry := r.fetchAndParseFiletree(ctx, baseRepo, entry)
		return updatedEntry
	}
	return entry
}

// Extracts all relevant files from the tree, since we fetch them regardless
func (r *RepoFetcher) fetchAndParseFiletree(ctx context.Context, baseRepo models.RepoMeta, entry *models.RepoEntry) *models.RepoEntry {
	// Fetch tree once if we need to search deeper for files
	// TODO: Should combine the "fetchDockerfileFromTree and fetchDependencyfilesFromTree into one function to avoid duplication!
	var treeEntries []TreeEntry
	var treeErr error

	treeEntries, treeErr = r.fetchRepoTreeREST(ctx, r.Cfg.Org, baseRepo.Name)
	if treeErr != nil {
		slog.Warn("Klarte ikke hente repo-tree", "repo", baseRepo.FullName, "error", treeErr)
	}
	if treeEntries != nil {
		files := r.FetchDockerfilesFromTree(ctx, r.Cfg.Org, baseRepo.Name, treeEntries)
		entry.Files["dockerfile"] = append(entry.Files["dockerfile"], files...)

		manifests := r.FetchDependencyfilesFromTree(ctx, r.Cfg.Org, baseRepo.Name, treeEntries)
		entry.Files["dependencies"] = append(entry.Files["dependencies"], manifests...)
		return entry
	}
	return entry
}

// sleepWithContext sleeps for duration d but returns early with ctx.Err() if the
// context is cancelled. This allows rate-limit and retry waits to be interrupted
// by a SIGTERM signal, which is essential for graceful Kubernetes job shutdown.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// MaxAttempts is the total number of attempts (including the initial call) for
// transient errors (5xx responses and network failures). E.g. MaxAttempts=3
// means: 1st call → wait 1s → 2nd call → wait 2s → 3rd call → give up.
// Rate-limit retries are unlimited and do not count against this total.
const MaxAttempts = 3

// doRequest runs a GitHub request through the shared per-resource limiter and retry policy.
// Set allow404=true for optional endpoints where 404 means "not available".
func doRequest(ctx context.Context, resource RateLimitResource, method, url, token string, body []byte, out interface{}, allow404 bool) error {
	_, err := doRequestWithHeaders(ctx, resource, method, url, token, body, out, allow404)
	return err
}

// doRequestWithHeaders behaves like doRequest but also returns the response headers.
func doRequestWithHeaders(ctx context.Context, resource RateLimitResource, method, url, token string, body []byte, out interface{}, allow404 bool) (http.Header, error) {
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := SharedRateLimiter.Wait(ctx, resource); err != nil {
			return nil, err
		}

		apiCallCounter.Add(1)

		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/vnd.github+json")
		if method == "POST" {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := HttpClient.Do(req)
		if err != nil {
			if attempt >= MaxAttempts {
				return nil, err
			}
			wait := RetryBackoff(attempt)
			slog.Warn("Nettverksfeil, prøver igjen", "forsøk", attempt, "venter", formatWaitForLog(wait), "error", err)
			if sleepErr := sleepWithContext(ctx, wait); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if wait, ok := rateLimitWait(resp.Header, resp.StatusCode); ok {
			_ = resp.Body.Close()
			blockResult := SharedRateLimiter.BlockFor(resource, wait)
			switch {
			case blockResult.StartedNewBlock:
				slog.Warn("Rate limit nådd", "ressurs", resource, "venter", formatWaitForLog(blockResult.RemainingCooldown), "reset_at", formatResetAtForLog(blockResult.BlockedUntil))
			case blockResult.ExtendedBlock:
				slog.Warn("Rate limit forlenget", "ressurs", resource, "venter", formatWaitForLog(blockResult.RemainingCooldown), "reset_at", formatResetAtForLog(blockResult.BlockedUntil))
			}
			attempt = 0 // reset transient counter; incremented to 1 at top of next iteration
			continue
		}

		if allow404 && resp.StatusCode == 404 {
			slog.Info("Ressurs ikke tilgjengelig (404)", "url", url)
			_ = resp.Body.Close()
			return nil, nil
		}

		if resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			if attempt >= MaxAttempts {
				return nil, fmt.Errorf("GitHub API-feil etter %d forsøk: status %d", MaxAttempts, resp.StatusCode)
			}
			wait := RetryBackoff(attempt)
			slog.Warn("Serverfeil, prøver igjen", "status", resp.StatusCode, "forsøk", attempt, "venter", formatWaitForLog(wait))
			if sleepErr := sleepWithContext(ctx, wait); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			slog.Error("GitHub-feil", "status", resp.StatusCode, "body", string(bodyBytes))
			return nil, fmt.Errorf("GitHub API-feil: status %d – %s", resp.StatusCode, string(bodyBytes))
		}

		err = json.NewDecoder(resp.Body).Decode(out)
		_ = resp.Body.Close()
		return resp.Header, err
	}
}

// DoRequestWithRateLimit issues a core REST request with shared rate-limit handling.
func DoRequestWithRateLimit(ctx context.Context, method, url, token string, body []byte, out interface{}) error {
	return doRequest(ctx, RateLimitResourceCore, method, url, token, body, out, false)
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

// doRequestWithRateLimitAndOptional404 is the core REST variant where 404 is non-fatal.
func doRequestWithRateLimitAndOptional404(ctx context.Context, method, url, token string, body []byte, out interface{}) error {
	return doRequest(ctx, RateLimitResourceCore, method, url, token, body, out, true)
}

func isGraphQLRateLimitError(errs interface{}) bool {
	errorList, ok := errs.([]interface{})
	if !ok || len(errorList) == 0 {
		return false
	}

	for _, rawErr := range errorList {
		errMap, ok := rawErr.(map[string]interface{})
		if !ok || !isRateLimitGraphQLErrorMap(errMap) {
			return false
		}
	}

	return true
}

func isRateLimitGraphQLErrorMap(errMap map[string]interface{}) bool {
	if strings.EqualFold(fmt.Sprint(errMap["type"]), "RATE_LIMIT") {
		return true
	}
	if strings.EqualFold(fmt.Sprint(errMap["code"]), "graphql_rate_limit") {
		return true
	}

	extensions, ok := errMap["extensions"].(map[string]interface{})
	if !ok {
		return false
	}

	code := fmt.Sprint(extensions["code"])
	return strings.EqualFold(code, "RATE_LIMIT") || strings.EqualFold(code, "graphql_rate_limit")
}

// graphQLRateLimitWait prefers server-provided wait hints, then falls back to retry backoff.
func graphQLRateLimitWait(headers http.Header, attempt int) time.Duration {
	if headers != nil {
		if wait, ok := retryAfterWait(headers.Get("Retry-After")); ok {
			return wait
		}
		reset := headers.Get("X-RateLimit-Reset")
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			wait := time.Until(time.Unix(ts, 0)) + time.Second
			if wait > 0 {
				return wait
			}
		}
	}

	return RetryBackoff(attempt)
}

// rateLimitWait derives a shared cooldown from a failed REST response.
func rateLimitWait(headers http.Header, statusCode int) (time.Duration, bool) {
	if headers == nil || statusCode < 400 {
		return 0, false
	}

	if wait, ok := retryAfterWait(headers.Get("Retry-After")); ok {
		return wait, true
	}

	if headers.Get("X-RateLimit-Remaining") != "0" {
		return 0, false
	}

	reset := headers.Get("X-RateLimit-Reset")
	ts, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		return 0, false
	}

	wait := time.Until(time.Unix(ts, 0)) + time.Second
	if wait <= 0 {
		return 0, false
	}
	return wait, true
}

// retryAfterWait parses Retry-After as either seconds or an HTTP date.
func retryAfterWait(retryAfter string) (time.Duration, bool) {
	if retryAfter == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		wait := time.Duration(seconds) * time.Second
		if wait > 0 {
			return wait, true
		}
		return 0, false
	}

	if ts, err := http.ParseTime(retryAfter); err == nil {
		wait := time.Until(ts)
		if wait > 0 {
			return wait, true
		}
	}

	return 0, false
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

// Extracts dockerfiles and dependencyfiles from graphql response
func ExtractFiles(data map[string]interface{}) map[string][]models.FileEntry {
	files := map[string][]map[string]string{}
	
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
				if isDockerfile(lowerName) {
					fileType = "dockerfile" 
				} else if isDependencyfile(lowerName) {
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

				if fileType == "dependencies" {
					files[fileType] = append(files[fileType], map[string]string{
						"path":    name,
						"content": "", // dependency files content is not interesting for now
					})
					continue
				}
				// For Dockerfiles, we want the content to analyze them later
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

// Heuristic for finding dockerfile by name
func isDockerfile(filename string) bool {
	return strings.Contains(filename, "dockerfile") && !strings.Contains(filename, "dockerignore");
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
		// Extract just the filename from the path
		pathParts := strings.Split(entry.Path, "/")
		filename := pathParts[len(pathParts)-1]

		// Skip root-level files (they were already fetched via GraphQL)
		if len(pathParts) == 1 {
			continue
		}
		if entry.Type != "blob" {
			continue
		}
		if entry.Size == 0 {
			continue
		}
		if !isDockerfile(filename) {
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

	for _, entry := range treeEntries {
		// Extract just the filename from the path
		pathParts := strings.Split(entry.Path, "/")
		filename := pathParts[len(pathParts)-1]

		// Skip root-level files (they were already fetched via GraphQL)
		if len(pathParts) == 1 {
			continue
		}
		if entry.Type != "blob" {
			continue
		}
		// Skip empty files, should they exist in the tree
		if entry.Size == 0 {
			continue
		}
		if parser.IsIgnoredPath(entry.Path) {
			slog.Debug("Skipping ignored dependency file", "repo", owner+"/"+repo, "path", entry.Path)
			continue
		}

		// Check if this filename is a dependency file we care about
		if !isDependencyfile(filename) {
			continue
		}

		results = append(results, models.FileEntry{
			Path:    entry.Path,
			Content: "",
			// Content is not interesting for now, maybe in a later feature.
		})
		slog.Debug("Fant dependency file i underkatalog", "repo", owner+"/"+repo, "path", entry.Path)
	}

	slog.Debug("Antall dependency files fra underkataloger", "repo", owner+"/"+repo, "count", len(results))
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
