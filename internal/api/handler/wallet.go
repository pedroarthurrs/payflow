package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"payflow/internal/api/dto"
	apierr "payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WalletHandler handles wallet-related HTTP endpoints.
type WalletHandler struct {
	walletSvc *service.WalletService
	permSvc   middleware.PermissionResolver
}

// NewWalletHandler creates a WalletHandler.
func NewWalletHandler(svc *service.WalletService, permSvc middleware.PermissionResolver) *WalletHandler {
	return &WalletHandler{walletSvc: svc, permSvc: permSvc}
}

// RegisterRoutes registers wallet routes on the protected group.
func (h *WalletHandler) RegisterRoutes(rg *gin.RouterGroup) {
	wallets := rg.Group("/wallets")
	wallets.GET("/me", middleware.RequirePermission(h.permSvc, domain.PermWalletsRead), h.GetOrgWallet)
	wallets.GET("/:wallet_id", middleware.RequirePermission(h.permSvc, domain.PermWalletsRead), h.GetByID)
	wallets.GET("/:wallet_id/entries", middleware.RequirePermission(h.permSvc, domain.PermWalletsRead), h.ListEntries)
	wallets.POST("/:wallet_id/payouts", middleware.RequirePermission(h.permSvc, domain.PermWalletsPayout), middleware.RequireIdempotency(), h.RequestPayout)
	wallets.GET("/:wallet_id/payouts", middleware.RequirePermission(h.permSvc, domain.PermWalletsRead), h.ListPayouts)
	wallets.POST("/:wallet_id/adjust", middleware.RequirePermission(h.permSvc, domain.PermWalletsAdjust), middleware.RequireIdempotency(), h.Adjust)
}

// --- DTOs ---

type walletResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organization_id"`
	Balance          int32     `json:"balance"`
	LockedBalance    int32     `json:"locked_balance"`
	AvailableBalance int32     `json:"available_balance"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func toWalletResponse(w *domain.Wallet) walletResponse {
	return walletResponse{
		ID:               w.ID,
		OrganizationID:   w.OrganizationID,
		Balance:          w.Balance,
		LockedBalance:    w.LockedBalance,
		AvailableBalance: w.Balance - w.LockedBalance,
		Currency:         w.Currency,
		Status:           string(w.Status),
		CreatedAt:        w.CreatedAt,
		UpdatedAt:        w.UpdatedAt,
	}
}

type ledgerEntryResponse struct {
	ID           uuid.UUID       `json:"id"`
	WalletID     uuid.UUID       `json:"wallet_id"`
	Type         string          `json:"type"`
	AmountCents  int32           `json:"amount_cents"`
	Category     string          `json:"category"`
	BalanceAfter int32           `json:"balance_after"`
	PaymentID    *uuid.UUID      `json:"payment_id,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func toLedgerEntryResponse(e domain.LedgerEntry) ledgerEntryResponse {
	return ledgerEntryResponse{
		ID:           e.ID,
		WalletID:     e.WalletID,
		Type:         string(e.Type),
		AmountCents:  e.AmountCents,
		Category:     string(e.Category),
		BalanceAfter: e.BalanceAfter,
		PaymentID:    e.PaymentID,
		Metadata:     e.Metadata,
		CreatedAt:    e.CreatedAt,
	}
}

