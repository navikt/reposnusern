-- name: InsertLanguage :exec
INSERT INTO repo_languages (
    repo_id, language, bytes
) VALUES (?, ?, ?);
