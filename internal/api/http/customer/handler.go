package customer

import (
	"encoding/json"
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

// Handler handles customer-related HTTP endpoints.
type Handler struct {
	customerSvc *service.CustomerService
}

// NewHandler creates a customer Handler.
func NewHandler(customerSvc *service.CustomerService) *Handler {
	return &Handler{customerSvc: customerSvc}
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

type createCustomerRequest struct {
	Name         string                `json:"name"          binding:"required"`
	Email        string                `json:"email"         binding:"required,email"`
	Phone        *string               `json:"phone"         binding:"omitempty,max=20"`
	Document     string                `json:"document"      binding:"required,min=1,max=20"`
	DocumentType string                `json:"document_type" binding:"required,min=1,max=20"`
	Metadata     json.RawMessage       `json:"metadata"      binding:"omitempty"`
	Address      *createAddressRequest `json:"address"       binding:"omitempty"`
}

type updateCustomerRequest struct {
	Name         string                `json:"name"          binding:"required"`
	Email        string                `json:"email"         binding:"required,email"`
	Phone        *string               `json:"phone"         binding:"omitempty,max=20"`
	Document     string                `json:"document"      binding:"required,min=1,max=20"`
	DocumentType string                `json:"document_type" binding:"required,min=1,max=20"`
	Metadata     json.RawMessage       `json:"metadata"      binding:"omitempty"`
	Address      *createAddressRequest `json:"address"       binding:"omitempty"`
}

type customerResponse struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Email        string          `json:"email"`
	Phone        *string         `json:"phone,omitempty"`
	Document     string          `json:"document"`
	DocumentType string          `json:"document_type"`
	Status       string          `json:"status"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
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

type createCustomerWithAddressResponse struct {
	customerResponse
	Address *addressResponse `json:"address,omitempty"`
}

func toAddressResponse(a *domain.Address) *addressResponse {
	if a == nil {
		return nil
	}
	return &addressResponse{
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

func toCustomerResponse(c *domain.Customer) customerResponse {
	return customerResponse{
		ID:           c.ID,
		Name:         c.Name,
		Email:        c.Email,
		Phone:        c.Phone,
		Document:     c.Document,
		DocumentType: c.DocumentType,
		Status:       c.Status,
		Metadata:     c.Metadata,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

// tenantID extracts the tenant UUID from the JWT claims stored in the context.
func tenantID(c *gin.Context) (uuid.UUID, bool) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return uuid.UUID{}, false
	}
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

// Create handles POST /v1/customers.
func (h *Handler) Create(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	var req createCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.CreateCustomerInput{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Document:     req.Document,
		DocumentType: req.DocumentType,
		Metadata:     req.Metadata,
	}

	if req.Address != nil {
		input.Address = &domain.CreateAddressInput{
			Type:         req.Address.Type,
			PostalCode:   req.Address.PostalCode,
			Street:       req.Address.Street,
			Number:       req.Address.Number,
			Complement:   req.Address.Complement,
			Neighborhood: req.Address.Neighborhood,
			City:         req.Address.City,
			State:        req.Address.State,
			Country:      req.Address.Country,
			IsMain:       req.Address.IsMain,
		}
	}

	cust, addr, err := h.customerSvc.Create(c.Request.Context(), tid, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	if addr != nil {
		c.JSON(http.StatusCreated, createCustomerWithAddressResponse{
			customerResponse: toCustomerResponse(cust),
			Address:          toAddressResponse(addr),
		})
		return
	}

	c.JSON(http.StatusCreated, toCustomerResponse(cust))
}

// List handles GET /v1/customers.
func (h *Handler) List(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	customers, err := h.customerSvc.List(c.Request.Context(), tid)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]customerResponse, len(customers))
	for i := range customers {
		resp[i] = toCustomerResponse(&customers[i])
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID handles GET /v1/customers/:id.
func (h *Handler) GetByID(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	cust, err := h.customerSvc.GetByID(c.Request.Context(), tid, cid)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(cust))
}

// Update handles PUT /v1/customers/:id.
func (h *Handler) Update(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	var req updateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.UpdateCustomerInput{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Document:     req.Document,
		DocumentType: req.DocumentType,
		Metadata:     req.Metadata,
	}

	if req.Address != nil {
		input.Address = &domain.CreateAddressInput{
			Type:         req.Address.Type,
			PostalCode:   req.Address.PostalCode,
			Street:       req.Address.Street,
			Number:       req.Address.Number,
			Complement:   req.Address.Complement,
			Neighborhood: req.Address.Neighborhood,
			City:         req.Address.City,
			State:        req.Address.State,
			Country:      req.Address.Country,
			IsMain:       req.Address.IsMain,
		}
	}

	cust, addr, err := h.customerSvc.Update(c.Request.Context(), tid, cid, input)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	if addr != nil {
		c.JSON(http.StatusOK, createCustomerWithAddressResponse{
			customerResponse: toCustomerResponse(cust),
			Address:          toAddressResponse(addr),
		})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(cust))
}

// Delete handles DELETE /v1/customers/:id.
func (h *Handler) Delete(c *gin.Context) {
	tid, ok := tenantID(c)
	if !ok {
		return
	}

	cid, ok := customerID(c)
	if !ok {
		return
	}

	err := h.customerSvc.Delete(c.Request.Context(), tid, cid)
	if err != nil {
		if errors.Is(err, domain.ErrCustomerNotFound) {
			_ = c.Error(apierr.NotFound("customer not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
