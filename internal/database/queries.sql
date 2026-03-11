-- name: GetTenant :one
SELECT * FROM tenants
WHERE id = $1 LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY name;