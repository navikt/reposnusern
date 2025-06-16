-- name: InsertCIConfig :exec
INSERT INTO ci_configs (
  repo_id, hentet_dato, path, content
) VALUES ($1, $2, $3, $4);