-- name: CreateAddress :one
INSERT INTO addresses (organization_id, customer_id, type, postal_code, street, number, complement, neighborhood, city, state, country, is_main)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: ListAddressesByCustomer :many
SELECT * FROM addresses
WHERE customer_id = $1 AND organization_id = $2 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

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
