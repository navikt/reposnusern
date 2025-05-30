-- name: InsertDockerfile :exec
INSERT INTO dockerfiles (
    repo_id, full_name, content
) VALUES (?, ?, ?);
