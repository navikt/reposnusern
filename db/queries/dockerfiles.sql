-- name: InsertOrUpdateDockerfile :one
INSERT INTO dockerfiles (
  repo_id, hentet_dato, full_name, path, content,
  base_image, base_tag, uses_latest_tag,
  has_user_instruction, has_copy_sensitive, has_package_installs,
  uses_multistage, has_healthcheck, uses_add_instruction,
  has_label_metadata, has_expose, has_entrypoint_or_cmd,
  installs_curl_or_wget, installs_build_tools, has_apt_get_clean,
  world_writable, has_secrets_in_env_or_arg
)
VALUES (
  $1, $2, $3, $4, $5,
  $6, $7, $8,
  $9, $10, $11,
  $12, $13, $14,
  $15, $16, $17,
  $18, $19, $20,
  $21, $22
)
ON CONFLICT (repo_id, hentet_dato, path) DO UPDATE SET
  full_name = EXCLUDED.full_name,
  content = EXCLUDED.content,
  base_image = EXCLUDED.base_image,
  base_tag = EXCLUDED.base_tag,
  uses_latest_tag = EXCLUDED.uses_latest_tag,
  has_user_instruction = EXCLUDED.has_user_instruction,
  has_copy_sensitive = EXCLUDED.has_copy_sensitive,
  has_package_installs = EXCLUDED.has_package_installs,
  uses_multistage = EXCLUDED.uses_multistage,
  has_healthcheck = EXCLUDED.has_healthcheck,
  uses_add_instruction = EXCLUDED.uses_add_instruction,
  has_label_metadata = EXCLUDED.has_label_metadata,
  has_expose = EXCLUDED.has_expose,
  has_entrypoint_or_cmd = EXCLUDED.has_entrypoint_or_cmd,
  installs_curl_or_wget = EXCLUDED.installs_curl_or_wget,
  installs_build_tools = EXCLUDED.installs_build_tools,
  has_apt_get_clean = EXCLUDED.has_apt_get_clean,
  world_writable = EXCLUDED.world_writable,
  has_secrets_in_env_or_arg = EXCLUDED.has_secrets_in_env_or_arg
RETURNING id;
