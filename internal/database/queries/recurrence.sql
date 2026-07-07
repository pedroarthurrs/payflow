-- name: ListSubscriptionsDueForBilling :many
SELECT id, organization_id, customer_id, status, cancel_at_period_end, current_period_end, payment_method
FROM subscriptions
WHERE current_period_end <= $1
  AND status IN ('active', 'past_due', 'trialing')
  AND (billing_locked_until IS NULL OR billing_locked_until < NOW())
ORDER BY current_period_end ASC
LIMIT $2
FOR UPDATE SKIP LOCKED;

-- name: LockSubscriptionForBilling :exec
UPDATE subscriptions SET billing_locked_until = $3 WHERE organization_id = $1 AND id = $2;

-- name: UnlockSubscriptionBilling :exec
UPDATE subscriptions SET billing_locked_until = NULL WHERE organization_id = $1 AND id = $2;
