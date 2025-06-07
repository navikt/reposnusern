-- name: InsertDependencyFile :exec
INSERT INTO dependency_files (
  repo_id, path
) VALUES ($1, $2);