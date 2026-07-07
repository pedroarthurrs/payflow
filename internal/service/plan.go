package service

import (
	"context"
	"log/slog"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// PlanService handles plan business logic.
type PlanService struct {
	plans domain.PlanRepository
}

// NewPlanService creates a PlanService.
func NewPlanService(plans domain.PlanRepository) *PlanService {
	return &PlanService{plans: plans}
}

// Create creates a new plan under a product.
func (s *PlanService) Create(ctx context.Context, organizationID uuid.UUID, input domain.PlanInput) (*domain.Plan, error) {
	return s.plans.Create(ctx, organizationID, input)
}

// GetByID returns a single plan by ID scoped to an organization.
func (s *PlanService) GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*domain.Plan, error) {
	return s.plans.GetByID(ctx, organizationID, planID)
}

// List returns plans for a product with optional filtering and pagination.
func (s *PlanService) List(ctx context.Context, organizationID, productID uuid.UUID, params domain.PlanListParams) ([]domain.Plan, int64, error) {
	plans, err := s.plans.List(ctx, organizationID, productID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.plans.Count(ctx, organizationID, productID, params)
	if err != nil {
		return nil, 0, err
	}
	return plans, total, nil
}

// Update updates an existing plan. Automatically increments version when pricing
// fields (amount_cents, interval, interval_count) change.
func (s *PlanService) Update(ctx context.Context, organizationID, planID uuid.UUID, input domain.PlanUpdateInput) (*domain.Plan, error) {
	current, err := s.plans.GetByID(ctx, organizationID, planID)
	if err != nil {
		return nil, err
	}

	input.Version = current.Version
	if input.AmountCents != current.AmountCents ||
		input.Interval != current.Interval ||
		input.IntervalCount != current.IntervalCount {
		input.Version = current.Version + 1
	}

	return s.plans.Update(ctx, organizationID, planID, input)
}

// Delete performs a soft delete on a plan.
func (s *PlanService) Delete(ctx context.Context, organizationID, planID uuid.UUID) error {
	return s.plans.SoftDelete(ctx, organizationID, planID)
}

// HardDelete permanently removes a plan. Returns ErrPlanInUse if active subscriptions exist.
func (s *PlanService) HardDelete(ctx context.Context, organizationID, planID uuid.UUID) error {
	has, err := s.plans.HasActiveSubscriptions(ctx, organizationID, planID)
	if err != nil {
		return err
	}
	if has {
		return domain.ErrPlanInUse
	}
	if err := s.plans.HardDelete(ctx, organizationID, planID); err != nil {
		return err
	}
	slog.Warn("plan permanently deleted", "organization_id", organizationID, "plan_id", planID, "reason", "hard_delete")
	return nil
}
