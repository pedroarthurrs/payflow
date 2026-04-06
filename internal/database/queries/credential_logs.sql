-- name: ListCredentialLogs :many
SELECT id, organization_id, credential_id, method, path,
       request_body, response_body, status_code, latency_ms,
       ip_address, user_agent, created_at
FROM credential_logs
WHERE organization_id = @organization_id
  AND (sqlc.narg('cursor_ts')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::uuid))
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR created_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR created_at <= sqlc.narg('to_ts')::timestamptz)
  AND (sqlc.narg('status_code')::int4 IS NULL OR status_code = sqlc.narg('status_code')::int4)
  AND (sqlc.narg('method')::text IS NULL OR method = sqlc.narg('method')::text)
  AND (sqlc.narg('path_prefix')::text IS NULL OR path LIKE sqlc.narg('path_prefix')::text || '%')
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: GetCredentialLogByID :one
SELECT id, organization_id, credential_id, method, path,
       request_body, response_body, status_code, latency_ms,
       ip_address, user_agent, created_at
FROM credential_logs
WHERE organization_id = $1 AND id = $2;
