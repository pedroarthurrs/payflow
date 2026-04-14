# Arquitetura

## Visão Geral

A arquitetura segue um modelo modular em camadas com forte uso de princípios de Clean Architecture (especialmente inversão de dependências e separação entre domínio/aplicação e infraestrutura).

Na prática, o projeto funciona como uma variação pragmática de Ports and Adapters:
- A camada de aplicação (`service/`) depende de contratos do domínio (`domain/`), não de SQLC ou banco diretamente.
- A infraestrutura (`database/repository/`) implementa esses contratos e encapsula persistência.
- A composição das dependências ocorre no `router` (composition root), mantendo o núcleo de negócio desacoplado.

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
- Composition root: instancia todos os repositórios, services e handlers
- Configura a cadeia de middleware
- Registra rotas via `handler.RegisterRoutes()`

**Handlers** (`api/handler/`)
- Um arquivo por recurso, todos no mesmo pacote `handler`
- Nomes qualificados: `AddressHandler`, `AuthHandler`, `CredentialHandler`, `CustomerHandler`, `UserHandler`, `WebhookHandler`, etc.
- `CredentialHandler` agrega tanto as rotas de credenciais quanto as de credential logs (handler unificado)
- Cada handler implementa `RegisterRoutes(rg *gin.RouterGroup)`
- Responsáveis por receber requisições HTTP, validar payload e chamar services

**DTOs** (`api/dto/`)
- Structs e helpers compartilhados entre handlers
- `context.go`: extração de IDs do contexto Gin (`OrganizationID`, `CustomerID`, `ProductID`)
- `request.go`: structs de requisição compartilhadas
- `response.go`: structs de resposta compartilhadas
- `filter.go`: helpers de parsing de query parameters (ver seção Query Filters)

**Middleware** (`api/middleware/`)
- `auth.go`: valida JWT e extrai claims
- `credential_log.go`: intercepta requisições autenticadas via API Key e persiste log assíncrono
- `errors.go`: transforma erros de domínio em respostas HTTP
- `idempotency.go`: previne operações duplicadas em POST/PATCH via `X-Idempotency-Key`
- `maintenance.go`: bloqueia tráfego quando modo manutenção está ativo
- `ratelimit.go`: rate limiting por IP e organization
- `rbac.go`: controle de acesso baseado em roles
- `requestid.go`: gera e propaga UUID v4 por requisição
- `logger.go`: logging estruturado com slog

**Tratamento de Erros** (`api/err/`)
- `apierr.go`: tipos de erro customizados (`BadRequest`, `Unauthorized`, etc.)
- `validation.go`: tradução de erros de validação em mensagens legíveis
- Mapeamento para status HTTP

---

### Camada de Serviço (`service/`)

**AuthService** (`service/auth.go`)
- Suporta dois modos: usuário (email + password) e sistema (API Key)
- Login por usuário resolve a organization a partir do registro do usuário
- Login por API Key valida via `Hasher.SHA256` e busca por hash no banco
- Verificação de senha via `PasswordHasher.Compare` (bcrypt)
- Emite pares access/refresh via `TokenProvider`

**CustomerService** (`service/customer.go`)
- CRUD completo de clientes com isolamento por organization
- Criação/atualização atômica de cliente + endereço via `TxManager`
- Soft delete padrão; hard delete para conformidade LGPD/GDPR

**AddressService** (`service/address.go`)
- CRUD de endereços vinculados a clientes
- Defaults automáticos: tipo `billing`, país `BR`, `is_main: true`

**UserService** (`service/user.go`)
- CRUD de usuários com validação de unicidade de email por organization
- Hash de senha via `PasswordHasher.Hash` (bcrypt cost 12)
- Bloqueio de criação de segundo `owner` por organization

**ProductService** (`service/product.go`)
- CRUD de produtos com soft/hard delete

**CredentialService** (`service/credential.go`)
- Geração de API Keys via `KeyGenerator.GenerateAPIKey` (prefixo `I9TEST_`/`I9LIVE_`)
- Armazena apenas `Hasher.SHA256WithPepper(pepper, rawKey)` — raw key nunca persiste
- Rotação atômica via `TxManager`: revoga a anterior e emite nova chave

**CredentialLogService** (`service/credential_log.go`)
- Listagem paginada de logs com keyset pagination (cursor-based)
- Filtros: `from`, `to`, `status_code`, `method`, `path`
- Cursor opaco: `base64url(JSON{t: RFC3339Nano, i: UUID})`

