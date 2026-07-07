package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PlanRepository defines persistence operations for the plan domain.
type PlanRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input PlanInput) (*Plan, error)
	GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*Plan, error)
	List(ctx context.Context, organizationID, productID uuid.UUID, params PlanListParams) ([]Plan, error)
	Count(ctx context.Context, organizationID, productID uuid.UUID, params PlanListParams) (int64, error)
	Update(ctx context.Context, organizationID, planID uuid.UUID, input PlanUpdateInput) (*Plan, error)
	SoftDelete(ctx context.Context, organizationID, planID uuid.UUID) error
	HardDelete(ctx context.Context, organizationID, planID uuid.UUID) error
	HasActiveSubscriptions(ctx context.Context, organizationID, planID uuid.UUID) (bool, error)
}

// Plan represents a pricing plan within a product.
type Plan struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	ProductID      uuid.UUID
	Name           string
	Slug           *string
	AmountCents    int32
	Currency       string
	Interval       string
	IntervalCount  int32
	Active         bool
	Version        int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// PlanInput carries the fields for creating a new plan.
type PlanInput struct {
	ProductID     uuid.UUID
	Name          string
	Slug          *string
	AmountCents   int32
	Currency      string
	Interval      string
	IntervalCount int32
	Active        bool
}

// PlanUpdateInput carries the fields for updating a plan.
// Version is managed internally by the service — not set by the caller.
type PlanUpdateInput struct {
	Name          string
	Slug          *string
	AmountCents   int32
	Currency      string
	Interval      string
	IntervalCount int32
	Active        bool
	Version       int32
}

// PlanListParams carries filter and pagination parameters for listing plans.
type PlanListParams struct {
	ListParams
	Active *bool
}
