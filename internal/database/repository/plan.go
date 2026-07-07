package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	plandb "payflow/internal/database/dbgen/plan"
	"payflow/internal/domain"

	"github.com/google/uuid"
)

// PlanRepo implements domain.PlanRepository backed by PostgreSQL via sqlc.
type PlanRepo struct {
	q plandb.Querier
}

// NewPlanRepo creates a PlanRepo.
func NewPlanRepo(q plandb.Querier) *PlanRepo {
	return &PlanRepo{q: q}
}

// Create persists a new plan under a product/organization.
func (r *PlanRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.PlanInput) (*domain.Plan, error) {
	row, err := r.q.CreatePlan(ctx, plandb.CreatePlanParams{
		OrganizationID: organizationID,
		ProductID:      input.ProductID,
		Name:           sql.NullString{String: input.Name, Valid: input.Name != ""},
		Slug:           toNullString(input.Slug),
		AmountCents:    input.AmountCents,
		Currency:       sql.NullString{String: input.Currency, Valid: input.Currency != ""},
		Interval:       sql.NullString{String: input.Interval, Valid: input.Interval != ""},
		IntervalCount:  sql.NullInt32{Int32: input.IntervalCount, Valid: true},
		Active:         sql.NullBool{Bool: input.Active, Valid: true},
	})
	if err != nil {
		slog.Error("failed to create plan", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPlan(row)
	return &p, nil
}

// GetByID retrieves a plan by ID scoped to an organization.
func (r *PlanRepo) GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*domain.Plan, error) {
	row, err := r.q.GetPlanByID(ctx, plandb.GetPlanByIDParams{
		ID:             planID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPlanNotFound
		}
		slog.Error("failed to get plan", "organization_id", organizationID, "plan_id", planID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPlan(row)
	return &p, nil
}

// List returns plans for a product, with optional active filter and pagination.
func (r *PlanRepo) List(ctx context.Context, organizationID, productID uuid.UUID, params domain.PlanListParams) ([]domain.Plan, error) {
	rows, err := r.q.ListPlansByProduct(ctx, plandb.ListPlansByProductParams{
		OrganizationID: organizationID,
		ProductID:      productID,
		Active:         toNullBool(params.Active),
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list plans", "organization_id", organizationID, "product_id", productID, "error", err)
		return nil, domain.ErrInternal
	}
	plans := make([]domain.Plan, len(rows))
	for i := range rows {
		plans[i] = toDomainPlan(rows[i])
	}
	return plans, nil
}

// Count returns the total number of plans matching the given filters.
func (r *PlanRepo) Count(ctx context.Context, organizationID, productID uuid.UUID, params domain.PlanListParams) (int64, error) {
	count, err := r.q.CountPlansByProduct(ctx, plandb.CountPlansByProductParams{
		OrganizationID: organizationID,
		ProductID:      productID,
		Active:         toNullBool(params.Active),
	})
	if err != nil {
		slog.Error("failed to count plans", "organization_id", organizationID, "product_id", productID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// Update updates a plan scoped to an organization.
func (r *PlanRepo) Update(ctx context.Context, organizationID, planID uuid.UUID, input domain.PlanUpdateInput) (*domain.Plan, error) {
	row, err := r.q.UpdatePlan(ctx, plandb.UpdatePlanParams{
		ID:             planID,
		OrganizationID: organizationID,
		Name:           sql.NullString{String: input.Name, Valid: input.Name != ""},
		Slug:           toNullString(input.Slug),
		AmountCents:    input.AmountCents,
		Currency:       sql.NullString{String: input.Currency, Valid: input.Currency != ""},
		Interval:       sql.NullString{String: input.Interval, Valid: input.Interval != ""},
		IntervalCount:  sql.NullInt32{Int32: input.IntervalCount, Valid: true},
		Active:         sql.NullBool{Bool: input.Active, Valid: true},
		Version:        sql.NullInt32{Int32: input.Version, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPlanNotFound
		}
		slog.Error("failed to update plan", "organization_id", organizationID, "plan_id", planID, "error", err)
		return nil, domain.ErrInternal
	}
	p := toDomainPlan(row)
	return &p, nil
}

// SoftDelete marks a plan as deleted without removing the record.
func (r *PlanRepo) SoftDelete(ctx context.Context, organizationID, planID uuid.UUID) error {
	// Verify existence first (SoftDeletePlan is :exec — no row returned)
	_, err := r.q.GetPlanByID(ctx, plandb.GetPlanByIDParams{
		ID:             planID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrPlanNotFound
		}
		slog.Error("failed to get plan for soft delete", "organization_id", organizationID, "plan_id", planID, "error", err)
		return domain.ErrInternal
	}
	if err := r.q.SoftDeletePlan(ctx, plandb.SoftDeletePlanParams{
		ID:             planID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to soft delete plan", "organization_id", organizationID, "plan_id", planID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// HardDelete permanently removes a plan from the database.
func (r *PlanRepo) HardDelete(ctx context.Context, organizationID, planID uuid.UUID) error {
	if err := r.q.HardDeletePlan(ctx, plandb.HardDeletePlanParams{
		ID:             planID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to hard delete plan", "organization_id", organizationID, "plan_id", planID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// HasActiveSubscriptions returns true if any non-canceled subscription references this plan.
func (r *PlanRepo) HasActiveSubscriptions(ctx context.Context, organizationID, planID uuid.UUID) (bool, error) {
	has, err := r.q.PlanHasActiveSubscriptions(ctx, plandb.PlanHasActiveSubscriptionsParams{
		PlanID:         planID,
		OrganizationID: organizationID,
	})
	if err != nil {
		slog.Error("failed to check active subscriptions for plan", "organization_id", organizationID, "plan_id", planID, "error", err)
		return false, domain.ErrInternal
	}
	return has, nil
}

// toDomainPlan converts a plandb.Plan row to a domain.Plan.
func toDomainPlan(row plandb.Plan) domain.Plan {
	p := domain.Plan{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ProductID:      row.ProductID,
		AmountCents:    row.AmountCents,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	if row.Name.Valid {
		p.Name = row.Name.String
	}
	if row.Slug.Valid {
		s := row.Slug.String
		p.Slug = &s
	}
	if row.Currency.Valid {
		p.Currency = row.Currency.String
	}
	if row.Interval.Valid {
		p.Interval = row.Interval.String
	}
	if row.IntervalCount.Valid {
		p.IntervalCount = row.IntervalCount.Int32
	}
	if row.Active.Valid {
		p.Active = row.Active.Bool
	}
	if row.Version.Valid {
		p.Version = row.Version.Int32
	}
	return p
}
