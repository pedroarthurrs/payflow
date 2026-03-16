package address

import (
	"errors"
	"net/http"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"
	"i9pay/internal/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler handles address-related HTTP endpoints.
type Handler struct {
	addressSvc *service.AddressService
}

// NewHandler creates an address Handler.
func NewHandler(addressSvc *service.AddressService) *Handler {
	return &Handler{addressSvc: addressSvc}
}

type createAddressRequest struct {
	Type         *string `json:"type"         binding:"omitempty,oneof=billing shipping home"`
	PostalCode   string  `json:"postal_code"  binding:"required,max=10"`
	Street       string  `json:"street"       binding:"required"`
	Number       *string `json:"number"       binding:"omitempty,max=10"`
	Complement   *string `json:"complement"   binding:"omitempty,max=100"`
	Neighborhood *string `json:"neighborhood" binding:"omitempty,max=100"`
	City         string  `json:"city"         binding:"required,max=100"`
	State        string  `json:"state"        binding:"required,len=2"`
	Country      *string `json:"country"      binding:"omitempty,len=2"`
	IsMain       *bool   `json:"is_main"`
}

type updateAddressRequest struct {
	Type         *string `json:"type"         binding:"omitempty,oneof=billing shipping home"`
	PostalCode   string  `json:"postal_code"  binding:"required,max=10"`
	Street       string  `json:"street"       binding:"required"`
	Number       *string `json:"number"       binding:"omitempty,max=10"`
	Complement   *string `json:"complement"   binding:"omitempty,max=100"`
	Neighborhood *string `json:"neighborhood" binding:"omitempty,max=100"`
	City         string  `json:"city"         binding:"required,max=100"`
	State        string  `json:"state"        binding:"required,len=2"`
	Country      *string `json:"country"      binding:"omitempty,len=2"`
	IsMain       *bool   `json:"is_main"`
}

type addressResponse struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	Type         string    `json:"type"`
	PostalCode   string    `json:"postal_code"`
	Street       string    `json:"street"`
	Number       *string   `json:"number,omitempty"`
	Complement   *string   `json:"complement,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	City         string    `json:"city"`
	State        string    `json:"state"`
	Country      string    `json:"country"`
	IsMain       bool      `json:"is_main"`
	CreatedAt    time.Time `json:"created_at"`
}

func toAddressResponse(a *domain.Address) addressResponse {
	return addressResponse{
		ID:           a.ID,
		CustomerID:   a.CustomerID,
		Type:         a.Type,
		PostalCode:   a.PostalCode,
		Street:       a.Street,
		Number:       a.Number,
		Complement:   a.Complement,
		Neighborhood: a.Neighborhood,
		City:         a.City,
		State:        a.State,
		Country:      a.Country,
		IsMain:       a.IsMain,
		CreatedAt:    a.CreatedAt,
	}
}

// tenantID extracts the tenant UUID from the JWT claims stored in the context.
func tenantID(c *gin.Context) (uuid.UUID, bool) {
	claims := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	tid, err := uuid.Parse(claims.TenantID)
	if err != nil {
		_ = c.Error(apierr.Internal("invalid tenant id in claims"))
		return uuid.UUID{}, false
	}
	return tid, true
}

// customerID parses the ":customer_id" path parameter as a UUID.
func customerID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid customer id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// addressID parses the ":id" path parameter as a UUID.
func addressID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid address id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// Create handles POST /v1/customers/:customer_id/addresses.
func (h *Handler) Create(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	var req createAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	addr, err := h.addressSvc.Create(c.Request.Context(), tid, cid, domain.CreateAddressInput{
		Type:         req.Type,
		PostalCode:   req.PostalCode,
		Street:       req.Street,
		Number:       req.Number,
		Complement:   req.Complement,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
		Country:      req.Country,
		IsMain:       req.IsMain,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toAddressResponse(addr))
}

// List handles GET /v1/customers/:customer_id/addresses.
func (h *Handler) List(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	addresses, err := h.addressSvc.List(c.Request.Context(), tid, cid)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]addressResponse, len(addresses))
	for i := range addresses {
		resp[i] = toAddressResponse(&addresses[i])
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID handles GET /v1/customers/:customer_id/addresses/:id.
func (h *Handler) GetByID(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	aid, ok := addressID(c)
	if !ok {
		return
	}

	addr, err := h.addressSvc.GetByID(c.Request.Context(), tid, cid, aid)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		if errors.Is(err, domain.ErrAddressNotFound) {
			_ = c.Error(apierr.NotFound("address not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toAddressResponse(addr))
}

// Update handles PUT /v1/customers/:customer_id/addresses/:id.
func (h *Handler) Update(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	aid, ok := addressID(c)
	if !ok {
		return
	}

	var req updateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	addr, err := h.addressSvc.Update(c.Request.Context(), tid, cid, aid, domain.UpdateAddressInput{
		Type:         req.Type,
		PostalCode:   req.PostalCode,
		Street:       req.Street,
		Number:       req.Number,
		Complement:   req.Complement,
		Neighborhood: req.Neighborhood,
		City:         req.City,
		State:        req.State,
		Country:      req.Country,
		IsMain:       req.IsMain,
	})
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		if errors.Is(err, domain.ErrAddressNotFound) {
			_ = c.Error(apierr.NotFound("address not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toAddressResponse(addr))
}

// Delete handles DELETE /v1/customers/:customer_id/addresses/:id.
func (h *Handler) Delete(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	aid, ok := addressID(c)
	if !ok {
		return
	}

	err := h.addressSvc.Delete(c.Request.Context(), tid, cid, aid)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		if errors.Is(err, domain.ErrAddressNotFound) {
			_ = c.Error(apierr.NotFound("address not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
