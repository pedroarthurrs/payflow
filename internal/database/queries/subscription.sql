-- =============================================================================
-- Subscriptions
-- =============================================================================

-- name: CreateSubscription :one
INSERT INTO subscriptions (
    organization_id,
    customer_id,
    status,
    current_period_start,
    current_period_end,
    cancel_at_period_end,
    metadata
) VALUES ($1, $2, $3, $4, $5, false, $6)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions
WHERE organization_id = $1 AND id = $2
LIMIT 1;

-- name: GetSubscriptionForUpdate :one
SELECT id, organization_id, status
FROM subscriptions
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: ListSubscriptions :many
SELECT * FROM subscriptions
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'))
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubscriptions :one
SELECT count(*) FROM subscriptions
WHERE organization_id = $1
  AND (sqlc.narg('customer_id')::uuid IS NULL OR customer_id = sqlc.narg('customer_id'))
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from'))
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to'));

-- name: UpdateSubscriptionStatus :one
UPDATE subscriptions
SET status = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: SetCancelAtPeriodEnd :one
UPDATE subscriptions
SET cancel_at_period_end = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- =============================================================================
-- Subscription Items
-- =============================================================================

-- name: CreateSubscriptionItem :one
INSERT INTO subscription_items (organization_id, subscription_id, plan_id, quantity)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSubscriptionItems :many
SELECT * FROM subscription_items
WHERE organization_id = $1 AND subscription_id = $2
ORDER BY created_at ASC;

-- name: DeleteSubscriptionItems :exec
DELETE FROM subscription_items
WHERE organization_id = $1 AND subscription_id = $2;

-- =============================================================================
-- Subscription Status History
-- =============================================================================

-- name: ListSubscriptionHistory :many
SELECT * FROM subscription_status_history
WHERE organization_id = $1 AND subscription_id = $2
ORDER BY created_at ASC;

-- =============================================================================
-- Subscription Schedules
-- =============================================================================

-- name: CreateSubscriptionSchedule :one
INSERT INTO subscription_schedules (
    organization_id,
    subscription_id,
    scheduled_at,
    target_status,
    target_plan_id
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPendingSchedule :one
SELECT * FROM subscription_schedules
WHERE organization_id = $1
  AND subscription_id = $2
  AND applied_at IS NULL
LIMIT 1;

-- name: CancelSchedule :exec
UPDATE subscription_schedules
SET applied_at = NOW(), updated_at = NOW()
WHERE organization_id = $1 AND id = $2 AND applied_at IS NULL;
