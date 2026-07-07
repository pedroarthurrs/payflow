-- name: GetIdempotencyKey :one
SELECT response_code, response_body
FROM idempotency_keys
WHERE organization_id = $1 AND idempotency_key = $2 AND expires_at > NOW();

-- name: SaveIdempotencyKey :exec
INSERT INTO idempotency_keys (organization_id, idempotency_key, request_path, response_code, response_body, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (organization_id, idempotency_key) DO UPDATE
SET response_code = EXCLUDED.response_code, response_body = EXCLUDED.response_body;

-- name: DeleteExpiredIdempotencyKeys :execresult
DELETE FROM idempotency_keys
WHERE expires_at <= NOW();
