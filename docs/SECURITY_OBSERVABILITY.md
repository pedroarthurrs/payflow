# Security & Observability

Documentação das capacidades de segurança e observabilidade implementadas na API i9pay.

## Visão Geral

A feature **API Security & Observability** adiciona três capacidades transversais à API:

1. **Rate Limiting** — Proteção contra abuso e força bruta com 3 tiers configuráveis
2. **Structured Logging** — Logs estruturados com `log/slog` (JSON em produção, texto em desenvolvimento)
3. **Input Validation Translation** — Mensagens de erro legíveis para validação de entrada

---

## Rate Limiting

### Propósito

Proteger a API contra:
- Ataques de força bruta em login
- Abuso de recursos por IP
- Consumo excessivo por tenant autenticado

### Tiers

#### 1. IP-Based (Rotas Públicas)

Limita requisições por endereço IP remoto em todas as rotas públicas.

**Configuração:**
```env
RATE_LIMIT_WINDOW_SECONDS=60        # Janela de tempo (padrão: 60s)
RATE_LIMIT_MAX_REQUESTS=100         # Máximo de requisições (padrão: 100)
```

**Exemplo:**
```bash
# 100 requisições por minuto por IP
curl http://localhost:8080/v1/health
```

**Resposta ao exceder limite (429):**
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

#### 2. Login-Specific (POST /v1/auth/login)

Limite mais restritivo para mitigar força bruta em credenciais.

**Configuração:**
```env
RATE_LIMIT_LOGIN_WINDOW_SECONDS=300     # Janela de tempo (padrão: 300s = 5min)
RATE_LIMIT_LOGIN_MAX_REQUESTS=5         # Máximo de requisições (padrão: 5)
```

**Exemplo:**
```bash
# Máximo 5 tentativas de login a cada 5 minutos por IP
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "wrong"}'
```

**Resposta ao exceder limite (429):**
```json
{
  "error": "too many login attempts, try again later"
}
```

#### 3. Tenant-Based (Rotas Protegidas)

Limita requisições por tenant autenticado para evitar consumo excessivo de recursos.

**Configuração:**
```env
RATE_LIMIT_TENANT_WINDOW_SECONDS=60     # Janela de tempo (padrão: 60s)
RATE_LIMIT_TENANT_MAX_REQUESTS=200      # Máximo de requisições (padrão: 200)
```

**Exemplo:**
```bash
# Máximo 200 requisições por minuto por tenant
curl http://localhost:8080/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Implementação

**Arquivo:** `internal/api/middleware/ratelimit.go`

**Componentes:**

- `RateLimiterStore` — In-memory store com `sync.Mutex` e `map[string]*visitor`
- `RateLimitByIP()` — Middleware para limitar por IP
- `RateLimitByTenant()` — Middleware para limitar por tenant

**Algoritmo:**

```
1. Extrair chave (IP ou tenant_id)
2. Verificar se visitor existe e não expirou
3. Se expirado ou não existe: criar novo com count=1
4. Se existe: incrementar count
5. Se count > maxRequests: rejeitar com 429
6. Caso contrário: permitir e retornar remaining
```

**Janelas de tempo:**

- Cada visitor tem `expiresAt` que marca o fim da janela
- Quando a janela expira, o contador é resetado
- Sem limpeza ativa (lazy cleanup ao acessar)

---

## Structured Logging

### Propósito

Registrar cada requisição HTTP com contexto estruturado para:
- Debugging e troubleshooting
- Auditoria de acesso
- Análise de performance
- Alertas em produção

### Configuração

**Arquivo:** `cmd/api/main.go`

**Função:** `SetupLogger(env string) *slog.Logger`

```go
// Produção: JSON (machine-readable)
if env == "production" {
    return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}

// Desenvolvimento: Texto (human-readable)
return slog.New(slog.NewTextHandler(os.Stdout, nil))
```

### Campos Registrados

Cada log contém:

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `request_id` | string | UUID v4 único por requisição |
| `method` | string | HTTP method (GET, POST, etc.) |
| `path` | string | URL path (/v1/me, /v1/auth/login, etc.) |
| `status` | int | HTTP status code (200, 401, 429, etc.) |
| `latency_ms` | float64 | Tempo de processamento em ms |
| `client_ip` | string | IP remoto do cliente |
| `user_id` | string | tenant_id do JWT (vazio se não autenticado) |

### Níveis de Log

| Status | Nível | Exemplo |
|--------|-------|---------|
| 2xx, 3xx | INFO | `GET /v1/me` |
| 4xx | WARN | `POST /v1/auth/login` (401) |
| 5xx | ERROR | `GET /v1/health` (500) |

### Exemplos

**Desenvolvimento (texto):**
```
time=2026-03-13T10:15:30.123Z level=INFO msg="GET /v1/me" request_id=550e8400-e29b-41d4-a716-446655440000 method=GET path=/v1/me status=200 latency_ms=5.2 client_ip=127.0.0.1 user_id=tenant-123
```

**Produção (JSON):**
```json
{
  "time": "2026-03-13T10:15:30.123Z",
  "level": "INFO",
  "msg": "GET /v1/me",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/v1/me",
  "status": 200,
  "latency_ms": 5.2,
  "client_ip": "127.0.0.1",
  "user_id": "tenant-123"
}
```

### Request ID

**Arquivo:** `internal/api/middleware/requestid.go`

Cada requisição recebe um UUID v4 único:

1. Se header `X-Request-ID` existe → reutiliza
2. Caso contrário → gera novo UUID v4
3. Armazena em contexto Gin
4. Retorna em header `X-Request-ID`

**Exemplo:**
```bash
curl http://localhost:8080/v1/health -v

