# Arquitetura

## Visão Geral da Arquitetura

A arquitetura segue o padrão **Modular com Camadas Lógicas**, sem usar Clean Architecture, DDD ou Hexagonal. É pragmática e escalável:

```
┌─────────────────────────────────────────────┐
│   HTTP Layer (router + middleware)          │
├─────────────────────────────────────────────┤
│   HTTP Handlers (handler/ — consolidado)    │
├─────────────────────────────────────────────┤
│   Business Logic (services/)                │
├─────────────────────────────────────────────┤
│   Domain Models (domain/)                   │
├─────────────────────────────────────────────┤
│   Database Access (database/)               │
└─────────────────────────────────────────────┘
```

## Fluxo de Dados

```
HTTP Request
    ↓
Router + Middleware (Auth, Errors)
    ↓
Handler (recebe payload HTTP)
    ↓
Service (lógica de negócio)
    ↓
Domain (validações e modelos)
    ↓
Database (SQLC queries)
    ↓
Response JSON
```

## Camadas

### 1. HTTP Layer (`api/`)

**Router** (`api/router/router.go`)
- Definição de todas as rotas
- Injeção de dependências (services, handlers)
- Configuração de middleware

**Handlers** (`api/handler/`)
- Consolidados em um único pacote (um arquivo por recurso)
- Tipos qualificados: `AddressHandler`, `AuthHandler`, `CustomerHandler`, etc.
- Consolidados em um único pacote (um arquivo por recurso)
- Tipos qualificados: `AddressHandler`, `AuthHandler`, `CustomerHandler`, etc.
- Recebem requisições HTTP
- Validam payload

**DTOs** (`api/dto/`)
- Data Transfer Objects compartilhados entre handlers
- `context.go`: Extração de IDs do contexto Gin (OrganizationID, CustomerID)
- `request.go`: Structs de request compartilhadas
- `response.go`: Structs de response compartilhadas
- Retornam respostas JSON

**DTOs** (`api/dto/`)
- Data Transfer Objects compartilhados entre handlers
- `context.go`: Extração de IDs do contexto Gin (OrganizationID, CustomerID)
- `request.go`: Structs de request compartilhadas
- `response.go`: Structs de response compartilhadas

**Middleware** (`api/middleware/`)
- `auth.go`: Valida JWT e extrai claims
- `errors.go`: Transforma erros de domínio em respostas HTTP
- `ratelimit.go`: Rate limiting por IP e organization
- `requestid.go`: Gera/propaga UUID v4 por requisição
- `logger.go`: Structured logging com slog

**Error Handling** (`api/err/`)
- `apierr.go`: Tipos de erro customizados (BadRequest, Unauthorized, etc.)
- `validation.go`: Tradução de erros de validação em mensagens legíveis
- Mapeamento para status HTTP

### 2. Service Layer (`service/`)

**AuthService** (`service/auth.go`)
- Orquestra fluxo de autenticação
- Valida credentials contra database
- Coordena token service
- Lógica de negócio isolada de HTTP

**CustomerService** (`service/customer.go`)
- Gerenciamento completo de clientes (CRUD)
- Soft delete padrão (marca `deleted_at`)
- Hard delete para conformidade legal (LGPD/GDPR)
- Validação de unicidade (email, documento por organization)
- Isolamento multi-organization

Futura expansão:
- `user_service.go` - Gerenciamento de usuários
- `subscription_service.go` - Lógica de assinaturas
- `invoice_service.go` - Geração de faturas

### 3. Domain Layer (`domain/`)

**Models**
- `organization.go`: Estruturas de dados puras (Credentials, etc.)
- `errors.go`: Erros tipados de domínio

Sem lógica de HTTP, sem banco de dados. Apenas tipos e erros.

### 4. Database Layer (`database/`)

**Conexão** (`database/postgres.go`)
- Gerencia conexão com PostgreSQL
- Expõe interface para queries via SQLC

**SQLC Generated** (`database/dbgen/`)
- `db.go`: Configuração do DB
- `models.go`: Structs geradas do schema
### 5. Auth Service (`auth/token/`)
- `queries.sql.go`: Implementação type-safe das queries
**JWT Service** (`auth/token/token.go`)
**SQL** (`database/queries.sql`)
- Queries acima dessa arquivo geram código Go automaticamente
- Versionadas com migrations
- Suporta claims estendidos (user_id, role, auth_type)

### 5. Token Service (`token/`)

**JWT Service** (`token/token.go`)
- Criação de tokens (access + refresh)
- Validação e parsing de JWT
- Isolado da lógica HTTP

### 6. Config (`config/`)

**Configuration** (`config/config.go`)
- Carrega variáveis de ambiente
- Validações de valores obrigatórios
- Suporte a múltiplos ambientes (dev/staging/prod)

## Padrões e Decisões

### Middleware Chain

A ordem de middleware é crítica para segurança e observabilidade:

```
1. Recovery()              — Captura panics
2. RequestID()             — Gera/propaga X-Request-ID (UUID v4)
3. StructuredLogger()      — Registra requisição com slog
4. Errors()                — Serializa erros para JSON
5. RateLimitByIP()         — Limita por IP (v1 group)
6. Auth()                  — Valida JWT (protected group)
7. RateLimitByOrganization()     — Limita por organization (protected group)
```

Cada middleware é responsável por uma preocupação específica:
- **Recovery**: Evita crashes
- **RequestID**: Rastreabilidade
- **StructuredLogger**: Observabilidade
- **Errors**: Serialização consistente
- **RateLimit**: Proteção contra abuso
- **Auth**: Segurança

### Rate Limiting

Implementado com 3 tiers independentes:

