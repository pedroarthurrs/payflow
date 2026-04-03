# Arquitetura

## Visão Geral

A arquitetura segue um modelo modular em camadas com forte uso de princípios de Clean Architecture (especialmente inversão de dependências e separação entre domínio/aplicação e infraestrutura).

Na prática, o projeto funciona como uma variação pragmática de Ports and Adapters:
- A camada de aplicação (`service/`) depende de contratos do domínio (`domain/`), não de SQLC ou banco diretamente.
- A infraestrutura (`database/repository/`) implementa esses contratos e encapsula persistência.
- A composição das dependências ocorre no `router` (composition root), mantendo o núcleo de negócio desacoplado.

Ou seja: não é uma implementação "by the book" de Clean Architecture em círculos formais, mas a direção arquitetural real do código é clean-oriented.

```
┌─────────────────────────────────────────────┐
│   Camada HTTP (router + middleware)         │
├─────────────────────────────────────────────┤
│   Handlers (api/handler/)                   │
├─────────────────────────────────────────────┤
│   Lógica de Negócio (service/)              │
├─────────────────────────────────────────────┤
│   Domínio (modelos + contratos + erros)     │
├─────────────────────────────────────────────┤
│   Repositórios (database/repository/)       │
├─────────────────────────────────────────────┤
│   SQLC + PostgreSQL (database/dbgen/)       │
└─────────────────────────────────────────────┘
```

## Fluxo de Dados

```
Requisição HTTP
    ↓
Router + Middleware (Auth, Errors, RateLimit, Logger)
    ↓
Handler (recebe e valida payload)
    ↓
Service (lógica de negócio)
    ↓
Domain (modelos, contratos e erros)
    ↓
Repository (implementa contratos do domínio)
    ↓
SQLC (queries type-safe)
    ↓
Resposta JSON
```

---

## Camadas

### Camada HTTP (`api/`)

**Router** (`api/router/router.go`)
- Definição de todas as rotas
- Injeção de dependências (services, handlers)
- Configuração da cadeia de middleware

**Handlers** (`api/handler/`)
- Um arquivo por recurso, todos no mesmo pacote
- Nomes qualificados: `AddressHandler`, `AuthHandler`, `CustomerHandler`, `UserHandler`
- Responsáveis por receber requisições HTTP e validar payload

**DTOs** (`api/dto/`)
- Structs compartilhadas entre handlers
- `context.go`: extração de IDs do contexto Gin (`OrganizationID`, `CustomerID`)
- `request.go`: structs de requisição compartilhadas
- `response.go`: structs de resposta compartilhadas

**Middleware** (`api/middleware/`)
- `auth.go`: valida JWT e extrai claims
- `errors.go`: transforma erros de domínio em respostas HTTP
- `ratelimit.go`: rate limiting por IP e organization
- `requestid.go`: gera e propaga UUID v4 por requisição
- `logger.go`: logging estruturado com slog

**Tratamento de Erros** (`api/err/`)
- `apierr.go`: tipos de erro customizados (`BadRequest`, `Unauthorized`, etc.)
- `validation.go`: tradução de erros de validação em mensagens legíveis
- Mapeamento para status HTTP

---

### Camada de Serviço (`service/`)

**AuthService** (`service/auth.go`)
- Orquestra o fluxo de autenticação
- Suporta dois modos: usuário (email + password) e sistema (API Key)
- Login por usuário resolve a organization automaticamente a partir do registro
- Login por API Key valida via hash SHA-256
- Coordena o token service para emissão de pares access/refresh
- Depende da interface `domain.AuthRepository` (sem acoplamento direto ao SQLC)
- Lógica de negócio isolada da camada HTTP

**CustomerService** (`service/customer.go`)
- CRUD completo de clientes
- Soft delete padrão (marca `deleted_at`)
- Hard delete para conformidade legal (LGPD/GDPR)
- Validação de unicidade (email, documento por organization)
- Criação/atualização atômica de cliente + endereço via `domain.TxManager`
- Depende de `domain.CustomerRepository` e `domain.AddressRepository`
- Isolamento multi-organization

**AddressService** (`service/address.go`)
- CRUD completo de endereços vinculados a clientes
- Validação de existência do cliente antes de operações
- Depende de `domain.CustomerRepository` e `domain.AddressRepository`
- Defaults automáticos: tipo `billing`, país `BR`, `is_main: true`
- Isolamento multi-organization

**UserService** (`service/user.go`)
- CRUD de usuários dentro de uma organization
- Soft delete (marca `deleted_at`)
- Hash de senha com bcrypt (cost 12)
- Validação de unicidade de email por organization
- Bloqueio de criação de segundo `owner` por organization
- Depende de `domain.UserRepository`
- Isolamento multi-organization

Expansões planejadas:
- `subscription.go` — lógica de assinaturas
- `invoice.go` — geração de faturas

---

