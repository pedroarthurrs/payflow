-- name: CreateCredential :one
INSERT INTO credentials (organization_id, name, key_hash, key_prefix, environment, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, organization_id, name, key_prefix, environment, last_used_at, expires_at, created_at;

-- name: GetCredentialByID :one
SELECT id, organization_id, name, key_prefix, environment, last_used_at, expires_at, created_at
FROM credentials
WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: ListCredentials :many
SELECT id, organization_id, name, key_prefix, environment, last_used_at, expires_at, created_at
FROM credentials
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: SoftDeleteCredential :exec
UPDATE credentials
SET deleted_at = NOW()
WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL;
