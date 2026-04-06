-- name: CreateProduct :one
INSERT INTO products (organization_id, name, active)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListProductsByOrganization :many
SELECT * FROM products
WHERE organization_id = $1 AND deleted_at is NULL
ORDER BY created_at DESC;

-- name: GetProductByID :one
SELECT * from products
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateProduct :one
UPDATE products
SET name = $3, active = $4
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteProduct :exec
UPDATE products SET deleted_at = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: HardDeleteProduct :exec
DELETE FROM products
WHERE id = $1 AND organization_id = $2;
