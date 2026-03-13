# Development Guide

## Estrutura de Pastas

```
i9pay/
├── cmd/                      # Aplicações executáveis
│   └── api/                  # API principal
├── internal/                 # Código privado (não importável)
│   ├── api/                  # Camada HTTP
│   │   ├── router/           # Roteador e inicialização
│   │   ├── middleware/       # Middlewares (auth, errors, etc)
│   │   ├── http/             # Handlers por feature
│   │   └── err/              # Tipos de erro customizados
│   ├── service/              # Lógica de negócio
│   ├── domain/               # Modelos e validações
│   ├── config/               # Configurações
│   ├── database/             # Conexão e queries (SQLC)
│   └── token/                # JWT service
├── migrations/               # SQL migrations
├── docker/                   # Dockerfiles e compose
│   ├── dev/                  # Development environment
│   └── prod/                 # Production environment
├── docs/					  # Documentação
├── go.mod / go.sum           # Dependências
├── sqlc.yml                  # Configuração SQLC
└── Makefile                  # Comandos úteis
```

**Nota**: Cada _feature_ cria subpacotes específicas, não há código genérico no `internal/`.

## Adicionando Nova Feature

Exemplo: Implementar CRUD de Produtos.

### 1. Estrutura

```bash
mkdir -p internal/api/http/products
mkdir -p internal/service
```

### 2. Handler (`internal/api/http/products/handler.go`)

```go
package products

import (
	"i9pay/internal/api/err"
	"i9pay/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	productSvc *service.ProductService
}

func NewHandler(productSvc *service.ProductService) *Handler {
	return &Handler{productSvc: productSvc}
}

type CreateRequest struct {
	Name  string `json:"name" binding:"required"`
	Price int64  `json:"price" binding:"required,min=0"`
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(err.BadRequest(e.Error()))
		return
	}

	product, e := h.productSvc.Create(c.Request.Context(), req.Name, req.Price)
	if e != nil {
		_ = c.Error(err.Internal(e.Error()))
		return
	}

	c.JSON(http.StatusCreated, product)
}

func (h *Handler) Get(c *gin.Context) {
	id, e := uuid.Parse(c.Param("id"))
	if e != nil {
		_ = c.Error(err.BadRequest("invalid product id"))
		return
	}

	product, e := h.productSvc.Get(c.Request.Context(), id)
	if e != nil {
		_ = c.Error(err.NotFound("product not found"))
		return
	}

	c.JSON(http.StatusOK, product)
}
```

### 3. Service (`internal/service/product.go`)

```go
package service

import (
	"context"
	"i9pay/internal/database"
	"time"

	"github.com/google/uuid"
)

type Product struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductService struct {
	db *database.DB
}

func NewProductService(db *database.DB) *ProductService {
	return &ProductService{db: db}
}

func (s *ProductService) Create(ctx context.Context, name string, price int64) (*Product, error) {
	// TODO: Usar s.db.Queries.CreateProduct() com query gerada pelo SQLC
	p := &Product{
		ID:        uuid.New(),
		Name:      name,
		Price:     price,
		CreatedAt: time.Now(),
	}
	return p, nil
}

func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (*Product, error) {
	// TODO: Usar s.db.Queries.GetProduct(ctx, id) com query gerada pelo SQLC
	return nil, nil
}
```

### 4. Domain (`internal/domain/product.go`) - Opcional

Se tiver validação complexa ou modelos compartilhados:

```go
package domain

import (
	"errors"

	"github.com/google/uuid"
)

var ErrInvalidPrice = errors.New("price must be positive")

type Product struct {
	ID    uuid.UUID
	Name  string
	Price int64
}

func (p *Product) IsValid() error {
	if p.Price <= 0 {
		return ErrInvalidPrice
	}
	return nil
}
```

### 5. SQL Queries (`internal/database/queries.sql`)

```sql
-- name: CreateProduct :one
INSERT INTO products (id, tenant_id, name, price, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, name, price, created_at;

-- name: GetProduct :one
SELECT id, tenant_id, name, price, created_at
FROM products
WHERE id = $1 AND tenant_id = $2;

-- name: ListProducts :many
SELECT id, tenant_id, name, price, created_at
FROM products
WHERE tenant_id = $1
ORDER BY created_at DESC;
```

Gere o código:
```bash
make sqlc
```

