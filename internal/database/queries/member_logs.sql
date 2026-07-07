-- name: ListMemberLogs :many
SELECT id, organization_id, member_id, action, method, path,
       status_code, latency_ms, ip_address, user_agent, metadata, created_at
FROM member_logs
WHERE organization_id = @organization_id
  AND (sqlc.narg('cursor_ts')::timestamptz IS NULL OR (created_at, id) < (sqlc.narg('cursor_ts')::timestamptz, sqlc.narg('cursor_id')::uuid))
  AND (sqlc.narg('member_id')::uuid IS NULL OR member_id = sqlc.narg('member_id')::uuid)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('from_ts')::timestamptz IS NULL OR created_at >= sqlc.narg('from_ts')::timestamptz)
  AND (sqlc.narg('to_ts')::timestamptz IS NULL OR created_at <= sqlc.narg('to_ts')::timestamptz)
  AND (sqlc.narg('status_code')::int4 IS NULL OR status_code = sqlc.narg('status_code')::int4)
  AND (sqlc.narg('method')::text IS NULL OR method = sqlc.narg('method')::text)
  AND (sqlc.narg('path_prefix')::text IS NULL OR path LIKE sqlc.narg('path_prefix')::text || '%')
ORDER BY created_at DESC, id DESC
LIMIT @page_limit;

-- name: GetMemberLogByID :one
SELECT id, organization_id, member_id, action, method, path,
       status_code, latency_ms, ip_address, user_agent, metadata, created_at
FROM member_logs
WHERE organization_id = $1 AND id = $2;

-- name: CreateMemberLog :exec
INSERT INTO member_logs (
    organization_id, member_id, action, method, path,
    status_code, latency_ms, ip_address, user_agent, metadata
) VALUES (
    @organization_id, @member_id, @action, @method, @path,
    @status_code, @latency_ms, @ip_address, @user_agent, @metadata
);

-- name: DeleteMemberLogsOlderThan :execresult
DELETE FROM member_logs
WHERE created_at < $1;
