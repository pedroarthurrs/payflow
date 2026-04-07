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
│   ├── database/
│   │   ├── repository/       — implementações de persistência
│   │   └── dbgen/            — código gerado pelo SQLC
│   ├── monitor/              — monitor de saúde de dependências
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

## Direção Arquitetural

O projeto segue uma arquitetura em camadas com princípios de Clean Architecture aplicados de forma pragmática:

- `service/` depende de contratos definidos em `domain/`
- `database/repository/` implementa esses contratos
- SQLC e banco ficam na borda de infraestrutura
- A composição das dependências ocorre no `router` (composition root)

Ao adicionar novas features, preserve essa regra: a camada de serviço não deve depender diretamente de SQLC ou `database.DB`.

---

## Adicionando Nova Feature

Exemplo: implementar CRUD de Planos (Plans).

### 1. Handler (`internal/api/handler/plan.go`)

```go
package handler

import (
    "errors"
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "i9pay/internal/api/dto"
    apierr "i9pay/internal/api/err"
    "i9pay/internal/domain"
    "i9pay/internal/service"
)

type PlanHandler struct {
    planSvc *service.PlanService
}

func NewPlanHandler(planSvc *service.PlanService) *PlanHandler {
    return &PlanHandler{planSvc: planSvc}
}

func (h *PlanHandler) RegisterRoutes(rg *gin.RouterGroup) {
    plans := rg.Group("/plans")
    plans.POST("", h.Create)
    plans.GET("", h.List)
    plans.GET("/:plan_id", h.GetByID)
    plans.PUT("/:plan_id", h.Update)
    plans.DELETE("/:plan_id", h.Delete)
}

type createPlanRequest struct {
    Name        string `json:"name"         binding:"required"`
    AmountCents int32  `json:"amount_cents" binding:"required,min=0"`
    Interval    string `json:"interval"     binding:"required,oneof=monthly yearly"`
}

func (h *PlanHandler) Create(c *gin.Context) {
    tid, ok := dto.OrganizationID(c)
    if !ok {
        return
    }

    var req createPlanRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        _ = c.Error(err)
        return
    }

    plan, err := h.planSvc.Create(c.Request.Context(), tid, domain.PlanInput{
        Name:        req.Name,
        AmountCents: req.AmountCents,
        Interval:    req.Interval,
    })
    if err != nil {
        _ = c.Error(err)
        return
    }

    c.JSON(http.StatusCreated, plan)
}

func (h *PlanHandler) GetByID(c *gin.Context) {
    tid, ok := dto.OrganizationID(c)
    if !ok {
        return
    }

    id, err := uuid.Parse(c.Param("plan_id"))
    if err != nil {
        _ = c.Error(apierr.BadRequest("invalid plan id"))
        return
    }

    plan, err := h.planSvc.GetByID(c.Request.Context(), tid, id)
    if err != nil {
        if errors.Is(err, domain.ErrNotFound) {
            _ = c.Error(apierr.NotFound("plan not found"))
            return
        }
        _ = c.Error(err)
        return
    }

    c.JSON(http.StatusOK, plan)
}
```

### 2. Service (`internal/service/plan.go`)

```go
package service

import (
    "context"

    "github.com/google/uuid"

    "i9pay/internal/domain"
)

type PlanService struct {
    plans domain.PlanRepository
}

func NewPlanService(plans domain.PlanRepository) *PlanService {
    return &PlanService{plans: plans}
}

func (s *PlanService) Create(ctx context.Context, organizationID uuid.UUID, input domain.PlanInput) (*domain.Plan, error) {
    return s.plans.Create(ctx, organizationID, input)
}

func (s *PlanService) GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*domain.Plan, error) {
    return s.plans.GetByID(ctx, organizationID, planID)
}
```

### 3. Domain (`internal/domain/plan.go`)

```go
package domain

import (
    "context"
    "time"

    "github.com/google/uuid"
)

type PlanRepository interface {
    Create(ctx context.Context, organizationID uuid.UUID, input PlanInput) (*Plan, error)
    GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*Plan, error)
    List(ctx context.Context, organizationID uuid.UUID) ([]Plan, error)
    Update(ctx context.Context, organizationID, planID uuid.UUID, input PlanInput) (*Plan, error)
    SoftDelete(ctx context.Context, organizationID, planID uuid.UUID) error
}

type Plan struct {
    ID          uuid.UUID
    ProductID   uuid.UUID
    Name        string
    AmountCents int32
    Interval    string
    Active      bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type PlanInput struct {
    ProductID   uuid.UUID
    Name        string
    AmountCents int32
    Interval    string
    Active      bool
}
```

### 4. SQL Queries (`internal/database/queries/plan.sql`)

```sql
-- name: CreatePlan :one
INSERT INTO plans (organization_id, product_id, name, amount_cents, interval, active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPlanByID :one
SELECT * FROM plans
WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
LIMIT 1;

-- name: ListPlansByOrganization :many
SELECT * FROM plans
WHERE organization_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;
```

Gere o código:
```bash
make sqlc
```

### 5. Repository (`internal/database/repository/plan.go`)

```go
package repository

import (
    "context"
    "database/sql"
    "errors"

    "github.com/google/uuid"

    plandb "i9pay/internal/database/dbgen/plan"
    "i9pay/internal/domain"
)

type PlanRepo struct {
    q plandb.Querier
}

func NewPlanRepo(q plandb.Querier) *PlanRepo {
    return &PlanRepo{q: q}
}

func (r *PlanRepo) GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*domain.Plan, error) {
    row, err := r.q.GetPlanByID(ctx, plandb.GetPlanByIDParams{
        ID:             planID,
        OrganizationID: organizationID,
    })
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, domain.ErrNotFound
        }
        return nil, domain.ErrInternal
    }
    p := toDomainPlan(row)
    return &p, nil
}

var _ domain.PlanRepository = (*PlanRepo)(nil)
```

### 6. Registrar no Router (`internal/api/router/router.go`)

```go
planRepo := repository.NewPlanRepo(db.PlanQ)
planSvc := service.NewPlanService(planRepo)
planHandler := handler.NewPlanHandler(planSvc)
planHandler.RegisterRoutes(protected)
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
func NewProductService(repo domain.ProductRepository) *ProductService {
    return &ProductService{repo: repo}
}

// Evite estado global
var globalRepo domain.ProductRepository
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

Prefira mocks simples por interface de domínio:

```go
type mockProductRepo struct {
    getFn func(ctx context.Context, id uuid.UUID) (*Product, error)
}

func (m *mockProductRepo) Get(ctx context.Context, id uuid.UUID) (*Product, error) {
    return m.getFn(ctx, id)
}

func TestServiceGetProduct(t *testing.T) {
    repo := &mockProductRepo{getFn: func(_ context.Context, _ uuid.UUID) (*Product, error) {
        return &Product{Name: "Widget"}, nil
    }}

    svc := NewProductService(repo)
    product, err := svc.Get(context.Background(), uuid.New())

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if product.Name != "Widget" {
        t.Fatalf("expected Widget, got %s", product.Name)
    }
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
    },
    "files.exclude": {
        "**/*_test.go": true
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
