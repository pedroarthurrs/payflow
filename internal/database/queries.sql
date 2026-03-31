-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1 LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY name;

-- =================================
-- Customer
-- =================================

-- name: CreateCustomer :one
INSERT INTO customers (organization_id, name, email, phone, document, document_type, external_reference, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListCustomersByOrganization :many
SELECT * FROM customers
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetCustomerByID :one
SELECT * FROM customers
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateCustomer :one
UPDATE customers
SET name = $3, email = $4, phone = $5, document = $6, document_type = $7, external_reference = $8, metadata = $9
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCustomer :exec
UPDATE customers SET deleted_at = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: HardDeleteCustomer :exec
DELETE FROM customers
WHERE id = $1 AND organization_id = $2;

-- =================================
-- Address
-- =================================

-- name: CreateAddress :one
INSERT INTO addresses (organization_id, customer_id, type, postal_code, street, number, complement, neighborhood, city, state, country, is_main)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: ListAddressesByCustomer :many
SELECT * FROM addresses
WHERE customer_id = $1 AND organization_id = $2 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetAddressByID :one
SELECT * FROM addresses
WHERE id = $1 AND customer_id = $2 AND organization_id = $3 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateAddress :one
UPDATE addresses
SET type = $4, postal_code = $5, street = $6, number = $7, complement = $8, neighborhood = $9, city = $10, state = $11, country = $12, is_main = $13
WHERE id = $1 AND customer_id = $2 AND organization_id = $3 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteAddress :exec
UPDATE addresses SET deleted_at = NOW()
WHERE id = $1 AND customer_id = $2 AND organization_id = $3 AND deleted_at IS NULL;

-- =================================
-- User
-- =================================

-- name: GetUserByEmailAndOrganization :one
SELECT * FROM users WHERE email = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (organization_id, name, email, password, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListUsersByOrganization :many
SELECT * FROM users WHERE organization_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC;

-- name: UpdateUser :one
UPDATE users SET name = $1, role = $2, updated_at = NOW()
WHERE id = $3 AND organization_id = $4 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- =================================
-- API Key
-- =================================

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys
WHERE key_hash = $1
  AND deleted_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW());

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = NOW() WHERE id = $1;