**WebhookService** (`service/webhook.go`)
- CRUD de endpoints de webhook por organization
- Listagem de logs com cursor-based pagination e filtros (`from`, `to`, `endpoint_id`, `event_type`, `success`)
- Listagem de event types suportados

**WebhookDispatcher** (`service/webhook_dispatcher.go`)
- Busca endpoints ativos que correspondem ao evento e ambiente
- Lê o `webhook_secret` da organization via `OrganizationSecretReader`
- Enfileira um `WebhookJob` por endpoint via `Queue.Enqueue`
- Falhas de enqueue são logadas e não interrompem o processamento dos demais endpoints

**PaymentService** (`service/payment.go`)
- `Create()` — persiste o registro de pagamento com status `pending`
- `GetByID()`, `List()` — consultas com filtros (status, customer, método, datas) e paginação offset

**PaymentFSMService** (`service/payment_fsm.go`)
- `Transition()` — valida a transição via FSM, persiste novo status + histórico atomicamente (`SELECT FOR UPDATE`)
- `AllowedTransitions()` — retorna transições válidas a partir do status atual
- `ListHistory()` — retorna o histórico de transições de um pagamento

**GatewayService** (`service/gateway.go`)
- `Charge()` — chama o gateway, persiste a referência externa e aciona `Transition` para `authorized`/`succeeded`/`failed`
- `Refund()`, `Cancel()` — idem, com transições FSM correspondentes

---

### Camada de Domínio (`domain/`)

Modelos de dados puros, contratos (interfaces) e erros tipados. Sem dependências de HTTP ou infraestrutura.

- `organization.go`: entidade Organization
- `auth.go`: `AuthRepository`, `AuthUser`, `APIKey`, `AuthType`, `LoginRequest`
- `user.go`: `UserRepository`, modelo User e roles (owner/admin/member)
- `customer.go`: `CustomerRepository`, modelo Customer, inputs de criação e atualização
- `address.go`: `AddressRepository`, modelo Address, inputs de criação e atualização
- `product.go`: `ProductRepository`, modelo Product, ProductInput
- `credential.go`: `CredentialRepository`, `CredentialRecord`, `CreateCredentialInput`
- `credential_log.go`: `CredentialLogRepository`, `CredentialLogWriter`, `CredentialLogRecord`, `CreateCredentialLogInput`, `ListCredentialLogsInput`, `CursorToken`
- `cleanup.go`: `CleanupRepository` — contratos para purga periódica de logs e chaves de idempotência
- `idempotency.go`: `IdempotencyRecord`, `IdempotencyStorage` — chave de idempotência e interface de persistência
- `payment.go`: modelos de pagamento (`Payment`, status, tipos)
- `payment_fsm.go`: `PaymentFSMRepository`, `TransitionHistory`, `ErrInvalidTransition` — FSM de pagamentos
- `webhook.go`: `WebhookEndpointRepository`, `WebhookLogRepository`, `OrganizationSecretReader`, modelos `WebhookEndpoint` e `WebhookLog`, event types
- `tx.go`: contratos transacionais (`TxManager`, `TxHandle`)
- `errors.go`: erros de domínio tipados (ErrNotFound, ErrConflict, ErrForbidden, etc.)

**Roles disponíveis:**
```
owner (3) > admin (2) > member (1)
```

**Tipos de autenticação:**
- `user` — login por email + password (inclui role no token)
- `system` — login por API Key (sem role no token)

**Event types de webhook:**
```
customer.created    customer.updated    customer.deleted
invoice.created     invoice.paid        invoice.payment_failed
subscription.created  subscription.updated  subscription.canceled
```

---

### Camada de Criptografia (`crypto/`)

Centraliza todas as operações criptográficas do projeto. Nenhum serviço importa `crypto/*` ou `golang.org/x/crypto` diretamente.

**PasswordHasher** (`crypto/crypto.go`)
- `Hash(password)` → bcrypt hash com salt automático (cost configurável, padrão 12)
- `Compare(hash, password)` → comparação segura contra hash armazenado
- Usado em: `UserService.Create`, `AuthService.loginByUser`