### 6. Registrar no Router (`internal/api/router/router.go`)

```go
// No setup de handlers
productSvc := service.NewProductService(db)
productHandler := products.NewHandler(productSvc)

// Adicione rotas
protected := v1.Group("/")
protected.Use(middleware.Auth(tokenSvc))
{
	protected.GET("/me", meHandler.Get)
	
	// Novo: Produtos
	productGroup := protected.Group("/products")
	{
		productGroup.POST("", productHandler.Create)
		productGroup.GET("/:id", productHandler.Get)
	}
}
```

---

## Padrões de Código

### Naming

- **Packages**: lowercase, sem underscore (`internal/api/http/products`)
- **Funcs/Methods**: camelCase (`createProduct`, `fecthTenantById`)
- **Consts**: UPPER_SNAKE_CASE (`DEFAULT_TTL`, `MAX_RETRY`)
- **Vars**: camelCase (`productID`, `isValid`)

### Error Handling

❌ **Evite:**
```go
if err != nil {
	return err  // Erro sem contexto
}
```

✅ **Prefira:**
```go
if err != nil {
	return fmt.Errorf("fetch product: %w", err)  // Wrapped com contexto
}
```

### Dependency Injection

Sempre explícita no constructor:

```go
// ✅ Bom
func NewProductService(db *database.DB, logger Logger) *ProductService {
	return &ProductService{db, logger}
}

// ❌ Evite
var globalDB *database.DB  // Global state
func NewProductService() *ProductService {
	return &ProductService{globalDB}
}
```

### Tests

Organize em `*_test.go` próximo ao código:

```
internal/
├── service/
│   ├── product.go
│   └── product_test.go      # Tests aqui
├── api/http/products/
│   ├── handler.go
│   └── handler_test.go      # Tests aqui
```

Exemplo test:

```go
func TestProductServiceCreate(t *testing.T) {
	db := setupTestDB(t)
	svc := service.NewProductService(db)

	product, err := svc.Create(context.Background(), "Widget", 1000)

	assert.NoError(t, err)
	assert.Equal(t, "Widget", product.Name)
	assert.Equal(t, int64(1000), product.Price)
}
```

---

## Migrations

### Criar Migration

```bash
make migrate-create name=add_products_table
```

Cria arquivo: `migrations/000004_add_products_table.up.sql`

### Escrever SQL

```sql
CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL CHECK (price > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_products_tenant_id ON products(tenant_id);
```

### Aplicar

```bash
make migrate-up
```

### Reverter

```bash
make migrate-down
```

---

## Testes

### Rodar Testes

```bash
# Todos
go test ./...

# Pacote específico
go test ./internal/service/...

# Verbose
go test -v ./...

# Com coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Mock de Dependências

Use [testify/mock](https://pkg.go.dev/github.com/stretchr/testify/mock):

```bash
go get github.com/stretchr/testify
```

Exemplo:

```go
import "github.com/stretchr/testify/mock"

type MockDB struct {
	mock.Mock
}

func (m *MockDB) GetProduct(ctx context.Context, id uuid.UUID) (Product, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(Product), args.Error(1)
}

func TestServiceGetProduct(t *testing.T) {
	mockDB := &MockDB{}
	mockDB.On("GetProduct", mock.Anything, mock.Anything).Return(Product{...}, nil)

	svc := NewProductService(mockDB)
	product, err := svc.Get(context.Background(), uuid.New())

	assert.NoError(t, err)
	assert.Equal(t, "Widget", product.Name)
}
```

---

## Code Quality

### Formatting

```bash
# Format all
go fmt ./...

# Automático no save (VSCode)
# Addione ao .vscode/settings.json:
"[go]": {
	"editor.formatOnSave": true,
	"editor.codeActionsOnSave": {
	"source.organizeImports": "always"
	}
},
```

### Linting

```bash
# Instale
golangci-lint run

# No Makefile:
make lint
```

### Vet

```bash
go vet ./...
```

---

## Committing

### Commit Message Format

```
feature: add product CRUD endpoints

- Create product with POST /v1/products
- Get product with GET /v1/products/{id}
- Add ProductService with database integration
```

---

## Recursos Úteis

- [Effective Go](https://golang.org/doc/effective_go)
- [Gin Framework](https://gin-gonic.com/)
- [SQLC Documentation](https://sqlc.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [JWT.io](https://jwt.io/)
