# Guia de Desenvolvimento

## Estrutura de Pastas

```
i9pay/
├── cmd/
│   └── server/               — servidor HTTP principal
├── internal/
│   ├── api/
│   │   ├── router/           — roteador e inicialização
│   │   ├── middleware/       — middlewares (auth, errors, logger, etc.)
│   │   ├── handler/          — handlers consolidados (um arquivo por recurso)
│   │   ├── dto/              — Data Transfer Objects compartilhados
│   │   └── err/              — tipos de erro customizados
│   ├── service/              — lógica de negócio
│   ├── domain/               — modelos e validações
│   ├── config/               — configurações
│   ├── database/             — conexão e queries (SQLC)
│   └── auth/token/           — JWT service
├── build/
│   └── docker/
│       ├── dev/              — Dockerfile de desenvolvimento
│       └── prod/             — Dockerfile de produção
├── deploy/
│   ├── docker/
│   │   ├── dev/              — docker-compose de desenvolvimento
│   │   └── prod/             — docker-compose de produção
│   └── migrations/           — SQL migrations
├── docs/                     — documentação
├── go.mod / go.sum           — dependências
├── sqlc.yml                  — configuração SQLC
└── Makefile                  — comandos de build e desenvolvimento
```

Handlers ficam consolidados em `internal/api/handler/` (um arquivo por recurso). DTOs compartilhados ficam em `internal/api/dto/`.

---

## Adicionando Nova Feature

Exemplo: implementar CRUD de Produtos.

### 1. Handler (`internal/api/handler/product.go`)

```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    apierr "i9pay/internal/api/err"
    "i9pay/internal/service"
)

type ProductHandler struct {
    productSvc *service.ProductService
}

func NewProductHandler(productSvc *service.ProductService) *ProductHandler {
    return &ProductHandler{productSvc: productSvc}
}

// RegisterRoutes registra as rotas do handler no grupo fornecido.
func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
    products := rg.Group("/products")
    products.POST("/", h.Create)
    products.GET("/:id", h.Get)
}

type createProductRequest struct {
    Name  string `json:"name" binding:"required"`
    Price int64  `json:"price" binding:"required,min=0"`
}

func (h *ProductHandler) Create(c *gin.Context) {
    var req createProductRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        _ = c.Error(apierr.BadRequest(err.Error()))
        return
    }

    product, err := h.productSvc.Create(c.Request.Context(), req.Name, req.Price)
    if err != nil {
        _ = c.Error(apierr.Internal(err.Error()))
        return
    }

    c.JSON(http.StatusCreated, product)
}

func (h *ProductHandler) Get(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        _ = c.Error(apierr.BadRequest("invalid product id"))
        return
    }

    product, err := h.productSvc.Get(c.Request.Context(), id)
    if err != nil {
        _ = c.Error(apierr.NotFound("product not found"))
        return
    }

    c.JSON(http.StatusOK, product)
}
```

### 2. Service (`internal/service/product.go`)

```go
package service

import (
    "context"
    "time"

    "github.com/google/uuid"

    "i9pay/internal/database"
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
    // Use s.db.Queries.CreateProduct() com query gerada pelo SQLC
    return &Product{
        ID:        uuid.New(),
        Name:      name,
        Price:     price,
        CreatedAt: time.Now(),
    }, nil
}

func (s *ProductService) Get(ctx context.Context, id uuid.UUID) (*Product, error) {
    // Use s.db.Queries.GetProduct(ctx, id) com query gerada pelo SQLC
    return nil, nil
}
```

### 3. Domain (`internal/domain/product.go`)

Opcional. Use quando houver validação complexa ou modelos compartilhados entre camadas.

```go
package domain

import "errors"

var ErrInvalidPrice = errors.New("price must be positive")

type Product struct {
    ID    string
    Name  string
    Price int64
}

func (p *Product) Validate() error {
    if p.Price <= 0 {
        return ErrInvalidPrice
    }
    return nil
}
```

### 4. SQL Queries (`internal/database/queries.sql`)

```sql
-- name: CreateProduct :one
INSERT INTO products (id, organization_id, name, price, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, organization_id, name, price, created_at;

-- name: GetProduct :one
SELECT id, organization_id, name, price, created_at
FROM products
WHERE id = $1 AND organization_id = $2;

-- name: ListProducts :many
SELECT id, organization_id, name, price, created_at
FROM products
WHERE organization_id = $1
ORDER BY created_at DESC;
```

Gere o código:
```bash
make sqlc
```

### 5. Registrar no Router (`internal/api/router/router.go`)

Cada handler implementa `RegisterRoutes()` para registrar suas rotas:

