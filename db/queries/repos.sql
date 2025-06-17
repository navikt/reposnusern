-- name: InsertOrUpdateRepo :exec
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
)
ON CONFLICT (id, hentet_dato) DO UPDATE SET
  name = EXCLUDED.name,
  full_name = EXCLUDED.full_name,
  description = EXCLUDED.description,
  stars = EXCLUDED.stars,
  forks = EXCLUDED.forks,
  archived = EXCLUDED.archived,
  private = EXCLUDED.private,
  is_fork = EXCLUDED.is_fork,
  language = EXCLUDED.language,
  size_mb = EXCLUDED.size_mb,
  updated_at = EXCLUDED.updated_at,
  pushed_at = EXCLUDED.pushed_at,
  created_at = EXCLUDED.created_at,
  html_url = EXCLUDED.html_url,
  topics = EXCLUDED.topics,
  visibility = EXCLUDED.visibility,
  license = EXCLUDED.license,
  open_issues = EXCLUDED.open_issues,
  languages_url = EXCLUDED.languages_url,
  has_security_md = EXCLUDED.has_security_md,
  has_dependabot = EXCLUDED.has_dependabot,
  has_codeql = EXCLUDED.has_codeql,
  readme_content = EXCLUDED.readme_content;