**KeyGenerator** (`crypto/crypto.go`)
- `GenerateRandomBytes(n)` → n bytes aleatórios via `crypto/rand` como hex string
- `GenerateAPIKey(environment)` → `I9TEST_{hex(32)}` ou `I9LIVE_{hex(32)}`
- Usado em: `CredentialService.Create`, `CredentialService.Rotate`

**Hasher** (`crypto/crypto.go`)
- `SHA256(data)` → hex(SHA-256(data))
- `SHA256WithPepper(pepper, data)` → hex(SHA-256(pepper + data))
- Usado em: `AuthService.loginByAPIKey`, `CredentialService` (hash de API Keys)

**HMACGenerator** (`crypto/crypto.go`)
- `HMACSHA256(secret, data)` → hex(HMAC-SHA256(secret, data))
- Usado em: `WebhookWorker.sign` para assinar payloads de webhook

---

### Sistema de Webhooks

O sistema de webhooks é composto por quatro partes independentes:

**1. Handler** (`api/handler/webhook.go`)
- CRUD de endpoints: `POST /webhooks`, `GET /webhooks`, `GET /webhooks/:id`, `PUT /webhooks/:id`, `DELETE /webhooks/:id`
- Logs: `GET /webhooks/logs`, `GET /webhooks/logs/:log_id`
- Event types: `GET /webhooks/events`
- Todos os endpoints requerem `admin` ou `owner`

**2. Service + Dispatcher** (`service/webhook.go`, `service/webhook_dispatcher.go`)
- `WebhookService`: CRUD e consulta de logs
- `WebhookDispatcher`: enfileira jobs para entrega assíncrona

**3. Queue** (`queue/`)
- Interface `Queue` com `Enqueue` e `Dequeue`
- `InMemoryQueue`: implementação com canal Go com buffer (dev/test)
- Extensível: trocar por SQS, Redis, etc. sem alterar call sites

**4. Worker** (`worker/webhook.go`)
- Consome jobs da fila com N goroutines paralelas (`workerCount`)
- Entrega via HTTP POST com header `X-Webhook-Signature` (HMAC-SHA256)
- Retry com backoff exponencial: 1min → 5min → 30min → 2h (máximo 5 tentativas)
- Persiste `WebhookLog` após cada tentativa (sucesso ou falha)
- Job é auto-suficiente: carrega `EndpointURL` e `Secret` para evitar acesso ao banco durante entrega

**Fluxo de entrega:**
```
Evento de negócio
    ↓
WebhookDispatcher.Dispatch(orgID, eventType, env, payload)
    ↓
Queue.Enqueue(WebhookJob)
    ↓
WebhookWorker.consume() [goroutine]
    ↓
HTTP POST → endpoint URL
    ↓
WebhookLogRepo.Create() [persiste resultado]
    ↓ (em caso de falha)
Queue.Enqueue(job com AttemptNumber+1, ScheduledAt+backoff)
```

---

### Cleanup Worker (`worker/cleanup.go`)

Worker de background que executa tarefas periódicas de purga de dados.

- Usa `robfig/cron` para scheduling
- Depende de `domain.CleanupRepository` (interface) — desacoplado da implementação
- Iniciado em goroutine separada no `main.go`: `go cleanupWorker.Start(ctx)`

**Jobs agendados:**

| Job | Horário | Retenção |
|-----|---------|----------|
| `credential_logs` mais antigos que N dias | 02:00 diário | 90 dias |
| `webhook_logs` mais antigos que N dias | 02:00 diário | 90 dias |
| `idempotency_keys` expiradas (`expires_at <= NOW()`) | 03:00 diário | — |

---

### Idempotency (`api/middleware/idempotency.go`)

Middleware que previne reprocessamento de operações de escrita duplicadas.

**Fluxo:**
1. Intercepta apenas `POST` e `PATCH`
2. Lê o header `X-Idempotency-Key` — ignora se ausente
3. Extrai `organization_id` dos claims JWT (requer middleware `Auth` antes)
4. **Cache HIT**: replays a resposta armazenada imediatamente (`c.Abort()`)
5. **Cache MISS**: envolve o `ResponseWriter` para capturar body, executa o handler e persiste o resultado com TTL de 24h

**Armazenamento**: `domain.IdempotencyStorage` — interface trocável (Postgres por padrão, extensível para Redis)

**TTL padrão**: `DefaultIdempotencyTTL = 24h`

---

### Query Filters (`api/dto/filter.go`)

