-- name: InsertDockerfileFeatures :exec
INSERT INTO dockerfile_features (
  dockerfile_id, base_image, base_tag, uses_latest_tag,
  has_user_instruction, has_copy_sensitive, has_package_installs, uses_multistage
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);