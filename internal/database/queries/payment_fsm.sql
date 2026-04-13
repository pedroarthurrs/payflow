-- name: GetPaymentForUpdate :one
SELECT id, organization_id, status
FROM payments
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET status = $3
WHERE organization_id = $1 AND id = $2;

-- name: InsertPaymentHistory :exec
INSERT INTO payment_status_history (
    organization_id,
    payment_id,
    old_status,
    new_status,
    event_source,
    event_source_id,
    reason
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListPaymentHistory :many
SELECT id, organization_id, payment_id, old_status, new_status,
       event_source, event_source_id, reason, created_at
FROM payment_status_history
WHERE organization_id = $1 AND payment_id = $2
ORDER BY created_at ASC;

-- name: CreatePayment :one
INSERT INTO payments (
    organization_id, customer_id, payment_method, amount_cents,
    status, currency, gateway_name, idempotency_key
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, organization_id, customer_id, invoice_id, gateway_name,
          gateway_reference, payment_method, amount_cents, status,
          currency, idempotency_key, error_log, created_at, updated_at;

-- name: GetPaymentByID :one
SELECT id, organization_id, customer_id, invoice_id, gateway_name,
       gateway_reference, payment_method, amount_cents, status,
       currency, idempotency_key, error_log, created_at, updated_at
FROM payments
WHERE organization_id = $1 AND id = $2;

-- name: ListPayments :many
SELECT id, organization_id, customer_id, invoice_id, gateway_name,
       gateway_reference, payment_method, amount_cents, status,
       currency, idempotency_key, error_log, created_at, updated_at
FROM payments
WHERE organization_id = $1
  AND (sqlc.narg('status')::payment_status_type IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('payment_method')::varchar IS NULL OR payment_method = sqlc.narg('payment_method'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountPayments :one
SELECT count(*) FROM payments
WHERE organization_id = $1
  AND (sqlc.narg('status')::payment_status_type IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('payment_method')::varchar IS NULL OR payment_method = sqlc.narg('payment_method'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'));

-- name: UpdatePaymentGatewayReference :exec
UPDATE payments
SET gateway_name = $3, gateway_reference = $4
WHERE organization_id = $1 AND id = $2;

-- name: UpdatePaymentErrorLog :exec
UPDATE payments
SET error_log = $3
WHERE organization_id = $1 AND id = $2;
