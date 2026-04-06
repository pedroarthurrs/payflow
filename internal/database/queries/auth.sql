-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetAPIKeyByHash :one
SELECT * FROM credentials
WHERE key_hash = $1
  AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW());

-- name: UpdateAPIKeyLastUsed :exec
UPDATE credentials SET last_used_at = NOW() WHERE id = $1;
