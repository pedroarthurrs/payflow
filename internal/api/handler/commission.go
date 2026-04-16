package handler

import (
	"errors"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CommissionHandler handles commission setting endpoints.
type CommissionHandler struct {
	wallets domain.WalletRepository
	permSvc middleware.PermissionResolver
}

// NewCommissionHandler creates a CommissionHandler.
func NewCommissionHandler(walletRepo domain.WalletRepository, permSvc middleware.PermissionResolver) *CommissionHandler {
	return &CommissionHandler{wallets: walletRepo, permSvc: permSvc}
}

// RegisterRoutes registers commission routes on the protected group.
func (h *CommissionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	commissions := rg.Group("/commissions")
	commissions.GET("", middleware.RequirePermission(h.permSvc, domain.PermCommissionsRead), h.List)
	commissions.PUT("/:customer_id", middleware.RequirePermission(h.permSvc, domain.PermCommissionsManage), h.Upsert)
	commissions.DELETE("/:customer_id", middleware.RequirePermission(h.permSvc, domain.PermCommissionsManage), h.Delete)
}

// --- DTOs ---

type upsertCommissionRequest struct {
	Type     string  `json:"type"       binding:"required,oneof=percentage fixed"`
	Value    float64 `json:"value"      binding:"required,min=0"`
	IsActive *bool   `json:"is_active"`
}

type commissionSettingResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	TargetCustomerID uuid.UUID `json:"target_customer_id"`
	Type             string    `json:"type"`
	Value            float64   `json:"value"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func toCommissionSettingResponse(cs domain.CommissionSetting) commissionSettingResponse {
	return commissionSettingResponse{
		ID:               cs.ID,
		OrganizationID:   cs.OrganizationID,
		TargetCustomerID: cs.TargetCustomerID,
		Type:             string(cs.Type),
		Value:            cs.Value,
		IsActive:         cs.IsActive,
		CreatedAt:        cs.CreatedAt,
		UpdatedAt:        cs.UpdatedAt,
	}
}

// --- Handlers ---

// List handles GET /v1/commissions
func (h *CommissionHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	settings, err := h.wallets.ListCommissionSettings(c.Request.Context(), orgID, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]commissionSettingResponse, len(settings))
	for i := range settings {
		resp[i] = toCommissionSettingResponse(settings[i])
	}
	c.JSON(http.StatusOK, dto.PagedResponse[commissionSettingResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  int64(len(resp)),
	})
}

// Upsert handles PUT /v1/commissions/:customer_id
func (h *CommissionHandler) Upsert(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	customerID, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	var req upsertCommissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	setting, err := h.wallets.UpsertCommissionSetting(c.Request.Context(), orgID, domain.CommissionSetting{
		OrganizationID:   orgID,
		TargetCustomerID: customerID,
		Type:             domain.CommissionType(req.Type),
		Value:            req.Value,
		IsActive:         isActive,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toCommissionSettingResponse(*setting))
}

// Delete handles DELETE /v1/commissions/:customer_id
func (h *CommissionHandler) Delete(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	customerID, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	err := h.wallets.SoftDeleteCommissionSetting(c.Request.Context(), orgID, customerID)
	if err != nil {
		if errors.Is(err, domain.ErrCommissionNotFound) {
			_ = c.Error(apierr.NotFound("commission setting not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}
