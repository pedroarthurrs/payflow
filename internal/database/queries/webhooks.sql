-- name: CreateWebhookEndpoint :one
INSERT INTO webhook_endpoints (organization_id, url, description, enabled_events, environment)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, organization_id, url, description, enabled_events, environment, is_active, created_at, updated_at;

-- name: GetWebhookEndpointByID :one
SELECT id, organization_id, url, description, enabled_events, environment, is_active, created_at, updated_at
FROM webhook_endpoints
WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: ListWebhookEndpoints :many
SELECT id, organization_id, url, description, enabled_events, environment, is_active, created_at, updated_at
FROM webhook_endpoints
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWebhookEndpoint :one
UPDATE webhook_endpoints
SET url = COALESCE(sqlc.narg('url'), url),
    description = COALESCE(sqlc.narg('description'), description),
    enabled_events = COALESCE(sqlc.narg('enabled_events'), enabled_events),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE organization_id = @organization_id AND id = @endpoint_id AND deleted_at IS NULL
RETURNING id, organization_id, url, description, enabled_events, environment, is_active, created_at, updated_at;

-- name: SoftDeleteWebhookEndpoint :exec
UPDATE webhook_endpoints
SET deleted_at = NOW()
WHERE organization_id = $1 AND id = $2 AND deleted_at IS NULL;

-- name: ListActiveWebhookEndpointsByEvent :many
SELECT id, organization_id, url, description, enabled_events, environment, is_active, created_at, updated_at
FROM webhook_endpoints
WHERE organization_id = $1
  AND is_active = true
  AND deleted_at IS NULL
  AND environment = $2::environment_type
  AND $3 = ANY(enabled_events);
