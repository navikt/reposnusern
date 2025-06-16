-- name: InsertDockerfileFeatures :exec
INSERT INTO dockerfile_features (
  dockerfile_id,
  hentet_dato,
  base_image,
  base_tag,
  uses_latest_tag,
  has_user_instruction,
  has_copy_sensitive,
  has_package_installs,
  uses_multistage,
  has_healthcheck,
  uses_add_instruction,
  has_label_metadata,
  has_expose,
  has_entrypoint_or_cmd,
  installs_curl_or_wget,
  installs_build_tools,
  has_apt_get_clean,
  world_writable,
  has_secrets_in_env_or_arg
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
);