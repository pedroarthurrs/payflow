-- name: CreateWebhookLog :exec
INSERT INTO webhook_logs (organization_id, endpoint_id, event_type, payload, response_body, response_status, duration_ms, attempt_number, success)
VALUES (@organization_id, @endpoint_id, @event_type, @payload, @response_body, @response_status, @duration_ms, @attempt_number, @success);

-- name: GetWebhookLogByID :one
SELECT id, organization_id, endpoint_id, event_type, payload, response_body, response_status, duration_ms, attempt_number, success, created_at
FROM webhook_logs
WHERE organization_id = $1 AND id = $2;

-- name: ListWebhookLogs :many
SELECT id, organization_id, endpoint_id, event_type, payload, response_body, response_status, duration_ms, attempt_number, success, created_at
FROM webhook_logs
WHERE organization_id = @organization_id
  AND (sqlc.narg('cursor_ts')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::uuid))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR created_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR created_at <= sqlc.narg('to_ts')::timestamptz)
  AND (sqlc.narg('endpoint_id')::uuid IS NULL OR endpoint_id = sqlc.narg('endpoint_id')::uuid)
  AND (sqlc.narg('event_type')::text IS NULL OR event_type = sqlc.narg('event_type')::text)
  AND (sqlc.narg('success')::boolean IS NULL OR success = sqlc.narg('success')::boolean)
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;
