-- name: InsertRepo :exec
INSERT INTO repos (
    id, name, full_name, description, stars, forks,
    archived, private, is_fork, language, size_mb,
    updated_at, pushed_at, created_at, html_url,
    topics, visibility, license, open_issues, languages_url
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);
