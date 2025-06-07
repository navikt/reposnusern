package fetcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

func GetDetailsActiveReposGraphQL(org, token string, repos []map[string]interface{}) OrgRepos {
	allData := OrgRepos{
		Org:   org,
		Repos: []map[string]interface{}{},
	}

	for i, r := range repos {
		fullName := r["full_name"].(string)
		if r["archived"].(bool) {
			continue
		}
		slog.Info("Bearbeider repo (GraphQL)", "index", i+1, "total", len(repos), "repo", fullName)

		parts := strings.Split(fullName, "/")
		owner, name := parts[0], parts[1]

		// Hent metadata via GraphQL
		data := FetchRepoGraphQL(owner, name, token)
		if data == nil {
			slog.Warn("Hoppet over repo pga. tomt GraphQL-svar", "repo", fullName)
			continue
		}

		// Strukturér resultatet
		result := map[string]interface{}{
			"repo":      r,
			"languages": data["languages"],
			"files":     data["files"],
			"security":  data["security"],
			"ci_config": data["ci_config"],
			"readme":    data["readme"],
			"sbom":      data["sbom"],
		}

		allData.Repos = append(allData.Repos, result)
	}
	return allData
}

func FetchRepoGraphQL(owner, name, token string) map[string]interface{} {
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

	reqBody := map[string]string{"query": query}
	bodyBytes, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://api.github.com/graphql", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		slog.Error("GraphQL kall feilet", "repo", owner+"/"+name, "error", err)
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	data := result["data"].(map[string]interface{})["repository"].(map[string]interface{})

	// Pakker relevant ut
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

	files := map[string][]map[string]string{}
	ci := []map[string]string{}
	security := map[string]bool{}

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

				// Hent .object.text hvis det finnes
				var content string
				if obj, ok := entry["object"].(map[string]interface{}); ok {
					if text, ok := obj["text"].(string); ok {
						content = text
					}
				}

				// Legg til dependency file
				if IsDependencyFile(lowerName) && content != "" {
					files[lowerName] = append(files[lowerName], map[string]string{
						"path":    name,
						"content": content,
					})
				}

				// Legg til dockerfile
				if strings.HasPrefix(lowerName, "dockerfile") && content != "" {
					files[lowerName] = append(files[lowerName], map[string]string{
						"path":    name,
						"content": content,
					})
				}
			}
		}
	}

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

	// Security metadata
	security["has_security_md"] = data["SECURITY"] != nil
	security["has_dependabot"] = data["dependabot"] != nil
	security["has_codeql"] = data["codeql"] != nil

	readme := ""
	if val, ok := data["README"].(map[string]interface{}); ok {
		if text, ok := val["text"].(string); ok {
			readme = text
		}
	}

	if result["errors"] != nil {
		slog.Warn("GraphQL-resultat har feil", "repo", owner+"/"+name, "errors", result["errors"])
	}

	sbom := FetchSBOM(owner, name, token)
	return map[string]interface{}{
		"languages": langs,
		"files":     files,
		"ci_config": ci,
		"security":  security,
		"readme":    readme,
		"sbom":      sbom,
	}
}

func FetchSBOM(owner, repo, token string) map[string]interface{} {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/dependency-graph/sbom", owner, repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("SBOM-kall feilet", "repo", owner+"/"+repo, "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("Ingen SBOM tilgjengelig", "repo", owner+"/"+repo, "status", resp.StatusCode)
		return nil
	}

	var sbom map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&sbom); err != nil {
		slog.Error("Kunne ikke parse SBOM", "repo", owner+"/"+repo, "error", err)
		return nil
	}
	return sbom
}
