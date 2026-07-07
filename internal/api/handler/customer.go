package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"payflow/internal/api/dto"
	apierr "payflow/internal/api/err"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CustomerHandler handles customer-related HTTP endpoints.
type CustomerHandler struct {
	customerSvc *service.CustomerService
}

// NewCustomerHandler creates a CustomerHandler.
func NewCustomerHandler(customerSvc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{customerSvc: customerSvc}
}

// RegisterRoutes registers the customer handler routes.
func (h *CustomerHandler) RegisterRoutes(rg *gin.RouterGroup) {
	customers := rg.Group("/customers")
	customers.POST("", h.Create)
	customers.GET("", h.List)
	customers.GET("/:customer_id", h.GetByID)
	customers.PUT("/:customer_id", h.Update)
	customers.DELETE("/:customer_id", h.Delete)
}

// customerRequest is the shared DTO for creating and updating a customer.
// Both operations require the same fields and validations.
type customerRequest struct {
	Name         string                    `json:"name"          binding:"required"`
	Email        string                    `json:"email"         binding:"required,email"`
	Phone        *string                   `json:"phone"         binding:"omitempty,max=20"`
	Document     string                    `json:"document"      binding:"required,min=1,max=20"`
	DocumentType string                    `json:"document_type" binding:"required,min=1,max=20"`
	Metadata     json.RawMessage           `json:"metadata"      binding:"omitempty"`
	Address      *dto.CreateAddressRequest `json:"address"       binding:"omitempty"`
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

type createCustomerWithAddressResponse struct {
	customerResponse
	Address *dto.AddressResponse `json:"address,omitempty"`
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

// Create handles POST /v1/customers.
func (h *CustomerHandler) Create(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.CustomerInput{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Document:     req.Document,
		DocumentType: req.DocumentType,
		Metadata:     req.Metadata,
	}

	if req.Address != nil {
		input.Address = &domain.AddressInput{
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
			Address:          dto.ToAddressResponse(addr),
		})
		return
	}

	c.JSON(http.StatusCreated, toCustomerResponse(cust))
}

// List handles GET /v1/customers.
func (h *CustomerHandler) List(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	customers, err := h.customerSvc.List(c.Request.Context(), tid, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]customerResponse, len(customers))
	for i := range customers {
		resp[i] = toCustomerResponse(&customers[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[customerResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// GetByID handles GET /v1/customers/:id.
func (h *CustomerHandler) GetByID(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
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
func (h *CustomerHandler) Update(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
	if !ok {
		return
	}

	var req customerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.CustomerInput{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Document:     req.Document,
		DocumentType: req.DocumentType,
		Metadata:     req.Metadata,
	}

	if req.Address != nil {
		input.Address = &domain.AddressInput{
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
			Address:          dto.ToAddressResponse(addr),
		})
		return
	}

	c.JSON(http.StatusOK, toCustomerResponse(cust))
}

// Delete handles DELETE /v1/customers/:id.
func (h *CustomerHandler) Delete(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	cid, ok := dto.CustomerID(c)
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
