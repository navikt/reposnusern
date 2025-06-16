-- name: InsertDockerfile :one
INSERT INTO dockerfiles (repo_id, hentet_dato, full_name, path, content)
VALUES ($1, $2, $3, $4, $5)
RETURNING id;