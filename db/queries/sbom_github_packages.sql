-- name: InsertOrUpdateGithubSBOM :exec
INSERT INTO sbom_github_packages (
  repo_id, hentet_dato, name, version, license, purl
) VALUES (
  $1, $2, $3, $4, $5, $6
)
ON CONFLICT (repo_id, hentet_dato, name, version) DO UPDATE SET
  license = EXCLUDED.license,
  purl = EXCLUDED.purl;