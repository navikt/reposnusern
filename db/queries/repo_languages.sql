-- name: InsertRepoLanguage :exec
INSERT INTO repo_languages (
  repo_id, hentet_dato, language, bytes
) VALUES ($1, $2, $3, $4);