type payoutResponse struct {
	ID          uuid.UUID       `json:"id"`
	WalletID    uuid.UUID       `json:"wallet_id"`
	AmountCents int32           `json:"amount_cents"`
	FeeCents    int32           `json:"fee_cents"`
	Status      string          `json:"status"`
	BankInfo    json.RawMessage `json:"bank_info,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func toPayoutResponse(p domain.Payout) payoutResponse {
	return payoutResponse{
		ID:          p.ID,
		WalletID:    p.WalletID,
		AmountCents: p.AmountCents,
		FeeCents:    p.FeeCents,
		Status:      string(p.Status),
		BankInfo:    p.BankInfo,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

type requestPayoutRequest struct {
	AmountCents int32           `json:"amount_cents" binding:"required,min=1"`
	FeeCents    int32           `json:"fee_cents"    binding:"min=0"`
	BankInfo    json.RawMessage `json:"bank_info"`
}

type adjustRequest struct {
	Direction   string `json:"direction"    binding:"required,oneof=credit debit"`
	AmountCents int32  `json:"amount_cents" binding:"required,min=1"`
	Reason      string `json:"reason"       binding:"required,min=1"`
}

// --- Handlers ---

// GetOrgWallet handles GET /v1/wallets/me
func (h *WalletHandler) GetOrgWallet(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	wallet, err := h.walletSvc.GetOrgWallet(c.Request.Context(), orgID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toWalletResponse(wallet))
}

// GetByID handles GET /v1/wallets/:wallet_id
func (h *WalletHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	walletID, ok := dto.WalletID(c)
	if !ok {
		return
	}
	wallet, err := h.walletSvc.GetByID(c.Request.Context(), orgID, walletID)
	if err != nil {
		if errors.Is(err, domain.ErrWalletNotFound) {
			_ = c.Error(apierr.NotFound("wallet not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toWalletResponse(wallet))
}

// ListEntries handles GET /v1/wallets/:wallet_id/entries
func (h *WalletHandler) ListEntries(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	walletID, ok := dto.WalletID(c)
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

	params := domain.LedgerListParams{
		ListParams:  pag.ToListParams(),
		CreatedFrom: dateRange.From,
		CreatedTo:   dateRange.To,
	}
	if raw := c.Query("category"); raw != "" {
		cat := domain.LedgerCategory(raw)
		params.Category = &cat
	}
	if raw := c.Query("type"); raw != "" {
		t := domain.LedgerEntryType(raw)
		params.Type = &t
	}

	entries, total, err := h.walletSvc.ListEntries(c.Request.Context(), orgID, walletID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]ledgerEntryResponse, len(entries))
	for i := range entries {
		resp[i] = toLedgerEntryResponse(entries[i])
	}
	c.JSON(http.StatusOK, dto.PagedResponse[ledgerEntryResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// RequestPayout handles POST /v1/wallets/:wallet_id/payouts
func (h *WalletHandler) RequestPayout(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	walletID, ok := dto.WalletID(c)
	if !ok {
		return
	}

	var req requestPayoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	payout, err := h.walletSvc.RequestPayout(c.Request.Context(), orgID, domain.CreatePayoutInput{
		WalletID:    walletID,
		AmountCents: req.AmountCents,
		FeeCents:    req.FeeCents,
		BankInfo:    req.BankInfo,
	})
	if err != nil {
		mapWalletError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toPayoutResponse(*payout))
}

// ListPayouts handles GET /v1/wallets/:wallet_id/payouts
func (h *WalletHandler) ListPayouts(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	walletID, ok := dto.WalletID(c)
	if !ok {
		return
	}
	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	payouts, err := h.walletSvc.ListPayouts(c.Request.Context(), orgID, walletID, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]payoutResponse, len(payouts))
	for i := range payouts {
		resp[i] = toPayoutResponse(payouts[i])
	}
	c.JSON(http.StatusOK, dto.PagedResponse[payoutResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  int64(len(resp)),
	})
}

// Adjust handles POST /v1/wallets/:wallet_id/adjust
func (h *WalletHandler) Adjust(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	walletID, ok := dto.WalletID(c)
	if !ok {
		return
	}

	var req adjustRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	wallet, err := h.walletSvc.Adjust(c.Request.Context(), orgID, walletID, domain.AdjustWalletInput{
		Direction:   domain.LedgerEntryType(req.Direction),
		AmountCents: req.AmountCents,
		Reason:      req.Reason,
	})
	if err != nil {
		mapWalletError(c, err)
		return
	}
	c.JSON(http.StatusOK, toWalletResponse(wallet))
}

func mapWalletError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrWalletNotFound):
		_ = c.Error(apierr.NotFound("wallet not found"))
	case errors.Is(err, domain.ErrPayoutNotFound):
		_ = c.Error(apierr.NotFound("payout not found"))
	case errors.Is(err, domain.ErrWalletSuspended):
		_ = c.Error(apierr.UnprocessableEntity("wallet is suspended"))
	case errors.Is(err, domain.ErrInsufficientBalance):
		_ = c.Error(apierr.UnprocessableEntity("insufficient balance"))
	case errors.Is(err, domain.ErrInvalidAdjustment):
		_ = c.Error(apierr.UnprocessableEntity("invalid adjustment"))
	default:
		_ = c.Error(err)
	}
}