Helpers centralizados para parsing de query parameters, usados por todos os handlers que suportam paginação ou filtragem.

**Cursor-based pagination** (keyset):
```go
f, ok := dto.ParsePagination(c)
// f.Limit  int               — padrão 20, máximo 100
// f.Cursor *domain.CursorToken — nil se ausente
```

**Offset-based pagination**:
```go
f, ok := dto.ParseOffsetPagination(c)
// f.Limit  int32 — padrão 20, máximo 100
// f.Offset int32 — padrão 0
params := f.ToListParams()
```

**Date range**:
```go
f, ok := dto.ParseDateRange(c)
// f.From *time.Time — RFC 3339, nil se ausente
// f.To   *time.Time — RFC 3339, nil se ausente
// Valida que From não é posterior a To
```

**Outros helpers**:
```go
dto.ParseInt32(c, "status_code")            // *int32, nil se ausente
dto.ParseString(c, "method")               // *string, nil se ausente
dto.ParseStatus(c, "active", "inactive")   // valida contra lista de valores permitidos
```

**Cursor encoding** (`service/credential_log.go`):
- `EncodeCursor(t, id)` → `base64url(JSON{t: RFC3339Nano, i: UUID})`
- `DecodeCursor(s)` → `*domain.CursorToken`
- Cursor é opaco para o cliente; contém timestamp + UUID para keyset pagination eficiente

---

### Feature Flags (`config/`)

Sistema de feature flags plugável, sem dependência de serviço externo por padrão.

**Interface** (`config/flags.go`):
```go
type FlagProvider interface {
    IsEnabled(flag Flag) bool
    Reload() error
}
```

**Implementação padrão** (`config/flags_env.go`):
- `EnvFlagProvider` lê variáveis `FEATURE_{FLAG_NAME}` do ambiente
- Thread-safe com `sync.RWMutex`
- Valores aceitos: `true`, `1`, `yes` (case-insensitive)
- `Reload()` re-lê o ambiente sem reiniciar o processo

**Flags disponíveis:**

| Flag | Env Var | Efeito |
|------|---------|--------|
| `FlagMaintenance` | `FEATURE_MAINTENANCE=true` | Bloqueia todos os endpoints exceto `/live` e `/ready` |

**Extensibilidade**: para usar AWS AppConfig, banco de dados ou outro provider remoto, basta implementar `FlagProvider` e injetar no router — nenhum outro código muda.

---

### Monitoramento (`monitor/`)

Integração opcional com Sentry para rastreamento de erros em produção.

- `Init(cfg)`: inicializa o SDK do Sentry; no-op se `SENTRY_DSN` estiver vazio
- `Flush()`: aguarda envio de eventos pendentes antes do processo encerrar
- `Middleware()`: middleware Gin que captura panics e erros por requisição
- `CaptureException(c, err)`: reporta erro usando o hub scoped à requisição
- `LogHandler`: `slog.Handler` customizado que encaminha logs ao Sentry como breadcrumbs e erros como eventos

Configuração:
```bash
SENTRY_DSN=https://...@sentry.io/...   # obrigatório para ativar
GO_ENV=production                       # define o environment no Sentry
```

---

### Camada de Banco de Dados (`database/`)

**Conexão** (`database/connection.go`)
- Gerencia conexão com PostgreSQL via `database/sql`
- Expõe `*sql.DB` e instâncias de `Queries` por domínio

**Código gerado pelo SQLC** (`database/dbgen/`)

Cada domínio possui seu próprio subpacote gerado:

| Subpacote | Domínio |
|-----------|---------|
| `dbgen/address` | Endereços |
| `dbgen/auth` | Autenticação |
| `dbgen/credential` | Credenciais (API Keys) |
| `dbgen/credentiallog` | Logs de credenciais |
| `dbgen/customer` | Clientes |
| `dbgen/idempotency` | Chaves de idempotência |
| `dbgen/organization` | Organizations |
| `dbgen/payment` | Pagamentos e FSM de transações |
| `dbgen/product` | Produtos |
| `dbgen/user` | Usuários |
| `dbgen/webhook` | Endpoints de webhook |
| `dbgen/webhooklog` | Logs de webhook |

**Repositórios** (`database/repository/`)
- Implementam os contratos da camada de domínio
- Encapsulam acesso ao SQLC e mapeamento de erros de domínio
- `tx.go`: `TxManager` e `txHandle` para operações atômicas com `defer sqlTx.Rollback()`

