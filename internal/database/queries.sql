-- name: GetTenant :one
SELECT * FROM tenants
WHERE id = $1 LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY name;

-- name: CreateCustomer :one
INSERT INTO customers (tenant_id, name, email, phone, document, document_type, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListCustomersByTenant :many
SELECT * FROM customers
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetCustomerByID :one
SELECT * FROM customers
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateCustomer :one
UPDATE customers
SET name = $3, email = $4, phone = $5, document = $6, document_type = $7, metadata = $8
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCustomer :exec
UPDATE customers SET deleted_at = NOW()
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: HardDeleteCustomer :exec
DELETE FROM customers
WHERE id = $1 AND tenant_id = $2;