### Camada de Domínio (`domain/`)

Modelos de dados puros, contratos (interfaces) e erros tipados. Sem dependências de HTTP.

- `organization.go`: entidade Organization
- `auth.go`: `AuthRepository`, `AuthUser`, `APIKey`, `AuthType`, `LoginRequest`
- `user.go`: `UserRepository`, modelo User e roles (owner/admin/member)
- `customer.go`: `CustomerRepository`, modelo Customer, inputs de criação e atualização
- `address.go`: `AddressRepository`, modelo Address, inputs de criação e atualização
- `tx.go`: contratos transacionais (`TxManager`, `TxHandle`)
- `errors.go`: erros de domínio tipados (ErrNotFound, ErrConflict, ErrForbidden, etc.)

**Roles disponíveis:**
```
owner (3) > admin (2) > member (1)
```

**Tipos de autenticação:**
- `user` — login por email + password (inclui role no token)
- `system` — login por API Key (sem role no token)

---

### Camada de Banco de Dados (`database/`)

**Conexão** (`database/connection.go`)
- Gerencia conexão com PostgreSQL
- Expõe interface para queries via SQLC

**Código gerado pelo SQLC** (`database/dbgen/`)
- `db.go`: configuração do banco
- `models.go`: structs geradas a partir do schema
- `queries.sql.go`: implementação type-safe das queries

**Repositórios** (`database/repository/`)
- Implementam os contratos da camada de domínio
- Encapsulam acesso ao SQLC e mapeamento de erros
- Incluem gerenciador transacional (`TxManager`) para casos atômicos

**SQL** (`database/queries.sql`)
- Queries neste arquivo geram código Go via `make sqlc`
- Versionadas junto com as migrations

---

### Health Monitor (`database/health.go`)

- Goroutine em background faz ping no banco a cada 15 segundos
- Cacheia o resultado em `dbStatus` com `sync.RWMutex` para leituras concorrentes seguras
- Expõe `DatabaseOK() bool`, consumido pelo `HealthHandler`
- Separa liveness (processo vivo) de readiness (banco acessível)

---

### Token Service (`auth/token/`)

- Criação de tokens (access e refresh)
- Validação e parsing de JWT
- Suporta claims estendidos (`organization_id`, `role`, `auth_type`, `token_type`)
- Dois tipos de autenticação: `user` (com role) e `system` (API Key, sem role)
- Isolado da lógica HTTP

---

### Configuração (`config/`)

- Carrega variáveis de ambiente
- Valida valores obrigatórios
- Suporte a múltiplos ambientes (development, staging, production)

---

## Padrões e Decisões

### Cadeia de Middleware

A ordem de middleware é crítica para segurança e observabilidade:

```
1. Recovery()                    — captura panics
2. RequestID()                   — gera/propaga X-Request-ID (UUID v4)
3. StructuredLogger()            — registra requisição com slog
4. Errors()                      — serializa erros para JSON
5. RateLimitByIP()               — limita por IP (grupo v1)
6. Auth()                        — valida JWT (grupo protegido)
7. RateLimitByOrganization()     — limita por organization (grupo protegido)
8. RequireRole()                 — RBAC por endpoint (onde aplicável)
```

### RBAC (Role-Based Access Control)

O middleware `RequireRole(minRole)` em `api/middleware/rbac.go` aplica controle de acesso baseado em roles:

- Verifica se o token possui claims válidos
- Tokens de sistema (`auth_type: "system"`) sem role recebem 403
- Compara o nível do role do usuário com o mínimo exigido
- Hierarquia: `owner (3) > admin (2) > member (1)`

Usado nos endpoints de gerenciamento de usuários:
- `POST /v1/users` → requer `owner`
- `GET /v1/users` → requer `admin`
- `PUT /v1/users/:user_id` → requer `owner`
- `DELETE /v1/users/:user_id` → requer `owner`

### Rate Limiting

Implementado com 3 tiers independentes:

1. Por IP — proteção geral contra abuso (100 req/min por IP)
2. Por login — mitigação de força bruta (5 tentativas/5min por IP)
3. Por organization — isolamento de recursos (200 req/min por organization)

Cada tier usa uma `RateLimiterStore` separada com `sync.Mutex` e `map[string]*visitor`. Limpeza lazy (ao acessar chave expirada).

### Logging Estruturado

Usa `log/slog` nativo do Go (1.21+):
- Produção: JSON handler (legível por máquina)
- Desenvolvimento: Text handler (legível por humano)

Cada log contém: `request_id`, `method`, `path`, `status`, `latency_ms`, `client_ip`, `user_id`.

### Tradução de Erros de Validação

Converte erros brutos do `go-playground/validator` em mensagens legíveis:
- `required` → `{field} is required`
- `email` → `{field} must be a valid email`
- Resposta estruturada com `field`, `rejected_value`, `message`

### Padrão de Handler

