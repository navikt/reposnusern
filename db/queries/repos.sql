-- name: InsertRepo :exec
INSERT INTO repos (
  id, hentet_dato,
  name, full_name, description, stars, forks, archived, private, is_fork,
  language, size_mb, updated_at, pushed_at, created_at, html_url, topics,
  visibility, license, open_issues, languages_url,
  has_security_md, has_dependabot, has_codeql, readme_content
) VALUES (
  $1, $2,
  $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17,
  $18, $19, $20, $21,
  $22, $23, $24, $25
);