**SQL** (`database/queries/`)
- Um arquivo `.sql` por domínio
- Queries geram código Go via `make sqlc`

---

### Health Monitor (`database/health.go`)

- Goroutine em background faz ping no banco a cada 15 segundos
- Cacheia resultado em `dbStatus` com `sync.RWMutex`
- Expõe `DatabaseOK() bool`, consumido pelo `HealthHandler`
- Separa liveness (processo vivo) de readiness (banco acessível)

---

### Token Service (`auth/token/`)

- Criação e validação de tokens JWT (access e refresh)
- Claims estendidos: `organization_id`, `user_id`, `role`, `auth_type`, `token_type`
- Access token TTL: 15 minutos; Refresh token TTL: 7 dias

---

### Configuração (`config/`)

- Carrega variáveis de ambiente via `godotenv`
- Valida valores obrigatórios na inicialização
- Expõe `Config` struct com todos os parâmetros da aplicação

---

## Padrões e Decisões

### Cadeia de Middleware

A ordem é crítica para segurança e observabilidade:

```
1. Recovery()                    — captura panics
2. monitor.Middleware()          — instrumentação Sentry por requisição
3. RequestID()                   — gera/propaga X-Request-ID (UUID v4)
4. StructuredLogger()            — registra requisição com slog
5. Maintenance()                 — bloqueia tráfego em modo manutenção
6. Errors()                      — serializa erros para JSON
7. RateLimitByIP()               — limita por IP (grupo v1)
8. Auth()                        — valida JWT (grupo protegido)
9. Idempotency()                 — previne duplicatas via X-Idempotency-Key (POST/PATCH)
10. CredentialLog()              — persiste log assíncrono para auth_type=system
11. RateLimitByOrganization()    — limita por organization (grupo protegido)
12. RequireRole()                — RBAC por endpoint (onde aplicável)
```

### RBAC

`RequireRole(minRole)` em `api/middleware/rbac.go`:
- Tokens de sistema (`auth_type: system`) sem role recebem 403
- Hierarquia: `owner (3) > admin (2) > member (1)`

### Rate Limiting

3 tiers independentes com `RateLimiterStore` separadas:

| Tier | Chave | Padrão |
|------|-------|--------|
| IP | IP do cliente | 100 req/min |
| Login | IP do cliente | 5 tentativas/5min |
| Organization | organization_id | 200 req/min |

### Padrão de Handler

```go
type FeatureHandler struct {
    svc *service.FeatureService
}

func NewFeatureHandler(svc *service.FeatureService) *FeatureHandler {
    return &FeatureHandler{svc: svc}
}

func (h *FeatureHandler) RegisterRoutes(rg *gin.RouterGroup) {
    g := rg.Group("/features")
    g.POST("", h.Create)
    g.GET("", h.List)
    g.GET("/:id", h.GetByID)
}
```

### Tratamento de Erros

1. Services retornam erros de domínio (`domain.ErrNotFound`, `domain.ErrConflict`, etc.)
2. Handlers convertem quando necessário (`apierr.BadRequest`, `apierr.Unauthorized`)
3. Middleware `Errors()` mapeia para status HTTP e serializa JSON

```go
// Handler
if err != nil {
    _ = c.Error(apierr.NotFound("customer not found"))
    return
}
// Middleware → {"error": "customer not found"} com status 404
```

### Transações

`TxManager.WithTx` garante atomicidade com rollback automático via `defer`:

```go
if err := s.txManager.WithTx(ctx, func(tx domain.TxHandle) error {
    if _, err := tx.CustomerRepo().Create(ctx, orgID, input); err != nil {
        return err // rollback automático
    }
    if _, err := tx.AddressRepo().Create(ctx, orgID, customerID, addr); err != nil {
        return err // rollback automático
    }
    return nil // commit
}); err != nil {
    return nil, nil, err
}
```

Operações atômicas implementadas:
- `CustomerService.Create` com endereço
- `CustomerService.Update` com endereço
- `CredentialService.Rotate` (revoga + cria)
- `PaymentFSMService` (criação de pagamento + registro de histórico FSM)

---

## Segurança

