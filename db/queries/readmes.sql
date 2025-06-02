-- name: InsertReadme :exec
INSERT INTO readmes (
  repo_id, content
) VALUES ($1, $2);