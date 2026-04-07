-- name: GetUserByEmailAndOrganization :one
SELECT * FROM users WHERE email = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (organization_id, name, email, password, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListUsersByOrganization :many
SELECT * FROM users WHERE organization_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: UpdateUser :one
UPDATE users SET name = $1, role = $2, updated_at = NOW()
WHERE id = $3 AND organization_id = $4 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;