- JWT com HMAC SHA-256, TTL curto (15min access, 7d refresh)
- Senhas com bcrypt cost 12 (salt automático)
- API Keys: apenas hash SHA-256 com pepper armazenado; raw key nunca persiste
- Webhook payloads assinados com HMAC-SHA256 (`X-Webhook-Signature`)
- Criptografia centralizada em `internal/crypto/` — nenhum serviço usa `crypto/*` diretamente
- RBAC com hierarquia de roles
- Rate limiting em 3 tiers
- Audit trail assíncrono para requisições via API Key
- Bloqueio de segundo owner por organization

---

## Conformidade Legal (LGPD/GDPR)

`CustomerService.HardDelete()` remove fisicamente o registro para atender:
- LGPD Art. 18 (Direito ao Esquecimento)
- GDPR Art. 17 (Right to Erasure)

Use `Delete()` (soft) para operações normais; `HardDelete()` apenas para requisições legais documentadas.

---

## Estratégia de Testes

- **Handlers**: mock de services, testa comportamento HTTP
- **Services**: mock de repositórios, testa lógica de negócio
- **Middleware**: testa rate limiting, RBAC, logging, erros
- **Crypto**: testes de hash, geração de chaves, HMAC, comparação segura + benchmarks
- **Domain**: sem dependências, 100% testável em isolamento

---

## Adicionando Nova Feature

```
internal/
├── api/handler/{feature}.go         — camada HTTP (RegisterRoutes)
├── service/{feature}.go             — lógica de negócio
├── domain/{feature}.go              — modelos e contratos
└── database/repository/{feature}.go — persistência (implementa contratos)
```

1. Escreva queries em `internal/database/queries/{feature}.sql`
2. Gere código: `make sqlc`
3. Implemente repositório usando o subpacote `dbgen/{feature}`
4. Implemente service dependendo de interfaces de domínio
5. Implemente handler com `RegisterRoutes`
6. Registre no `router.go` (composition root)

---

## Frontend (Vue 3 + TypeScript)

O frontend em `web/` é servido pelo Vite em desenvolvimento e consome a API Go diretamente.

### Stack

- **Framework**: Vue 3 com Composition API
- **Roteamento**: Vue Router 4 com navigation guards de autenticação e RBAC
- **Estado global**: Pinia (`stores/auth.ts`, `stores/status.ts`)
- **HTTP**: Axios com interceptores de token e refresh automático
- **UI**: Tailwind CSS 4 + shadcn/vue (radix-vue, reka-ui) + lucide-vue-next
- **Testes**: Vitest + @vue/test-utils + fast-check (property-based)
- **Gerenciador de pacotes**: pnpm

### Estrutura (`web/src/`)

```
web/src/
├── router/         — rotas com guards de autenticação (isAuthenticated, hasRole)
├── views/          — páginas por recurso (customers/, users/, addresses/)
├── components/     — componentes reutilizáveis (DataTable, FormField, etc.)
│   └── ui/         — biblioteca de UI (button, input, dialog, table, etc.)
├── composables/    — useAuth, useCrud, useToast, useErrorHandler
├── stores/         — Pinia: auth (tokens, role, login/logout) e status (maintenance)
├── services/
│   └── http.ts     — Axios instance: base URL /api, interceptor de Bearer token, refresh on 401
├── types/          — DTOs TypeScript espelhando as respostas da API Go
└── lib/utils.ts    — helpers utilitários
```

### Comunicação com a API

O Vite proxia `/api` para `VITE_API_PROXY_TARGET` (padrão: `http://api:8080`) em desenvolvimento. O browser usa `VITE_API_BASE_URL` (ou `http://<host>:8080` como fallback) em produção.

### Rotas do Frontend

| Rota | Acesso | Descrição |
|------|--------|-----------|
| `/login` | público | Login com email/senha |
| `/dashboard` | autenticado | Painel principal |
| `/customers` | autenticado | Lista de clientes |
| `/customers/new` | autenticado | Criação de cliente |
| `/customers/:id` | autenticado | Detalhes do cliente |
| `/customers/:id/edit` | autenticado | Edição de cliente |
| `/customers/:id/addresses/new` | autenticado | Novo endereço |
| `/customers/:id/addresses/:addressId/edit` | autenticado | Edição de endereço |
| `/users` | admin+ | Lista de usuários |
| `/users/new` | owner | Criação de usuário |
| `/users/:id/edit` | owner | Edição de usuário |
| `/maintenance` | público | Modo manutenção |
| `/403` | público | Acesso negado |

