-- name: CreatePlan :one
INSERT INTO plans (organization_id, product_id, name, slug, amount_cents, currency, interval, interval_count, active)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetPlanByID :one
SELECT * FROM plans
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: ListPlansByProduct :many
SELECT * FROM plans
WHERE organization_id = $1
  AND product_id = $2
  AND deleted_at IS NULL
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'))
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountPlansByProduct :one
SELECT count(*) FROM plans
WHERE organization_id = $1
  AND product_id = $2
  AND deleted_at IS NULL
  AND (sqlc.narg('active')::boolean IS NULL OR active = sqlc.narg('active'));

-- name: UpdatePlan :one
UPDATE plans
SET name           = $3,
    slug           = $4,
    amount_cents   = $5,
    currency       = $6,
    interval       = $7,
    interval_count = $8,
    active         = $9,
    version        = $10,
    updated_at     = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePlan :exec
UPDATE plans SET deleted_at = NOW()
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL;

-- name: HardDeletePlan :exec
DELETE FROM plans
WHERE id = $1 AND organization_id = $2;

-- name: PlanHasActiveSubscriptions :one
SELECT EXISTS(
    SELECT 1
    FROM subscription_items si
    JOIN subscriptions s ON s.id = si.subscription_id
    WHERE si.plan_id        = $1
      AND si.organization_id = $2
      AND s.status NOT IN ('canceled', 'unpaid')
);