# Response headers:
# X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
```

---

## Input Validation Translation

### Propósito

Converter erros brutos do `go-playground/validator` em mensagens legíveis e estruturadas.

### Antes vs Depois

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

| Tag | Mensagem | Exemplo |
|-----|----------|---------|
| `required` | `{field} is required` | "email is required" |
| `email` | `{field} must be a valid email` | "email must be a valid email" |
| `min` | `{field} must be at least {param}` | "password must be at least 8" |
| `max` | `{field} must be at most {param}` | "name must be at most 50" |
| outros | `{field} failed {tag} validation` | "age failed numeric validation" |

### Implementação

**Arquivo:** `internal/api/err/validation.go`

**Função:** `TranslateValidationErrors(err error) []ValidationFieldError`

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

### Exemplo de Uso

**Handler:**
```go
type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

func (h *Handler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        _ = c.Error(err)
        return
    }
    
    validate := validator.New()
    if err := validate.Struct(req); err != nil {
        _ = c.Error(err)
        return
    }
    
    // ... lógica de login
}
```

**Resposta (400 Bad Request):**
```json
{
  "errors": [
    {
      "field": "email",
      "rejected_value": "invalid-email",
      "message": "email must be a valid email"
    },
    {
      "field": "password",
      "rejected_value": "short",
      "message": "password must be at least 8"
    }
  ]
}
```

---

## Middleware Chain

A ordem de middleware é crítica:

```
1. Recovery()              — Captura panics
2. RequestID()             — Gera/propaga X-Request-ID
3. StructuredLogger()      — Registra requisição
4. Errors()                — Serializa erros para JSON
5. RateLimitByIP()         — Limita por IP (v1 group)
6. Auth()                  — Valida JWT (protected group)
7. RateLimitByTenant()     — Limita por tenant (protected group)
```

**Diagrama:**

```
HTTP Request
    ↓
Recovery (catch panics)
    ↓
RequestID (generate UUID)
    ↓
StructuredLogger (log start)
    ↓
Errors (catch errors)
    ↓
RateLimitByIP (check limit)
    ↓
Handler
    ↓
Auth (validate JWT)
    ↓
RateLimitByTenant (check limit)
    ↓
Handler Logic
    ↓
StructuredLogger (log end)
    ↓
HTTP Response
```

---

## Configuração Completa

**Arquivo:** `.env.example`

```env
# Rate Limiting — IP (public routes)
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_REQUESTS=100

# Rate Limiting — Login
RATE_LIMIT_LOGIN_WINDOW_SECONDS=300
RATE_LIMIT_LOGIN_MAX_REQUESTS=5

# Rate Limiting — Tenant (protected routes)
RATE_LIMIT_TENANT_WINDOW_SECONDS=60
RATE_LIMIT_TENANT_MAX_REQUESTS=200
```

---

## Testing

### Unit Tests

Todos os componentes têm testes unitários:

```bash
# Rate Limiter Store
go test ./internal/api/middleware -run TestAllow

# Rate Limiter Middleware
go test ./internal/api/middleware -run TestRateLimit

# Request ID
go test ./internal/api/middleware -run TestRequestID

# Structured Logger
go test ./internal/api/middleware -run TestStructuredLogger

# Validation Translator
go test ./internal/api/err -run TestTranslateValidationErrors

# Errors Middleware
go test ./internal/api/middleware -run TestErrors
```

**Cobertura:**

- 26 testes unitários
- 100% de cobertura em componentes críticos
- Testes de integração para middleware chain

### Executar Todos os Testes

```bash
make test
# ou
go test ./... -v
```

---

## Performance

### Rate Limiter

- **Complexidade:** O(1) por requisição
- **Memória:** O(n) onde n = número de chaves únicas (IPs ou tenants)
- **Limpeza:** Lazy (ao acessar chave expirada)

### Structured Logger

- **Overhead:** ~1-2ms por requisição
- **Formato:** JSON em produção (otimizado para parsing)
- **Buffering:** Stdout padrão (sem buffering adicional)

### Request ID

- **Geração:** UUID v4 via `github.com/google/uuid`
- **Overhead:** <0.1ms por requisição

---

## Troubleshooting

### Rate Limit Atingido

**Sintoma:** Receber 429 Too Many Requests

**Solução:**
1. Aguarde `Retry-After` segundos
2. Verifique `X-RateLimit-Reset` para timestamp exato
3. Ajuste limites em `.env` se necessário

### Logs Não Aparecem

**Sintoma:** Nenhum log em stdout

**Verificação:**
1. Confirme `GO_ENV` está definido
2. Verifique se stdout não está redirecionado
3. Teste com `curl -v` para ver headers

### Validação Não Traduzida

**Sintoma:** Erro bruto em vez de mensagem traduzida

**Verificação:**
1. Confirme que `validator.ValidationErrors` é passado para `Errors()` middleware
2. Verifique se handler está chamando `c.Error(err)`
3. Teste com payload inválido

---

## Roadmap

- [ ] Rate limiting por endpoint específico
- [ ] Whitelist de IPs para bypass de rate limit
- [ ] Métricas de Prometheus para rate limiter
- [ ] Alertas automáticos para padrões de ataque
- [ ] Correlação de logs com trace IDs distribuídos
- [ ] Exportação de logs para ELK/Datadog

---

## Referências

- [Rate Limiting Best Practices](https://cloud.google.com/architecture/rate-limiting-strategies-techniques)
- [Structured Logging with slog](https://pkg.go.dev/log/slog)
- [go-playground/validator](https://github.com/go-playground/validator)
- [Gin Middleware](https://gin-gonic.com/docs/examples/using-middleware/)

