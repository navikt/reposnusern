-- name: InsertCIConfig :exec
INSERT INTO ci_configs (
  repo_id, path, content
) VALUES ($1, $2, $3);
