package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"payflow/internal/api/dto"
	apierr "payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/internal/service"

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
