package handler

import (
	"errors"
	"net/http"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AddressHandler handles address-related HTTP endpoints.
type AddressHandler struct {
	addressSvc *service.AddressService
}

// NewAddressHandler creates an AddressHandler.
func NewAddressHandler(addressSvc *service.AddressService) *AddressHandler {
	return &AddressHandler{addressSvc: addressSvc}
}

// RegisterRoutes registers the address handler routes.
func (h *AddressHandler) RegisterRoutes(rg *gin.RouterGroup) {
	addresses := rg.Group("/customers/:customer_id/addresses")
	addresses.POST("", h.Create)
	addresses.GET("", h.List)
	addresses.GET("/:id", h.GetByID)
	addresses.PUT("/:id", h.Update)
	addresses.DELETE("/:id", h.Delete)
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
func (h *AddressHandler) Create(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	var req dto.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	addr, err := h.addressSvc.Create(c.Request.Context(), tid, cid, domain.AddressInput{
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

	c.JSON(http.StatusCreated, dto.ToAddressResponse(addr))
}

// List handles GET /v1/customers/:customer_id/addresses.
func (h *AddressHandler) List(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	addresses, err := h.addressSvc.List(c.Request.Context(), tid, cid, pag.ToListParams())
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]*dto.AddressResponse, len(addresses))
	for i := range addresses {
		resp[i] = dto.ToAddressResponse(&addresses[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[*dto.AddressResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// GetByID handles GET /v1/customers/:customer_id/addresses/:id.
func (h *AddressHandler) GetByID(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
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

	c.JSON(http.StatusOK, dto.ToAddressResponse(addr))
}

// Update handles PUT /v1/customers/:customer_id/addresses/:id.
func (h *AddressHandler) Update(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	aid, ok := addressID(c)
	if !ok {
		return
	}

	var req dto.CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	addr, err := h.addressSvc.Update(c.Request.Context(), tid, cid, aid, domain.AddressInput{
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

	c.JSON(http.StatusOK, dto.ToAddressResponse(addr))
}

// Delete handles DELETE /v1/customers/:customer_id/addresses/:id.
func (h *AddressHandler) Delete(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
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
