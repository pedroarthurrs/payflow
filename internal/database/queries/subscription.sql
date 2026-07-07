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
    payment_method,
    metadata
) VALUES ($1, $2, $3, $4, $5, false, $6, $7)
RETURNING id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at;

-- name: GetSubscriptionByID :one
SELECT id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at FROM subscriptions
WHERE organization_id = $1 AND id = $2
LIMIT 1;

-- name: GetSubscriptionForUpdate :one
SELECT id, organization_id, customer_id, status
FROM subscriptions
WHERE organization_id = $1 AND id = $2
FOR UPDATE;

-- name: ListSubscriptions :many
SELECT id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at FROM subscriptions
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
RETURNING id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at;

-- name: SetCancelAtPeriodEnd :one
UPDATE subscriptions
SET cancel_at_period_end = $3, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at;

-- name: AdvanceSubscriptionPeriod :one
UPDATE subscriptions
SET current_period_start = $3, current_period_end = $4, billing_locked_until = NULL, updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING id, organization_id, customer_id, status, current_period_start, current_period_end, cancel_at_period_end, payment_method, billing_locked_until, metadata, created_at, updated_at;

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

-- name: InsertSubscriptionHistory :exec
INSERT INTO subscription_status_history (
    organization_id, subscription_id,
    old_status, new_status,
    event_source, event_source_id, reason
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListSubscriptionHistory :many
SELECT id, organization_id, subscription_id, old_status, new_status, event_source, event_source_id, reason, created_at
FROM subscription_status_history
WHERE organization_id = $1 AND subscription_id = $2
  AND (sqlc.narg('cursor_time')::timestamptz IS NULL
       OR (created_at, id) > (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::uuid))
ORDER BY created_at ASC, id ASC
LIMIT $3;

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
