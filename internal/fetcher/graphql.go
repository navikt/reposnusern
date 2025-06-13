package fetcher

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jonmartinstorm/reposnusern/internal/models"
)

func (g *GitHubAPIClient) Fetch(owner, name, token string, baseRepo models.RepoMeta) *models.RepoEntry {
	return FetchRepoGraphQL(owner, name, token, baseRepo)
}

func FetchRepoGraphQL(owner, name, token string, baseRepo models.RepoMeta) *models.RepoEntry {
	query := BuildRepoQuery(owner, name)

	reqBody := map[string]string{"query": query}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		slog.Error("Kunne ikke serialisere GraphQL-request", "repo", owner+"/"+name, "error", err)
		return nil
	}

	var result map[string]interface{}
	err = DoRequestWithRateLimit("POST", "https://api.github.com/graphql", token, bodyBytes, &result)
	if err != nil {
		slog.Error("GraphQL-kall feilet", "repo", owner+"/"+name, "error", err)
		return nil
	}

	if errs, ok := result["errors"]; ok {
		slog.Warn("GraphQL-resultat har feil", "repo", owner+"/"+name, "errors", errs)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok || data["repository"] == nil {
		slog.Warn("Ingen repository-data fra GraphQL", "repo", owner+"/"+name)
		return nil
	}

	sbom := FetchSBOM(owner, name, token)
	entry := ParseRepoData(data, baseRepo)

	entry.SBOM = sbom
	return entry
}

func FetchSBOM(owner, repo, token string) map[string]interface{} {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/dependency-graph/sbom", owner, repo)

	var sbom map[string]interface{}
	err := DoRequestWithRateLimit("GET", url, token, nil, &sbom)
	if err != nil {
		slog.Warn("SBOM-kall feilet", "repo", owner+"/"+repo, "error", err)
		return nil
	}
	return sbom
}

func ParseRepoData(data map[string]interface{}, baseRepo models.RepoMeta) *models.RepoEntry {
	repoData, ok := data["repository"].(map[string]interface{})
	if !ok {
		slog.Warn("Mangler 'repository'-data i GraphQL-response")
		return nil
	}
	return &models.RepoEntry{
		Repo:      baseRepo,
		Languages: ExtractLanguages(repoData),
		Files:     ExtractFiles(repoData),
		CIConfig:  ExtractCI(repoData),
		Readme:    ExtractReadme(repoData),
		Security:  ExtractSecurity(repoData),
	}
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

				if !strings.Contains(lowerName, "dockerfile") {
					continue
				}

				var content string
				if obj, ok := entry["object"].(map[string]interface{}); ok {
					if text, ok := obj["text"].(string); ok {
						content = text
					}
				}

				if content != "" {
					files[lowerName] = append(files[lowerName], map[string]string{
						"path":    name,
						"content": content,
					})
				}
			}
		}
	}
	return ConvertFiles(files)
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
