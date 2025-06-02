-- name: InsertSecurityFeatures :exec
INSERT INTO repo_security_features (
  repo_id, has_security_md, has_dependabot, has_codeql
) VALUES ($1, $2, $3, $4);