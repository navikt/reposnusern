-- name: InsertOrUpdateCIConfig :exec
INSERT INTO ci_configs (
  repo_id, hentet_dato, path, content
) VALUES (
  $1, $2, $3, $4
)
ON CONFLICT (repo_id, hentet_dato, path) DO UPDATE SET
  content = EXCLUDED.content;