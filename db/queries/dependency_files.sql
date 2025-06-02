-- name: InsertDependencyFile :exec
INSERT INTO dependency_files (
  repo_id, path, content
) VALUES ($1, $2, $3);