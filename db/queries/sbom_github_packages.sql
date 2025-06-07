-- name: InsertGithubSBOM :exec
INSERT INTO sbom_github_packages (
  repo_id, name, version, license, purl
) VALUES ($1, $2, $3, $4, $5);