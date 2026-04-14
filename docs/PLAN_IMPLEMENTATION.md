# Implementação: Plans

Guia passo a passo para implementar o módulo de Plans. Siga a ordem exata das seções — cada etapa depende da anterior.

## Sumário de arquivos

| Ação | Arquivo |
|------|---------|
| Criar | `internal/domain/plan.go` |
| Modificar | `internal/domain/errors.go` |
| Criar | `internal/database/queries/plan.sql` |
| Modificar | `sqlc.yml` |
| Executar | `make sqlc` |
| Criar | `internal/database/repository/plan.go` |
| Modificar | `internal/database/connection.go` |
| Criar | `internal/service/plan.go` |
| Modificar | `internal/api/dto/context.go` |
| Criar | `internal/api/handler/plan.go` |
| Modificar | `internal/api/router/router.go` |

---

## 1. Domain

### `internal/domain/plan.go` (criar)

```go
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
```

### `internal/domain/errors.go` (modificar)

Adicionar ao bloco `var (...)` existente:

```go
	ErrPlanNotFound = errors.New("plan not found")
	ErrPlanInUse    = errors.New("plan is referenced by active subscriptions")
```

---

## 2. SQL Queries

### `internal/database/queries/plan.sql` (criar)

```sql
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
```

---

## 3. Configurar SQLC

### `sqlc.yml` (modificar)

Adicionar o bloco abaixo como último item da lista `sql:`:

```yaml
  - schema: "deploy/migrations"
    queries: "internal/database/queries/plan.sql"
    engine: "postgresql"
    gen:
      go:
        package: "plandb"
        out: "internal/database/dbgen/plan"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: true
```

### Gerar código

```bash
make sqlc
```

Isso cria `internal/database/dbgen/plan/` com `models.go`, `query.sql.go` e `querier.go`.
O struct gerado em `models.go` será:

```go
type Plan struct {
    ID             uuid.UUID      `json:"id"`
    OrganizationID uuid.UUID      `json:"organization_id"`
    ProductID      uuid.UUID      `json:"product_id"`
    Name           sql.NullString `json:"name"`
    Slug           sql.NullString `json:"slug"`
    AmountCents    int32          `json:"amount_cents"`
    Currency       sql.NullString `json:"currency"`
    Interval       sql.NullString `json:"interval"`
    IntervalCount  sql.NullInt32  `json:"interval_count"`
    Active         sql.NullBool   `json:"active"`
    Version        sql.NullInt32  `json:"version"`
    CreatedAt      sql.NullTime   `json:"created_at"`
    UpdatedAt      sql.NullTime   `json:"updated_at"`
    DeletedAt      sql.NullTime   `json:"deleted_at"`
}
```

> Esse shape já existe em `internal/database/dbgen/product/models.go` (porque SQLC gera modelos de todas as tabelas do schema). O novo pacote `plandb` terá o mesmo struct no namespace correto.

---

## 4. Repository

### `internal/database/repository/plan.go` (criar)

```go
package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	plandb "i9pay/internal/database/dbgen/plan"
	"i9pay/internal/domain"

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
```

> **Nota:** Os nomes dos parâmetros gerados por SQLC para `sqlc.narg('active')` e os campos de `ListPlansByProductParams` / `CountPlansByProductParams` devem ser verificados em `internal/database/dbgen/plan/query.sql.go` após executar `make sqlc`. O campo tipicamente é `Active sql.NullBool`.

---

## 5. Connection

### `internal/database/connection.go` (modificar)

**Adicionar import:**

```go
plandb "i9pay/internal/database/dbgen/plan"
```

**Adicionar campo na struct `DB`:**

```go
PlanQ plandb.Querier
```

**Adicionar inicialização na função `New`, junto aos outros `Querier`:**

```go
PlanQ: plandb.New(db),
```

---

## 6. Service

### `internal/service/plan.go` (criar)

```go
package service

import (
	"context"
	"log/slog"

	"i9pay/internal/domain"

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
```

---

## 7. HTTP

### `internal/api/dto/context.go` (modificar)

Adicionar após a função `ProductID`:

```go
// PlanID parses the ":plan_id" path parameter as a UUID.
func PlanID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("plan_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid plan id"))
		return uuid.UUID{}, false
	}
	return id, true
}
```

### `internal/api/handler/plan.go` (criar)

