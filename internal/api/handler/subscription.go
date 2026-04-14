package handler

import (
	"errors"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SubscriptionHandler handles subscription-related HTTP endpoints.
type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

// NewSubscriptionHandler creates a SubscriptionHandler.
func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// RegisterRoutes registers all subscription routes.
func (h *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subs := rg.Group("/subscriptions")
	subs.POST("", h.Create)
	subs.GET("", h.List)
	subs.GET("/:subscription_id", h.GetByID)
	subs.POST("/:subscription_id/cancel", middleware.RequireRole("member"), h.Cancel)
	subs.POST("/:subscription_id/pause", middleware.RequireRole("member"), h.Pause)
	subs.POST("/:subscription_id/resume", middleware.RequireRole("member"), h.Resume)
	subs.PUT("/:subscription_id/items", middleware.RequireRole(domain.RoleAdmin), h.ReplaceItems)
	subs.GET("/:subscription_id/items", h.ListItems)
	subs.GET("/:subscription_id/history", middleware.RequireRole(domain.RoleAdmin), h.ListHistory)
	subs.POST("/:subscription_id/schedules", middleware.RequireRole(domain.RoleAdmin), h.CreateSchedule)
	subs.DELETE("/:subscription_id/schedules/pending", middleware.RequireRole(domain.RoleAdmin), h.CancelSchedule)
}

// =============================================================================
// Request / Response DTOs
// =============================================================================

type subscriptionItemRequest struct {
	PlanID   uuid.UUID `json:"plan_id"  binding:"required"`
	Quantity int32     `json:"quantity" binding:"omitempty,min=1"`
}

type createSubscriptionRequest struct {
	CustomerID uuid.UUID                 `json:"customer_id" binding:"required"`
	TrialEnd   *time.Time                `json:"trial_end"   binding:"omitempty"`
	Metadata   map[string]any            `json:"metadata"    binding:"omitempty"`
	Items      []subscriptionItemRequest `json:"items"       binding:"required,min=1"`
}

type cancelSubscriptionRequest struct {
	Reason            string `json:"reason"              binding:"required"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}

type pauseResumeRequest struct {
	Reason string `json:"reason" binding:"omitempty"`
}

type createScheduleRequest struct {
	ScheduledAt  time.Time  `json:"scheduled_at"   binding:"required"`
	TargetStatus *string    `json:"target_status"  binding:"omitempty"`
	TargetPlanID *uuid.UUID `json:"target_plan_id" binding:"omitempty"`
}

type subscriptionItemResponse struct {
	ID        uuid.UUID `json:"id"`
	PlanID    uuid.UUID `json:"plan_id"`
	Quantity  int32     `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type subscriptionResponse struct {
	ID                 uuid.UUID                  `json:"id"`
	OrganizationID     uuid.UUID                  `json:"organization_id"`
	CustomerID         uuid.UUID                  `json:"customer_id"`
	Status             string                     `json:"status"`
	CurrentPeriodStart time.Time                  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                  `json:"current_period_end"`
	CancelAtPeriodEnd  bool                       `json:"cancel_at_period_end"`
	Metadata           any                        `json:"metadata,omitempty"`
	Items              []subscriptionItemResponse `json:"items,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

type subscriptionHistoryResponse struct {
	ID        uuid.UUID `json:"id"`
	OldStatus *string   `json:"old_status,omitempty"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type scheduleResponse struct {
	ID             uuid.UUID  `json:"id"`
	SubscriptionID uuid.UUID  `json:"subscription_id"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	TargetStatus   *string    `json:"target_status,omitempty"`
	TargetPlanID   *uuid.UUID `json:"target_plan_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toSubResponse(s *domain.Subscription) subscriptionResponse {
	resp := subscriptionResponse{
		ID:                 s.ID,
		OrganizationID:     s.OrganizationID,
		CustomerID:         s.CustomerID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		Metadata:           s.Metadata,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
	if len(s.Items) > 0 {
		resp.Items = make([]subscriptionItemResponse, len(s.Items))
		for i, item := range s.Items {
			resp.Items[i] = subscriptionItemResponse{
				ID:        item.ID,
				PlanID:    item.PlanID,
				Quantity:  item.Quantity,
				CreatedAt: item.CreatedAt,
				UpdatedAt: item.UpdatedAt,
			}
		}
	}
	return resp
}

func toScheduleResponse(s *domain.SubscriptionSchedule) scheduleResponse {
	r := scheduleResponse{
		ID:             s.ID,
		SubscriptionID: s.SubscriptionID,
		ScheduledAt:    s.ScheduledAt,
		TargetPlanID:   s.TargetPlanID,
		CreatedAt:      s.CreatedAt,
	}
	if s.TargetStatus != nil {
		str := string(*s.TargetStatus)
		r.TargetStatus = &str
	}
	return r
}

// =============================================================================
// Handlers
// =============================================================================

// Create handles POST /v1/subscriptions.
func (h *SubscriptionHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]domain.SubscriptionItemInput, len(req.Items))
	for i, item := range req.Items {
		q := item.Quantity
		if q == 0 {
			q = 1
		}
		items[i] = domain.SubscriptionItemInput{PlanID: item.PlanID, Quantity: q}
	}

	input := domain.CreateSubscriptionInput{
		CustomerID: req.CustomerID,
		TrialEnd:   req.TrialEnd,
		Items:      items,
	}

	sub, err := h.svc.Create(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toSubResponse(sub))
}

// List handles GET /v1/subscriptions.
func (h *SubscriptionHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	dateRange, ok := dto.ParseDateRange(c)
	if !ok {
		return
	}

	params := domain.SubscriptionListParams{
		ListParams:  pag.ToListParams(),
		CreatedFrom: dateRange.From,
		CreatedTo:   dateRange.To,
	}

	if raw := c.Query("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid customer_id: must be a valid UUID"))
			return
		}
		params.CustomerID = &id
	}

	for _, s := range c.QueryArray("status") {
		params.Status = append(params.Status, domain.SubscriptionStatus(s))
	}

	subs, total, err := h.svc.List(c.Request.Context(), orgID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionResponse, len(subs))
	for i := range subs {
		resp[i] = toSubResponse(&subs[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[subscriptionResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// GetByID handles GET /v1/subscriptions/:subscription_id.
func (h *SubscriptionHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	sub, err := h.svc.GetByID(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Cancel handles POST /v1/subscriptions/:subscription_id/cancel.
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req cancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	sub, err := h.svc.Cancel(c.Request.Context(), orgID, subID, req.Reason, req.CancelAtPeriodEnd)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Pause handles POST /v1/subscriptions/:subscription_id/pause.
func (h *SubscriptionHandler) Pause(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req pauseResumeRequest
	_ = c.ShouldBindJSON(&req) // optional body

	sub, err := h.svc.Pause(c.Request.Context(), orgID, subID, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// Resume handles POST /v1/subscriptions/:subscription_id/resume.
func (h *SubscriptionHandler) Resume(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	sub, err := h.svc.Resume(c.Request.Context(), orgID, subID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrInvalidSubscriptionTransition):
			_ = c.Error(apierr.Conflict(err.Error()))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusOK, toSubResponse(sub))
}

// ReplaceItems handles PUT /v1/subscriptions/:subscription_id/items.
func (h *SubscriptionHandler) ReplaceItems(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req []subscriptionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if len(req) == 0 {
		_ = c.Error(apierr.BadRequest("items must not be empty"))
		return
	}

	items := make([]domain.SubscriptionItemInput, len(req))
	for i, item := range req {
		q := item.Quantity
		if q == 0 {
			q = 1
		}
		items[i] = domain.SubscriptionItemInput{PlanID: item.PlanID, Quantity: q}
	}

	inserted, err := h.svc.ReplaceItems(c.Request.Context(), orgID, subID, items)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionItemResponse, len(inserted))
	for i, item := range inserted {
		resp[i] = subscriptionItemResponse{
			ID:        item.ID,
			PlanID:    item.PlanID,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ListItems handles GET /v1/subscriptions/:subscription_id/items.
func (h *SubscriptionHandler) ListItems(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	items, err := h.svc.ListItems(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionItemResponse, len(items))
	for i, item := range items {
		resp[i] = subscriptionItemResponse{
			ID:        item.ID,
			PlanID:    item.PlanID,
			Quantity:  item.Quantity,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

// ListHistory handles GET /v1/subscriptions/:subscription_id/history.
func (h *SubscriptionHandler) ListHistory(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	history, err := h.svc.ListHistory(c.Request.Context(), orgID, subID)
	if err != nil {
		if errors.Is(err, domain.ErrSubscriptionNotFound) {
			_ = c.Error(apierr.NotFound("subscription not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]subscriptionHistoryResponse, len(history))
	for i, h := range history {
		r := subscriptionHistoryResponse{
			ID:        h.ID,
			NewStatus: string(h.NewStatus),
			Reason:    h.Reason,
			CreatedAt: h.CreatedAt,
		}
		if h.OldStatus != nil {
			s := string(*h.OldStatus)
			r.OldStatus = &s
		}
		resp[i] = r
	}
	c.JSON(http.StatusOK, resp)
}

// CreateSchedule handles POST /v1/subscriptions/:subscription_id/schedules.
func (h *SubscriptionHandler) CreateSchedule(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	var req createScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	if req.TargetStatus == nil && req.TargetPlanID == nil {
		_ = c.Error(apierr.BadRequest("one of target_status or target_plan_id is required"))
		return
	}
	if req.TargetStatus != nil && req.TargetPlanID != nil {
		_ = c.Error(apierr.BadRequest("only one of target_status or target_plan_id may be provided"))
		return
	}

	input := domain.CreateScheduleInput{
		ScheduledAt:  req.ScheduledAt,
		TargetPlanID: req.TargetPlanID,
	}
	if req.TargetStatus != nil {
		s := domain.SubscriptionStatus(*req.TargetStatus)
		input.TargetStatus = &s
	}

	schedule, err := h.svc.CreateSchedule(c.Request.Context(), orgID, subID, input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		case errors.Is(err, domain.ErrSubscriptionCanceled):
			_ = c.Error(apierr.UnprocessableEntity("subscription is already canceled"))
		case errors.Is(err, domain.ErrScheduleAlreadyPending):
			_ = c.Error(apierr.Conflict("subscription already has a pending schedule"))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.JSON(http.StatusCreated, toScheduleResponse(schedule))
}

// CancelSchedule handles DELETE /v1/subscriptions/:subscription_id/schedules/pending.
func (h *SubscriptionHandler) CancelSchedule(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	subID, ok := dto.SubscriptionID(c)
	if !ok {
		return
	}

	if err := h.svc.CancelSchedule(c.Request.Context(), orgID, subID); err != nil {
		switch {
		case errors.Is(err, domain.ErrScheduleNotFound):
			_ = c.Error(apierr.NotFound("no pending schedule found"))
		case errors.Is(err, domain.ErrSubscriptionNotFound):
			_ = c.Error(apierr.NotFound("subscription not found"))
		default:
			_ = c.Error(err)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
