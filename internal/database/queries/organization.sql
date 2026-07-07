-- name: GetOrganization :one
SELECT * FROM organizations
WHERE id = $1 LIMIT 1;

-- name: ListOrganizations :many
SELECT * FROM organizations
ORDER BY name;
