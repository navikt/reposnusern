-- name: InsertDockerfile :one
INSERT INTO dockerfiles (repo_id, full_name, path, content)
VALUES ($1, $2, $3, $4)
RETURNING id;