1. **IP-Based** — Proteção geral contra abuso (100 req/min por IP)
2. **Login-Specific** — Mitigação de força bruta (5 tentativas/5min por IP)
3. **Organization-Based** — Isolamento de recursos (200 req/min por organization)

Cada tier usa uma `RateLimiterStore` separada com `sync.Mutex` e `map[string]*visitor`. Sem limpeza ativa (lazy cleanup).

### Structured Logging

Usa `log/slog` nativo do Go (1.21+):
- **Produção**: JSON handler (machine-readable)
- **Desenvolvimento**: Text handler (human-readable)

Cada log contém: `request_id`, `method`, `path`, `status`, `latency_ms`, `client_ip`, `user_id`.

### Validation Translation

Converte erros brutos do `go-playground/validator` em mensagens legíveis:
- `required` → `{field} is required`
- `email` → `{field} must be a valid email`
Todos os handlers ficam no pacote `internal/api/handler/`, com nomes qualificados para evitar conflitos:
- Uma struct `{Recurso}Handler` com dependências
- Uma função `New{Recurso}Handler()` para construção
Resposta estruturada com `field`, `rejected_value`, `message`.

### Handler Pattern
package handler
Todos os handlers ficam no pacote `internal/api/handler/`, com nomes qualificados para evitar conflitos:
type AuthHandler struct {
- Uma função `New{Recurso}Handler()` para construção
- Métodos para cada ação HTTP

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
    return &AuthHandler{authSvc: authSvc}

type AuthHandler struct {
func (h *AuthHandler) Login(c *gin.Context) { /* ... */ }
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
    return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Login(c *gin.Context) { /* ... */ }
```

authHandler := handler.NewAuthHandler(authSvc)

Sem usar framework DI. Dependências são passadas explicitamente no `router.go`:

```go
tokenSvc := token.NewService(secret, ttl1, ttl2)
authSvc := service.NewAuthService(tokenSvc)
authHandler := handler.NewAuthHandler(authSvc)
```

### Tratamento de Erros

1. Services retornam erros de domínio
2. Handlers convertem em `apierr.{Type}`
3. Middleware `Errors()` serializa para JSON

```go
// Service
pair, err := h.authSvc.Login(creds)
if err != nil {
    _ = c.Error(err.Unauthorized("invalid credentials"))
}

// Middleware converte para JSON
{
  "error": "invalid credentials"
1. Crie arquivo em `internal/api/handler/{feature}.go`:
```
   touch internal/api/handler/product.go

### Adicionar Nova Feature

1. Crie arquivo em `internal/api/handler/{feature}.go`:
   ```bash
   touch internal/api/handler/product.go
   ```

2. Implemente handlers com dependências

3. Se necessário, crie service em `internal/service/`

4. Use domain models em `internal/domain/`

├── api/handler/product.go          # HTTP API

Exemplo: Feature de Produtos

```
internal/
├── api/handler/product.go          # HTTP API
├── service/product.go              # Business logic
└── domain/product.go               # Models
```

### Usar Queries Geradas pelo SQLC

1. Escreva query em `queries.sql`:
```sql
-- name: GetProduct :one
SELECT id, name, price FROM products WHERE id = $1;
```

2. Gere código: `make sqlc`

3. Use em service:
```go
product, err := s.db.Queries.GetProduct(ctx, productID)
```

## Testing Strategy

**Unittests**
- Services: mock dependency (database)
- Handlers: mock service
- Domain: sem dependências, 100% testável

**Integration Tests**
- API endpoints com test database
- Database transactions com rollback

    h := handler.NewAuthHandler(mockAuthSvc)

```go
func TestLoginHandler(t *testing.T) {
    // Mock service
    mockAuthSvc := &mockAuthService{}
    h := handler.NewAuthHandler(mockAuthSvc)
    
    // Test
    c := &gin.Context{...}
    h.Login(c)
    
    // Assert
    assert.Equal(t, http.StatusOK, c.Writer.Status())
}
```

## Segurança

- **JWT Signing**: HMAC SHA-256
- **Token TTLs**: Access curto (15min), Refresh longo (7dias)
- **Password**: Placeholder, deve usar bcrypt em produção
- **Headers**: Authorization obrigatório em rotas protegidas
- **Middleware Recovery**: Captura panics

## Conformidade Legal (LGPD/GDPR)

### Hard Delete para Direito ao Esquecimento

O `CustomerService` implementa o método `HardDelete()` para atender requisitos legais de remoção permanente de dados:

**Casos de Uso:**
- LGPD Art. 18 (Direito ao Esquecimento)
- GDPR Art. 17 (Right to Erasure)
- Ordens judiciais
- Políticas de retenção de dados

**Implementação:**
```go
// Uso interno apenas - não exposto via HTTP
err := customerSvc.HardDelete(ctx, organizationID, customerID)
```

**Características:**
- Operação irreversível (DELETE físico no banco)
- Requer verificação de existência prévia
- Log de auditoria automático (nível WARN)
- Isolamento por organization (segurança multi-organization)

**Diferença entre Delete e HardDelete:**
- `Delete()`: Soft delete (marca `deleted_at`, dados preservados)
- `HardDelete()`: Remoção física permanente (conformidade legal)

**Recomendações:**
- Use `Delete()` para operações normais
- Use `HardDelete()` apenas para requisições legais documentadas
- Implemente processo de aprovação antes de executar
- Mantenha logs de auditoria das operações de hard delete

## Performance

- SQLC compila queries em build-time (zero runtime cost)
- Middleware Recovery evita crashes
- Short-lived access tokens reduzem exposição
- Connection pooling via database/sql padrão
