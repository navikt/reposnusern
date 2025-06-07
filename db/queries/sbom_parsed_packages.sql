-- name: InsertParsedSBOM :exec
INSERT INTO sbom_parsed_packages (
  repo_id, name, pkg_group, version, type, path
) VALUES ($1, $2, $3, $4, $5, $6);