Todos os handlers ficam no pacote `internal/api/handler/`, com nomes qualificados para evitar conflitos. Cada handler implementa `RegisterRoutes()` para registrar suas rotas:

```go
package handler

type AuthHandler struct {
    authSvc *service.AuthService
    loginRL gin.HandlerFunc
}

func NewAuthHandler(authSvc *service.AuthService, loginRL gin.HandlerFunc) *AuthHandler {
    return &AuthHandler{authSvc: authSvc, loginRL: loginRL}
}

// RegisterRoutes registra as rotas do handler no grupo fornecido.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
    auth := rg.Group("/auth")
    auth.POST("/login", h.loginRL, h.Login)
    auth.POST("/refresh", h.Refresh)
}
```

Sem framework de injeção de dependências. Dependências são passadas explicitamente no `router.go`:

```go
tokenSvc := token.NewService(secret, ttl1, ttl2)
customerRepo := repository.NewCustomerRepo(db.Queries)
addressRepo := repository.NewAddressRepo(db.Queries)
userRepo := repository.NewUserRepo(db.Queries)
authRepo := repository.NewAuthRepo(db.Queries)
txManager := repository.NewTxManager(db)

authSvc := service.NewAuthService(tokenSvc, authRepo)
customerSvc := service.NewCustomerService(customerRepo, addressRepo, txManager)
addressSvc := service.NewAddressService(customerRepo, addressRepo)
userSvc := service.NewUserService(userRepo)

authHandler := handler.NewAuthHandler(authSvc, loginRL)

// Cada handler registra suas próprias rotas
authHandler.RegisterRoutes(v1)
customerHandler.RegisterRoutes(protected)
addressHandler.RegisterRoutes(protected)
userHandler.RegisterRoutes(protected)
```

### Tratamento de Erros

1. Services retornam erros de domínio
2. Handlers convertem quando necessário (ex.: `apierr.BadRequest`/`apierr.Unauthorized`)
3. Middleware `Errors()` mapeia erros de domínio para status HTTP padronizados

```go
// Handler
if err != nil {
    _ = c.Error(apierr.Unauthorized("invalid credentials"))
    return
}

// Middleware converte para JSON
// {"error": "invalid credentials"}
```

---

## Adicionando Nova Feature

### Estrutura de arquivos

```
internal/
├── api/handler/{feature}.go    — camada HTTP
├── service/{feature}.go        — lógica de negócio
├── domain/{feature}.go         — modelos/contratos (se necessário)
└── database/repository/{feature}_repo.go — persistência
```

### Usando queries geradas pelo SQLC

1. Escreva a query em `internal/database/queries.sql`:
```sql
-- name: GetProduct :one
SELECT id, name, price FROM products WHERE id = $1;
```

2. Gere o código:
```bash
make sqlc
```

3. Implemente o repositório em `internal/database/repository/` usando o código gerado:
```go
type ProductRepo struct {
    q dbgen.Querier
}

func (r *ProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    row, err := r.q.GetProduct(ctx, id)
    if err != nil {
        return nil, err
    }
    return &domain.Product{ID: row.ID, Name: row.Name}, nil
}
```

4. Injete o repositório no service e registre as rotas em `internal/api/router/router.go`.

---

## Segurança

- Assinatura JWT com HMAC SHA-256
- TTLs curtos para access token (15min) e longos para refresh (7 dias)
- Header `Authorization` obrigatório em rotas protegidas
- Middleware `Recovery` captura panics
- RBAC com hierarquia de roles (owner > admin > member)
- Senhas armazenadas com bcrypt (cost 12)
- API Keys validadas via hash SHA-256
- Bloqueio de criação de segundo owner por organization

---

## Conformidade Legal (LGPD/GDPR)

O `CustomerService` implementa `HardDelete()` para atender requisitos de remoção permanente de dados.

**Casos de uso:**
- LGPD Art. 18 (Direito ao Esquecimento)
- GDPR Art. 17 (Right to Erasure)
- Ordens judiciais
- Políticas de retenção de dados

**Diferença entre os métodos:**
- `Delete()`: soft delete (marca `deleted_at`, dados preservados)
- `HardDelete()`: remoção física permanente (conformidade legal)

**Recomendações:**
- Use `Delete()` para operações normais
- Use `HardDelete()` apenas para requisições legais documentadas
- Mantenha logs de auditoria das operações de hard delete

---

## Estratégia de Testes

**Testes unitários:**
- Services: mock de repositórios/interfaces de domínio
- Handlers: mock de services
- Domain: sem dependências, 100% testável

**Testes de integração (planejados):**
- Endpoints da API com banco de dados de teste
- Transações com rollback

---

## Performance

- SQLC compila queries em build-time (zero custo em runtime)
- Rate limiter com complexidade O(1) por requisição
- Logging estruturado com overhead de ~1-2ms por requisição
- Connection pooling via `database/sql` padrão
