# Segurança e Observabilidade

Documentação das capacidades de segurança e observabilidade implementadas na API PayFlow.

## Visão Geral

Capacidades transversais aplicadas via middleware:

1. Rate Limiting — proteção contra abuso e força bruta com 3 tiers configuráveis
2. Logging Estruturado — logs com `log/slog` (JSON em produção, texto em desenvolvimento)
3. Tradução de Erros de Validação — mensagens legíveis para erros de entrada
4. RBAC — controle de acesso por hierarquia de roles + 30 permissões granulares
5. Credential Log — audit trail assíncrono para requisições via API Key
6. User Log — audit trail assíncrono para requisições via JWT de usuário
7. Idempotência — prevenção de operações duplicadas via `X-Idempotency-Key`
8. Monitoramento de Erros — integração opcional com Sentry

---

## Rate Limiting

### Propósito

Proteger a API contra:
- Ataques de força bruta em login
- Abuso de recursos por IP
- Consumo excessivo por organization autenticada

### Tiers

#### 1. Por IP (rotas públicas)

Limita requisições por endereço IP em todas as rotas públicas.

**Configuração:**
```env
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_REQUESTS=100
```

**Resposta ao exceder o limite (429):**
```json
{
  "error": "too many requests, try again later"
}
```

**Headers de resposta:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1678800000
Retry-After: 45
```

#### 2. Por login (`POST /v1/auth/login`)

Limite mais restritivo para mitigar força bruta em credenciais.

**Configuração:**
```env
RATE_LIMIT_LOGIN_WINDOW_SECONDS=300
RATE_LIMIT_LOGIN_MAX_REQUESTS=5
```

**Resposta ao exceder o limite (429):**
```json
{
  "error": "too many login attempts, try again later"
}
```

#### 3. Por organization (rotas protegidas)

Limita requisições por organization autenticada para evitar consumo excessivo de recursos.

**Configuração:**
```env
RATE_LIMIT_ORGANIZATION_WINDOW_SECONDS=60
RATE_LIMIT_ORGANIZATION_MAX_REQUESTS=200
```

### Implementação

**Arquivo:** `internal/api/middleware/ratelimit.go`

**Componentes:**
- `RateLimiterStore` — store in-memory com `sync.Mutex` e `map[string]*visitor`
- `RateLimitByIP()` — middleware para limitar por IP
- `RateLimitByOrganization()` — middleware para limitar por organization

**Algoritmo:**
```
1. Extrair chave (IP ou organization_id)
2. Verificar se o visitor existe e não expirou
3. Se expirado ou inexistente: criar novo com count=1
4. Se existe: incrementar count
5. Se count > maxRequests: rejeitar com 429
6. Caso contrário: permitir e retornar remaining
```

Cada visitor tem `expiresAt` que marca o fim da janela. Quando a janela expira, o contador é resetado. Limpeza lazy (ao acessar chave expirada).

---

## Logging Estruturado

### Propósito

Registrar cada requisição HTTP com contexto estruturado para debugging, auditoria e análise de performance.

### Configuração

**Arquivo:** `cmd/server/main.go`

```go
// Produção: JSON (legível por máquina)
if env == "production" {
    return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// Desenvolvimento: texto (legível por humano)
return slog.New(slog.NewTextHandler(os.Stdout, nil))
```

### Campos Registrados

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `request_id` | string | UUID v4 único por requisição |
| `method` | string | Método HTTP (GET, POST, etc.) |
| `path` | string | Caminho da URL |
| `status` | int | Código de status HTTP |
| `latency_ms` | float64 | Tempo de processamento em ms |
| `client_ip` | string | IP remoto do cliente |
| `user_id` | string | `organization_id` do JWT (vazio se não autenticado) |

### Níveis de Log

| Status | Nível |
|--------|-------|
| 2xx, 3xx | INFO |
| 4xx | WARN |
| 5xx | ERROR |

### Exemplos

**Desenvolvimento (texto):**
```
time=2026-04-13T10:15:30.123Z level=INFO msg="GET /v1/me" request_id=550e8400 method=GET path=/v1/me status=200 latency_ms=5.2 client_ip=127.0.0.1 user_id=org-123
```

**Produção (JSON):**
```json
{
  "time": "2026-04-13T10:15:30.123Z",
  "level": "INFO",
  "msg": "GET /v1/me",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/v1/me",
  "status": 200,
  "latency_ms": 5.2,
  "client_ip": "127.0.0.1",
  "user_id": "org-123"
}
```

### Request ID

**Arquivo:** `internal/api/middleware/requestid.go`

Cada requisição recebe um UUID v4 único:

1. Se o header `X-Request-ID` existe, reutiliza o valor
2. Caso contrário, gera um novo UUID v4
3. Armazena no contexto Gin
4. Retorna no header `X-Request-ID` da resposta

---

## Tradução de Erros de Validação

### Propósito

Converter erros brutos do `go-playground/validator` em mensagens legíveis e estruturadas.

### Antes e Depois

**Antes (erro bruto):**
```json
{
  "error": "Key: 'loginRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"
}
```

**Depois (traduzido):**
```json
{
  "errors": [
    {
      "field": "email",
      "rejected_value": "",
      "message": "email is required"
    }
  ]
}
```

### Tags Suportadas

| Tag | Mensagem |
|-----|----------|
| `required` | `{field} is required` |
| `email` | `{field} must be a valid email` |
| `min` | `{field} must be at least {param}` |
| `max` | `{field} must be at most {param}` |
| outros | `{field} failed {tag} validation` |

### Implementação

**Arquivo:** `internal/api/err/validation.go`

```go
type ValidationFieldError struct {
    Field         string `json:"field"`
    RejectedValue any    `json:"rejected_value"`
    Message       string `json:"message"`
}
```

**Fluxo:**
1. Handler recebe payload JSON
2. Valida com `validator.Struct()`
3. Se erro: passa para `TranslateValidationErrors()`
4. Middleware `Errors()` serializa para JSON com status 400

---

## Cadeia de Middleware

A ordem é crítica para segurança e observabilidade:

```
1.  Recovery()                   — captura panics
2.  monitor.Middleware()         — instrumentação Sentry por requisição
3.  RequestID()                  — gera/propaga X-Request-ID (UUID v4)
4.  StructuredLogger()           — registra requisição com slog
5.  Maintenance()                — bloqueia tráfego em modo manutenção
6.  Errors()                     — serializa erros para JSON
7.  RateLimitByIP()              — limita por IP (grupo v1)
8.  Auth()                       — valida JWT (grupo protegido)
9.  Idempotency()                — previne duplicatas via X-Idempotency-Key (POST/PATCH)
10. CredentialLog()              — persiste log assíncrono para auth_type=system
11. UserLog()                    — persiste log assíncrono para auth_type=user
12. RateLimitByOrganization()    — limita por organization (grupo protegido)
13. RequireRole() / RequirePermission() — RBAC por endpoint
```

---

## Controle de Acesso (RBAC)

### Hierarquia de Roles

```
owner (4) > admin (3) > developer (2) > support (1)
```

Tokens gerados via API Key (`auth_type: "system"`) não possuem role e recebem `403 Forbidden` em qualquer endpoint que exija role.

### Middleware `RequireRole`

**Arquivo:** `internal/api/middleware/rbac.go`

O middleware `RequireRole(minRole)` verifica:

1. Se claims JWT existem no contexto → caso contrário, 401
2. Se o token possui role (tokens de sistema não possuem) → caso contrário, 403
3. Se o nível do role é >= ao mínimo exigido → caso contrário, 403

### Permissões Granulares

**Arquivo:** `internal/api/middleware/permission.go`

Além dos roles, o sistema oferece 30 permissões granulares no formato `{resource}:{action}`. O middleware `RequirePermission(perm)` verifica via `PermissionService.HasPermission`:

1. Overrides individuais do usuário (`user_permissions`) têm prioridade
2. Fallback para as permissões padrão do role
3. Resultados cacheados em memória com TTL de 5 minutos (`CachingPermissionResolver`)

**Permissões por role (padrão):**

| Permissão | owner | admin | developer | support |
|-----------|:-----:|:-----:|:---------:|:-------:|
| `users:read` | ✓ | ✓ | — | — |
| `users:invite` | ✓ | ✓ | — | — |
| `users:manage` | ✓ | — | — | — |
| `credentials:read` | ✓ | ✓ | ✓ | — |
| `credentials:manage` | ✓ | ✓ | ✓ | — |
| `customers:read` | ✓ | ✓ | ✓ | ✓ |
| `customers:write` | ✓ | ✓ | ✓ | ✓ |
| `customers:delete` | ✓ | ✓ | — | — |
| `products:read` | ✓ | ✓ | ✓ | ✓ |
| `products:write` | ✓ | ✓ | — | — |
| `plans:read` | ✓ | ✓ | ✓ | ✓ |
| `plans:write` | ✓ | ✓ | — | — |
| `subscriptions:read` | ✓ | ✓ | ✓ | ✓ |
| `subscriptions:write` | ✓ | ✓ | — | ✓ |
| `subscriptions:items` | ✓ | ✓ | — | — |
| `payments:read` | ✓ | ✓ | ✓ | ✓ |
| `payments:charge` | ✓ | ✓ | ✓ | — |
| `payments:refund` | ✓ | ✓ | — | ✓ |
| `payments:cancel` | ✓ | ✓ | — | ✓ |
| `webhooks:read` | ✓ | ✓ | ✓ | — |
| `webhooks:manage` | ✓ | ✓ | ✓ | — |
| `logs:read` | ✓ | ✓ | ✓ | — |
| `wallets:read` | ✓ | ✓ | ✓ | — |
| `wallets:payout` | ✓ | ✓ | — | — |
| `wallets:adjust` | ✓ | — | — | — |
| `commissions:read` | ✓ | ✓ | ✓ | — |
| `commissions:manage` | ✓ | ✓ | — | — |
| `invoices:read` | ✓ | ✓ | ✓ | — |
| `invoices:write` | ✓ | ✓ | — | — |
| `invoices:manage` | ✓ | ✓ | — | — |

Overrides individuais são gerenciados via `PUT /v1/members/:user_id/permissions` e `DELETE /v1/members/:user_id/permissions/:permission`.

### Respostas de Erro

**401 Unauthorized** (sem token):
```json
{ "error": "authorization required" }
```

**403 Forbidden** (token de sistema sem role):
```json
{ "error": "user role required" }
```

**403 Forbidden** (role ou permissão insuficiente):
```json
{ "error": "insufficient permissions" }
```

---

## Credential Log (Audit Trail — API Key)

### Propósito

Registrar todas as requisições autenticadas via API Key (`auth_type: "system"`) para auditoria e rastreabilidade.

### Funcionamento

**Arquivo:** `internal/api/middleware/credential_log.go`

O middleware `CredentialLog(writer)` é aplicado no grupo protegido, após `Auth()`:

1. Verifica se `auth_type == "system"` nos claims JWT — ignora requisições de usuário
2. Captura método, path, IP, User-Agent e corpo da requisição (até 64 KB, somente JSON válido)
3. Envolve o `ResponseWriter` para capturar o corpo da resposta
4. Após `c.Next()`, coleta status code, latência e corpo da resposta
5. Persiste o log **assincronamente** via goroutine — sem impacto na latência

### Campos Registrados

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `organization_id` | UUID | Organization autenticada |
| `credential_id` | UUID | ID da credencial usada |
| `method` | string | Método HTTP |
| `path` | string | Path da requisição |
| `request_body` | JSON | Corpo da requisição (máx 64 KB) |
| `response_body` | JSON | Corpo da resposta (máx 64 KB) |
| `status_code` | int | Código de status HTTP |
| `latency_ms` | int | Latência em milissegundos |
| `ip_address` | string | IP do cliente |
| `user_agent` | string | User-Agent do cliente |

### Consulta dos Logs

Acessível via `GET /v1/credentials/logs` com paginação por cursor (keyset) e filtros: `from`, `to`, `status_code`, `method`, `path`.

---

## User Log (Audit Trail — JWT)

### Propósito

Registrar todas as requisições autenticadas via JWT de usuário (`auth_type: "user"`) para auditoria de ações de membros.

### Funcionamento

**Arquivo:** `internal/api/middleware/member_log.go`

Análogo ao `CredentialLog`, mas ativado para `auth_type == "user"`:

1. Captura `user_id` dos claims JWT
2. Registra método, path, status, latência, IP e User-Agent
3. Persiste **assincronamente** via goroutine

### Consulta dos Logs

Acessível via `GET /v1/members/logs` com paginação por cursor (keyset) e filtros: `user_id`, `action`, `from`, `to`, `status_code`, `method`, `path`.

---

## Idempotência

### Propósito

Prevenir reprocessamento de operações de escrita duplicadas (ex: retry de rede).

### Funcionamento

**Arquivo:** `internal/api/middleware/idempotency.go`

1. Intercepta apenas `POST` e `PATCH`
2. Lê o header `X-Idempotency-Key` — ignora se ausente
3. Extrai `organization_id` dos claims JWT
4. **Cache HIT**: replays a resposta armazenada imediatamente (`c.Abort()`)
5. **Cache MISS**: captura body da resposta, executa o handler e persiste com TTL de 24h

**TTL padrão:** 24 horas

**Armazenamento:** `domain.IdempotencyStorage` — interface trocável (PostgreSQL por padrão)

### Uso

```
X-Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

O header é ignorado em métodos de leitura (`GET`, `DELETE`).

---

## Monitoramento de Erros (Sentry)

### Propósito

Capturar e rastrear erros em produção com contexto de requisição.

### Configuração

```env
SENTRY_DSN=https://...@sentry.io/...   # obrigatório para ativar
GO_ENV=production                       # define o environment no Sentry
```

Se `SENTRY_DSN` estiver vazio, o Sentry é desativado sem impacto na aplicação.

### Funcionamento

**Arquivo:** `internal/monitor/monitor.go`

- `Init(cfg)`: inicializa o SDK; no-op se DSN vazio
- `Flush()`: aguarda envio de eventos pendentes antes do processo encerrar
- `Middleware()`: middleware Gin que captura panics e erros por requisição (posição 2 na cadeia)
- `CaptureException(c, err)`: reporta erro usando o hub scoped à requisição
- `LogHandler`: `slog.Handler` customizado que encaminha logs ao Sentry como breadcrumbs e erros como eventos

---

## Modo Manutenção

**Arquivo:** `internal/api/middleware/maintenance.go`

Ativado via variável de ambiente:

```env
FEATURE_MAINTENANCE=true
```

Quando ativo, bloqueia todos os endpoints exceto `/v1/live` e `/v1/ready` com resposta `503 Service Unavailable`. Pode ser recarregado sem reiniciar o processo via `FlagProvider.Reload()`.

---

## Configuração Completa

```env
# Rate Limiting — IP (rotas públicas)
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_REQUESTS=100

# Rate Limiting — Login
RATE_LIMIT_LOGIN_WINDOW_SECONDS=300
RATE_LIMIT_LOGIN_MAX_REQUESTS=5

# Rate Limiting — Organization (rotas protegidas)
RATE_LIMIT_ORGANIZATION_WINDOW_SECONDS=60
RATE_LIMIT_ORGANIZATION_MAX_REQUESTS=200

# Segurança de API Keys
API_KEY_PEPPER=your-pepper-secret-change-me

# Monitoramento (opcional)
SENTRY_DSN=https://...@sentry.io/...

# Modo manutenção (opcional)
FEATURE_MAINTENANCE=false
```

---

## Testes

```bash
# Rate Limiter
go test ./internal/api/middleware -run TestRateLimit

# Request ID
go test ./internal/api/middleware -run TestRequestID

# Logger
go test ./internal/api/middleware -run TestStructuredLogger

# Credential Log
go test ./internal/api/middleware -run TestCredentialLog

# RBAC
go test ./internal/api/middleware -run TestRequireRole

# Idempotência
go test ./internal/api/middleware -run TestIdempotency

# Tradução de validação
go test ./internal/api/err -run TestTranslateValidationErrors

# Middleware de erros
go test ./internal/api/middleware -run TestErrors

# Todos os testes
make test
```

---

## Troubleshooting

### Rate limit atingido

**Sintoma:** resposta 429 Too Many Requests

**Solução:**
1. Aguarde o número de segundos indicado em `Retry-After`
2. Verifique `X-RateLimit-Reset` para o timestamp exato de reset
3. Ajuste os limites em `.env` se necessário

### Logs não aparecem

1. Confirme que `GO_ENV` está definido
2. Verifique se stdout não está redirecionado
3. Teste com `curl -v` para ver os headers

### Erros de validação não traduzidos

1. Confirme que `validator.ValidationErrors` está sendo passado para o middleware `Errors()`
2. Verifique se o handler está chamando `c.Error(err)`

---

## Referências

- [Rate Limiting Best Practices](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [Structured Logging with slog](https://pkg.go.dev/log/slog)
- [go-playground/validator](https://github.com/go-playground/validator)
- [Gin Middleware](https://gin-gonic.com/docs/examples/using-middleware/)
