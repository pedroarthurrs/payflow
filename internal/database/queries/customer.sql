-- name: CreateCustomer :one
INSERT INTO customers (organization_id, name, email, phone, document, document_type, external_reference, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListCustomersByOrganization :many
SELECT * FROM customers
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

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
