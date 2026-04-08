-- name: GetTransactionForUpdate :one
SELECT id, organization_id, status
FROM transactions
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: UpdateTransactionStatus :exec
UPDATE transactions
SET status = $3
WHERE organization_id = $1 AND id = $2;

-- name: InsertTransitionHistory :exec
INSERT INTO transaction_status_history (
    organization_id,
    transaction_id,
    old_status,
    new_status,
    event_source,
    event_source_id,
    reason
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListTransitionHistory :many
SELECT id, organization_id, transaction_id, old_status, new_status,
       event_source, event_source_id, reason, created_at
FROM transaction_status_history
WHERE organization_id = $1 AND transaction_id = $2
ORDER BY created_at ASC;
