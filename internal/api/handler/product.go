package handler

import (
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

// ProductHandler handles product-related HTTP endpoints.
type ProductHandler struct {
	productSvc *service.ProductService
}

// NewProductHandler creates a ProductHandler.
func NewProductHandler(productSvc *service.ProductService) *ProductHandler {
	return &ProductHandler{productSvc: productSvc}
}

// RegisterRoutes registers the product handler routes.
func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
	products := rg.Group("/products")
	products.POST("", h.Create)
	products.GET("", h.List)
	products.GET("/:product_id", h.GetByID)
	products.PUT("/:product_id", h.Update)
	products.DELETE("/:product_id", h.Delete)
}

type productRequest struct {
	Name   string `json:"name"   binding:"required"`
	Active *bool  `json:"active" binding:"required"`
}

type productResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toProductResponse(p *domain.Product) productResponse {
	return productResponse{
		ID:        p.ID,
		Name:      p.Name,
		Active:    p.Active,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// Create handles POST /v1/products.
func (h *ProductHandler) Create(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.ProductInput{
		Name:   req.Name,
		Active: *req.Active,
	}

	product, err := h.productSvc.Create(c.Request.Context(), tid, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toProductResponse(product))
}

// List handles GET /v1/products.
func (h *ProductHandler) List(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	products, err := h.productSvc.List(c.Request.Context(), tid, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]productResponse, len(products))
	for i := range products {
		resp[i] = toProductResponse(&products[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[productResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// GetByID handles GET /v1/products/:product_id.
func (h *ProductHandler) GetByID(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}

	product, err := h.productSvc.GetByID(c.Request.Context(), tid, pid)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			_ = c.Error(apierr.NotFound("product not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toProductResponse(product))
}

// Update handles PUT /v1/products/:product_id.
func (h *ProductHandler) Update(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.ProductInput{
		Name:   req.Name,
		Active: *req.Active,
	}

	product, err := h.productSvc.Update(c.Request.Context(), tid, pid, input)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			_ = c.Error(apierr.NotFound("product not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toProductResponse(product))
}

// Delete handles DELETE /v1/products/:product_id.
func (h *ProductHandler) Delete(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.ProductID(c)
	if !ok {
		return
	}

	err := h.productSvc.Delete(c.Request.Context(), tid, pid)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			_ = c.Error(apierr.NotFound("product not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