```go
package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PlanHandler handles plan-related HTTP endpoints.
type PlanHandler struct {
	planSvc *service.PlanService
}

// NewPlanHandler creates a PlanHandler.
func NewPlanHandler(planSvc *service.PlanService) *PlanHandler {
	return &PlanHandler{planSvc: planSvc}
}

// RegisterRoutes registers plan routes nested under /products/:product_id/plans.
func (h *PlanHandler) RegisterRoutes(rg *gin.RouterGroup) {
	plans := rg.Group("/products/:product_id/plans")
	plans.POST("", middleware.RequireRole(domain.RoleAdmin), h.Create)
	plans.GET("", h.List)
	plans.GET("/:plan_id", h.GetByID)
	plans.PUT("/:plan_id", middleware.RequireRole(domain.RoleAdmin), h.Update)
	plans.DELETE("/:plan_id", middleware.RequireRole(domain.RoleAdmin), h.Delete)
}

// createPlanRequest is the request body for plan creation.
type createPlanRequest struct {
	Name          string  `json:"name"           binding:"required"`
	Slug          *string `json:"slug"           binding:"omitempty"`
	AmountCents   int32   `json:"amount_cents"   binding:"required,min=0"`
	Currency      string  `json:"currency"       binding:"required,len=3"`
	Interval      string  `json:"interval"       binding:"required,oneof=day week month year"`
	IntervalCount int32   `json:"interval_count" binding:"required,min=1"`
	Active        *bool   `json:"active"         binding:"required"`
}

// updatePlanRequest is the request body for plan update.
type updatePlanRequest struct {
	Name          string  `json:"name"           binding:"required"`
	Slug          *string `json:"slug"           binding:"omitempty"`
	AmountCents   int32   `json:"amount_cents"   binding:"required,min=0"`
	Currency      string  `json:"currency"       binding:"required,len=3"`
	Interval      string  `json:"interval"       binding:"required,oneof=day week month year"`
	IntervalCount int32   `json:"interval_count" binding:"required,min=1"`
	Active        *bool   `json:"active"         binding:"required"`
}

// planResponse is the API response shape for a plan.
type planResponse struct {
	ID            uuid.UUID `json:"id"`
	ProductID     uuid.UUID `json:"product_id"`
	Name          string    `json:"name"`
	Slug          *string   `json:"slug,omitempty"`
	AmountCents   int32     `json:"amount_cents"`
	Currency      string    `json:"currency"`
	Interval      string    `json:"interval"`
	IntervalCount int32     `json:"interval_count"`
	Active        bool      `json:"active"`
	Version       int32     `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toPlanResponse(p *domain.Plan) planResponse {
	return planResponse{
		ID:            p.ID,
		ProductID:     p.ProductID,
		Name:          p.Name,
		Slug:          p.Slug,
		AmountCents:   p.AmountCents,
		Currency:      p.Currency,
		Interval:      p.Interval,
		IntervalCount: p.IntervalCount,
		Active:        p.Active,
		Version:       p.Version,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

// Create handles POST /v1/products/:product_id/plans.
func (h *PlanHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}

	var req createPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.PlanInput{
		ProductID:     pid,
		Name:          req.Name,
		Slug:          req.Slug,
		AmountCents:   req.AmountCents,
		Currency:      req.Currency,
		Interval:      req.Interval,
		IntervalCount: req.IntervalCount,
		Active:        *req.Active,
	}

	plan, err := h.planSvc.Create(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toPlanResponse(plan))
}

// List handles GET /v1/products/:product_id/plans.
func (h *PlanHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	params := domain.PlanListParams{
		ListParams: pag.ToListParams(),
	}
	if raw := c.Query("active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("active must be true or false"))
			return
		}
		params.Active = &v
	}

	plans, total, err := h.planSvc.List(c.Request.Context(), orgID, pid, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]planResponse, len(plans))
	for i := range plans {
		resp[i] = toPlanResponse(&plans[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[planResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// GetByID handles GET /v1/products/:product_id/plans/:plan_id.
func (h *PlanHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}
	planID, ok := dto.PlanID(c)
	if !ok {
		return
	}

	plan, err := h.planSvc.GetByID(c.Request.Context(), orgID, planID)
	if err != nil {
		if errors.Is(err, domain.ErrPlanNotFound) {
			_ = c.Error(apierr.NotFound("plan not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	// Ensure plan belongs to the product in the URL
	if plan.ProductID != pid {
		_ = c.Error(apierr.NotFound("plan not found"))
		return
	}

	c.JSON(http.StatusOK, toPlanResponse(plan))
}

// Update handles PUT /v1/products/:product_id/plans/:plan_id.
func (h *PlanHandler) Update(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}
	planID, ok := dto.PlanID(c)
	if !ok {
		return
	}

	var req updatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	// Verify product ownership before update
	current, err := h.planSvc.GetByID(c.Request.Context(), orgID, planID)
	if err != nil {
		if errors.Is(err, domain.ErrPlanNotFound) {
			_ = c.Error(apierr.NotFound("plan not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	if current.ProductID != pid {
		_ = c.Error(apierr.NotFound("plan not found"))
		return
	}

	input := domain.PlanUpdateInput{
		Name:          req.Name,
		Slug:          req.Slug,
		AmountCents:   req.AmountCents,
		Currency:      req.Currency,
		Interval:      req.Interval,
		IntervalCount: req.IntervalCount,
		Active:        *req.Active,
	}

	plan, err := h.planSvc.Update(c.Request.Context(), orgID, planID, input)
	if err != nil {
		if errors.Is(err, domain.ErrPlanNotFound) {
			_ = c.Error(apierr.NotFound("plan not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toPlanResponse(plan))
}

// Delete handles DELETE /v1/products/:product_id/plans/:plan_id.
// Performs a soft delete. Use query param ?hard=true for permanent deletion.
func (h *PlanHandler) Delete(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}
	planID, ok := dto.PlanID(c)
	if !ok {
		return
	}

	// Verify the plan belongs to the product in the URL
	plan, err := h.planSvc.GetByID(c.Request.Context(), orgID, planID)
	if err != nil {
		if errors.Is(err, domain.ErrPlanNotFound) {
			_ = c.Error(apierr.NotFound("plan not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	if plan.ProductID != pid {
		_ = c.Error(apierr.NotFound("plan not found"))
		return
	}

	if c.Query("hard") == "true" {
		if err := h.planSvc.HardDelete(c.Request.Context(), orgID, planID); err != nil {
			if errors.Is(err, domain.ErrPlanInUse) {
				_ = c.Error(apierr.Conflict("plan is referenced by active subscriptions"))
				return
			}
			_ = c.Error(err)
			return
		}
		c.Status(http.StatusNoContent)
		return
	}

	if err := h.planSvc.Delete(c.Request.Context(), orgID, planID); err != nil {
		if errors.Is(err, domain.ErrPlanNotFound) {
			_ = c.Error(apierr.NotFound("plan not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
```

---

## 8. Router

### `internal/api/router/router.go` (modificar)

**Adicionar import** ao bloco de imports do pacote `plandb`:

Não é necessário importar `plandb` diretamente no router — apenas usar o repositório e serviço.

**Adicionar na seção de Repositories** (após `productRepo`):

```go
planRepo := repository.NewPlanRepo(adapters.DB.PlanQ)
```

**Adicionar na seção de Services** (após `productSvc`):

```go
planSvc := service.NewPlanService(planRepo)
```

**Adicionar na seção de Handlers** (após `productHandler`):

```go
planHandler := handler.NewPlanHandler(planSvc)
```

**Adicionar no registro de rotas protegidas** (após `productHandler.RegisterRoutes(protected)`):

```go
planHandler.RegisterRoutes(protected)
```

---

## 9. Verificação

### Compilar

```bash
go build ./...
```

### Testar manualmente (com servidor rodando)

```bash
# Criar um plano
curl -s -X POST http://localhost:8080/v1/products/<product_id>/plans \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pro Mensal",
    "amount_cents": 4900,
    "currency": "BRL",
    "interval": "month",
    "interval_count": 1,
    "active": true
  }' | jq

# Listar planos
curl -s http://localhost:8080/v1/products/<product_id>/plans \
  -H "Authorization: Bearer <token>" | jq

# Listar somente planos ativos
curl -s "http://localhost:8080/v1/products/<product_id>/plans?active=true" \
  -H "Authorization: Bearer <token>" | jq
```

### Testes unitários

Seguir o padrão de `internal/service/product.go` e criar `internal/service/plan_test.go`.
Os mocks do repositório seguem o padrão de `internal/service/mocks_test.go`.

---

## Observações

- **Versão automática:** a lógica de `Version + 1` fica no `PlanService.Update`, não no handler nem no repositório. O campo `Version` nunca é exposto na request body.
- **Produto vs. plano na URL:** o handler valida `plan.ProductID == pid` para garantir que o plano pertence ao produto da URL, evitando traversal entre recursos.
- **Hard delete:** exposto como `DELETE ?hard=true` para manter a interface simples. Apenas para uso administrativo — requer role `admin`.
- **`interval` é palavra reservada no PostgreSQL:** SQLC lida com a geração automaticamente. O campo Go gerado será `Interval sql.NullString`.
