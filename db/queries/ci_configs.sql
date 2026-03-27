-- name: InsertOrUpdateCIConfig :exec
INSERT INTO ci_configs (
  repo_id, hentet_dato, path, content,
  uses_npm_install,
  uses_npm_ci_without_ignore_scripts,
  uses_yarn_install_without_frozen,
  uses_pip_install_without_no_cache,
  uses_pip_install_without_hashes,
  uses_curl_bash_pipe,
  uses_sudo
) VALUES (
  $1, $2, $3, $4,
  $5, $6, $7, $8, $9, $10, $11
)
ON CONFLICT (repo_id, hentet_dato, path) DO UPDATE SET
  content = EXCLUDED.content,
  uses_npm_install = EXCLUDED.uses_npm_install,
  uses_npm_ci_without_ignore_scripts = EXCLUDED.uses_npm_ci_without_ignore_scripts,
  uses_yarn_install_without_frozen = EXCLUDED.uses_yarn_install_without_frozen,
  uses_pip_install_without_no_cache = EXCLUDED.uses_pip_install_without_no_cache,
  uses_pip_install_without_hashes = EXCLUDED.uses_pip_install_without_hashes,
  uses_curl_bash_pipe = EXCLUDED.uses_curl_bash_pipe,
  uses_sudo = EXCLUDED.uses_sudo;