```go
// O handler registra suas próprias rotas
func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
    products := rg.Group("/products")
    products.POST("/", h.Create)
    products.GET("/:id", h.Get)
}

// No router.go, basta chamar RegisterRoutes no grupo correto
productSvc := service.NewProductService(db)
productHandler := handler.NewProductHandler(productSvc)
productHandler.RegisterRoutes(protected)
```

Para endpoints que exigem role específico, use `middleware.RequireRole()`:

```go
func (h *ProductHandler) RegisterRoutes(rg *gin.RouterGroup) {
    products := rg.Group("/products")
    products.POST("/", middleware.RequireRole(domain.RoleAdmin), h.Create)
    products.GET("/", h.List)
    products.DELETE("/:id", middleware.RequireRole(domain.RoleOwner), h.Delete)
}
```

---

## Padrões de Código

### Middleware

Ao adicionar novo middleware, respeite a ordem de execução:

```go
r := gin.New()
r.Use(middleware.Recovery())
r.Use(middleware.RequestID())
r.Use(middleware.StructuredLogger(logger))
r.Use(middleware.Errors())
```

Cada middleware deve:
- Ser independente e testável isoladamente
- Registrar erros via `c.Error(err)` em vez de responder diretamente
- Chamar `c.Next()` para passar para o próximo middleware

### Validação

Use `go-playground/validator` com tags estruturadas:

```go
type createRequest struct {
    Email    string `json:"email"    binding:"required,email"`
    Password string `json:"password" binding:"required,min=8,max=64"`
    Name     string `json:"name"     binding:"required,max=100"`
}
```

Mensagens são traduzidas automaticamente em `internal/api/err/validation.go`.

### Nomenclatura

- Pacotes: lowercase sem underscore (`internal/api/handler`)
- Funções e métodos: camelCase (`createProduct`, `fetchByID`)
- Constantes: UPPER_SNAKE_CASE (`DEFAULT_TTL`, `MAX_RETRY`)
- Variáveis: camelCase (`productID`, `isValid`)

### Tratamento de Erros

Sempre envolva erros com contexto:

```go
// Evite
if err != nil {
    return err
}

// Prefira
if err != nil {
    return fmt.Errorf("fetch product: %w", err)
}
```

### Injeção de Dependências

Sempre explícita no construtor:

```go
// Correto
func NewProductService(db *database.DB) *ProductService {
    return &ProductService{db: db}
}

// Evite estado global
var globalDB *database.DB
```

### Organização de Testes

Arquivos `*_test.go` colocados junto ao código:

```
internal/
├── service/
│   ├── product.go
│   └── product_test.go
├── api/handler/
│   ├── product.go
│   └── product_test.go
```

---

## Migrations

### Criar migration

```bash
make migrate-create name=add_products_table
```

### Escrever SQL

```sql
CREATE TABLE IF NOT EXISTS products (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    price           INTEGER NOT NULL CHECK (price > 0),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_products_organization_id ON products(organization_id);
```

### Aplicar e reverter

```bash
make migrate-up
make migrate-down
```

---

## Testes

### Executar testes

```bash
# Todos os testes
go test ./...

# Pacote específico
go test ./internal/service/...

# Com cobertura
go test -cover ./...

# Relatório de cobertura
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Mock de dependências

Use [testify/mock](https://pkg.go.dev/github.com/stretchr/testify/mock):

```go
import "github.com/stretchr/testify/mock"

type mockDB struct {
    mock.Mock
}

func (m *mockDB) GetProduct(ctx context.Context, id uuid.UUID) (Product, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(Product), args.Error(1)
}

func TestServiceGetProduct(t *testing.T) {
    db := &mockDB{}
    db.On("GetProduct", mock.Anything, mock.Anything).Return(Product{Name: "Widget"}, nil)

    svc := NewProductService(db)
    product, err := svc.Get(context.Background(), uuid.New())

    assert.NoError(t, err)
    assert.Equal(t, "Widget", product.Name)
}
```

---

## Qualidade de Código

### Formatação

```bash
go fmt ./...
```

Configuração recomendada para VSCode (`.vscode/settings.json`):
```json
"[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.organizeImports": "always"
    }
}
```

### Lint e vet

```bash
make lint
make vet
```

---

## Convenção de Commits

```
feat: add product CRUD endpoints

- Create product with POST /v1/products
- Get product with GET /v1/products/{id}
- Add ProductService with database integration
```

---

## Referências

- [Effective Go](https://golang.org/doc/effective_go)
- [Gin Framework](https://gin-gonic.com/)
- [SQLC Documentation](https://sqlc.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [JWT.io](https://jwt.io/)
