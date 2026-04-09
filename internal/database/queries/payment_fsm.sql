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
