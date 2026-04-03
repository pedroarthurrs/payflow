# API

## URL Base

```
http://localhost:8080
```

## Autenticação

Rotas protegidas requerem o header `Authorization`:

```
Authorization: Bearer <access_token>
```

O `access_token` é obtido via `POST /v1/auth/login` ou `POST /v1/auth/refresh`.

## Formato de Resposta

Todas as respostas são JSON. Erros seguem o formato:

```json
{
  "error": "mensagem de erro descritiva"
}
```

---

## Rotas Públicas

### GET /

Informações básicas sobre a API.

**Resposta (200 OK):**
```json
{
  "name": "i9pay"
}
```

---

### GET /v1/live

Liveness probe — indica que o processo está rodando.

**Resposta (200 OK):**
```json
{
  "status": "alive"
}
```

---

### GET /v1/ready

Readiness probe — indica que a aplicação está pronta para receber tráfego (banco de dados acessível).

Um `HealthMonitor` em background faz ping no banco a cada 15 segundos e cacheia o resultado. A resposta reflete o último estado conhecido.

**Resposta (200 OK):**
```json
{
  "status": "ready"
}
```

**Resposta (503 Service Unavailable):**
```json
{
  "status": "unavailable"
}
```

---

### POST /v1/auth/login

Autentica e emite um par de tokens. Suporta dois modos de autenticação:

**Modo 1 — Usuário (email + password):**
```json
{
  "email": "user@example.com",
  "password": "senha123"
}
```

A organization é resolvida automaticamente a partir do registro do usuário. O token gerado inclui `auth_type: "user"` e o `role` do usuário.

**Modo 2 — API Key (sistema):**
```json
{
  "api_key": "sk_live_abc123..."
}
```

A API key é validada via hash SHA-256. O token gerado inclui `auth_type: "system"` sem role associado.

**Resposta (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2026-03-13T10:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2026-03-20T10:00:00Z"
}
```

**Resposta (400 Bad Request):**
```json
{
  "error": "organization_id, email and password or api_key required"
}
```

**Resposta (401 Unauthorized):**
```json
{
  "error": "invalid credentials"
}
```

---

### POST /v1/auth/refresh

Emite um novo par de tokens usando um refresh token válido.

**Requisição:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Resposta (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2026-03-13T10:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2026-03-20T10:00:00Z"
}
```

**Resposta (400 Bad Request):**
```json
{
  "error": "refresh_token is required"
}
```

**Resposta (401 Unauthorized):**
```json
{
  "error": "invalid or expired refresh token"
}
```

---

## Rotas Protegidas

Todas as rotas abaixo requerem `Authorization: Bearer <access_token>`.

### GET /v1/me

Retorna informações da organization autenticada extraídas do JWT.

**Resposta (200 OK):**
```json
{
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "issued_at": "2026-03-13T10:00:00Z",
  "expires_at": "2026-03-13T10:15:00Z"
}
```

**Resposta (401 Unauthorized):**
```json
{
  "error": "authorization header required"
}
```

---

### POST /v1/customers

Cria um novo cliente para a organization autenticada. Opcionalmente cria um endereço junto.

**Requisição:**
```json
{
  "name": "João Silva",
  "email": "joao@example.com",
  "phone": "+5511999999999",
  "document": "12345678900",
  "document_type": "CPF",
  "metadata": { "source": "web" },
  "address": {
    "type": "billing",
    "postal_code": "01310-100",
    "street": "Avenida Paulista",
    "number": "1000",
    "complement": "Apto 42",
    "neighborhood": "Bela Vista",
    "city": "São Paulo",
    "state": "SP",
    "country": "BR",
    "is_main": true
  }
}
```

Os campos `phone`, `metadata` e `address` são opcionais.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "João Silva",
  "email": "joao@example.com",
  "phone": "+5511999999999",
  "document": "12345678900",
  "document_type": "CPF",
  "status": "active",
  "metadata": { "source": "web" },
  "created_at": "2026-03-17T10:00:00Z",
  "updated_at": "2026-03-17T10:00:00Z",
  "address": {
    "id": "550e8400-e29b-41d4-a716-446655440002",
    "customer_id": "550e8400-e29b-41d4-a716-446655440001",
    "type": "billing",
    "postal_code": "01310-100",
    "street": "Avenida Paulista",
    "number": "1000",
    "complement": "Apto 42",
    "neighborhood": "Bela Vista",
    "city": "São Paulo",
    "state": "SP",
    "country": "BR",
    "is_main": true,
    "created_at": "2026-03-17T10:00:00Z"
  }
}
```

O campo `address` só aparece na resposta se foi enviado na requisição.

**Resposta (409 Conflict):**
```json
{
  "error": "customer with this email already exists"
}
```

---

### GET /v1/customers

Lista todos os clientes da organization autenticada.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "João Silva",
    "email": "joao@example.com",
    "document": "12345678900",
    "document_type": "CPF",
    "status": "active",
    "created_at": "2026-03-17T10:00:00Z",
    "updated_at": "2026-03-17T10:00:00Z"
  }
]
```

---

### GET /v1/customers/:customer_id

Retorna um cliente pelo ID.

**Resposta (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "name": "João Silva",
  "email": "joao@example.com",
  "phone": "+5511999999999",
  "document": "12345678900",
  "document_type": "CPF",
  "status": "active",
  "created_at": "2026-03-17T10:00:00Z",
  "updated_at": "2026-03-17T10:00:00Z"
}
```

**Resposta (404 Not Found):**
```json
{
  "error": "customer not found"
}
```

---

### PUT /v1/customers/:customer_id

Atualiza um cliente. Todos os campos obrigatórios devem ser enviados.

**Requisição:**
```json
{
  "name": "João Silva Atualizado",
  "email": "joao.novo@example.com",
  "document": "12345678900",
  "document_type": "CPF"
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/customers/:customer_id`.

**Resposta (404 Not Found):**
```json
{
  "error": "customer not found"
}
```

---

### DELETE /v1/customers/:customer_id

Remove um cliente. Por padrão realiza soft delete. Para remoção permanente (conformidade LGPD/GDPR), use o parâmetro `?hard=true`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `hard` | bool | Se `true`, remove permanentemente os dados |

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "customer not found"
}
```

---

### POST /v1/customers/:customer_id/addresses

Adiciona um endereço a um cliente.

**Requisição:**
```json
{
  "type": "shipping",
  "postal_code": "04538-133",
  "street": "Rua Funchal",
  "number": "418",
  "city": "São Paulo",
  "state": "SP",
  "country": "BR",
  "is_main": false
}
```

`type` deve ser um de: `billing`, `shipping`, `home`. Os campos `number`, `complement`, `neighborhood` e `country` são opcionais.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440003",
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "type": "shipping",
  "postal_code": "04538-133",
  "street": "Rua Funchal",
  "number": "418",
  "city": "São Paulo",
  "state": "SP",
  "country": "BR",
  "is_main": false,
  "created_at": "2026-03-17T10:05:00Z"
}
```

---

### GET /v1/customers/:customer_id/addresses

Lista todos os endereços de um cliente.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440003",
    "customer_id": "550e8400-e29b-41d4-a716-446655440001",
    "type": "billing",
    "postal_code": "01310-100",
    "street": "Avenida Paulista",
    "city": "São Paulo",
    "state": "SP",
    "country": "BR",
    "is_main": true,
    "created_at": "2026-03-17T10:00:00Z"
  }
]
```

---

### GET /v1/customers/:customer_id/addresses/:id

Retorna um endereço específico.

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

**Resposta (404 Not Found):**
```json
{
  "error": "address not found"
}
```

---

### PUT /v1/customers/:customer_id/addresses/:id

Atualiza um endereço.

**Requisição:** mesmo formato do `POST /v1/customers/:customer_id/addresses`.

**Resposta (200 OK):** mesmo formato do `GET /v1/customers/:customer_id/addresses/:id`.

---

### DELETE /v1/customers/:customer_id/addresses/:id

Remove um endereço.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "address not found"
}
```

---

### POST /v1/users

Cria um novo usuário na organization. Requer role `owner`. Apenas um `owner` é permitido por organization.

**Requisição:**
```json
{
  "name": "Alice",
  "email": "alice@example.com",
  "password": "senha123",
  "role": "member"
}
```

`role` deve ser um de: `owner`, `admin`, `member`.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440010",
  "name": "Alice",
  "email": "alice@example.com",
  "role": "member",
  "created_at": "2026-03-17T10:00:00Z"
}
```

**Resposta (403 Forbidden):**
```json
{
  "error": "forbidden"
}
```

Retornado ao tentar criar um segundo `owner` na organization.

**Resposta (409 Conflict):**
```json
{
  "error": "email already exists for this organization"
}
```

---

### GET /v1/users

Lista todos os usuários da organization. Requer role `admin` ou superior.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440010",
    "name": "Alice",
    "email": "alice@example.com",
    "role": "member",
    "created_at": "2026-03-17T10:00:00Z"
  }
]
```

---

### PUT /v1/users/:user_id

Atualiza nome e role de um usuário. Requer role `owner`.

**Requisição:**
```json
{
  "name": "Alice",
  "role": "admin"
}
```

**Resposta (200 OK):** mesmo formato do item da listagem acima.

**Resposta (404 Not Found):**
```json
{
  "error": "user not found"
}
```

---

### DELETE /v1/users/:user_id

Remove um usuário da organization (soft delete). Requer role `owner`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "user not found"
}
```

---

## Controle de Acesso (RBAC)

Endpoints protegidos podem exigir um role mínimo. A hierarquia de roles é:

```
owner (3) > admin (2) > member (1)
```

| Endpoint | Role Mínimo |
|----------|-------------|
| `POST /v1/users` | owner |
| `PUT /v1/users/:user_id` | owner |
| `DELETE /v1/users/:user_id` | owner |
| `GET /v1/users` | admin |
| Demais rotas protegidas | Qualquer role (autenticação JWT válida) |

Tokens gerados via API Key (`auth_type: "system"`) não possuem role e recebem `403 Forbidden` em endpoints que exigem role.

**Resposta (403 Forbidden):**
```json
{
  "error": "insufficient permissions"
}
```

---

## Códigos de Status HTTP

| Código | Significado | Exemplo |
|--------|-------------|---------|
| 200 | OK | Requisição bem-sucedida |
| 201 | Created | Recurso criado com sucesso |
| 204 | No Content | Operação bem-sucedida sem retorno |
| 400 | Bad Request | Validação falhou (payload inválido) |
| 401 | Unauthorized | Token ausente ou inválido |
| 403 | Forbidden | Acesso proibido |
| 404 | Not Found | Recurso não encontrado |
| 409 | Conflict | Conflito (ex: email já existe) |
| 429 | Too Many Requests | Limite de requisições excedido |
| 500 | Internal Server Error | Erro inesperado |
| 503 | Service Unavailable | Dependência indisponível (ex: banco de dados) |

---

## Tokens JWT

### Access Token

- Algoritmo: HS256
- TTL: 15 minutos
- Claims:
  ```json
  {
    "sub": "550e8400-e29b-41d4-a716-446655440000",
    "token_type": "access",
    "iss": "i9pay",
    "iat": 1615634400,
    "exp": 1615635300
  }
  ```

### Refresh Token

- Algoritmo: HS256
- TTL: 7 dias
- Claims:
  ```json
  {
    "sub": "550e8400-e29b-41d4-a716-446655440000",
    "token_type": "refresh",
    "iss": "i9pay",
    "iat": 1615634400,
    "exp": 1622016000
  }
  ```

---

## Rate Limiting

A API implementa rate limiting em 3 tiers para proteção contra abuso e força bruta.

### Tiers

**1. Por IP (rotas públicas)**
- Limite: 100 requisições por 60 segundos por IP
- Aplica-se a todas as rotas públicas

**2. Por login**
- Limite: 5 tentativas por 300 segundos (5 minutos) por IP
- Aplica-se apenas a `POST /v1/auth/login`

**3. Por organization (rotas protegidas)**
- Limite: 200 requisições por 60 segundos por organization
- Aplica-se a todas as rotas protegidas

### Headers de Resposta

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1678800000
Retry-After: 45
```

`Retry-After` é incluído apenas em respostas 429.

### Resposta ao Exceder o Limite (429)

```json
{
  "error": "too many requests, try again later"
}
```

Para login:

```json
{
  "error": "too many login attempts, try again later"
}
```

Veja [SECURITY_OBSERVABILITY.md](SECURITY_OBSERVABILITY.md#rate-limiting) para configuração via variáveis de ambiente.

---

## Erros de Validação

Quando a validação de entrada falha, a API retorna um array estruturado de erros:

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

### Mensagens por Tag de Validação

| Tag | Mensagem |
|-----|----------|
| `required` | `{field} is required` |
| `email` | `{field} must be a valid email` |
| `min` | `{field} must be at least {param}` |
| `max` | `{field} must be at most {param}` |

Veja [SECURITY_OBSERVABILITY.md](SECURITY_OBSERVABILITY.md#tradução-de-erros-de-validação) para mais detalhes.

---

## Exemplos com cURL

### Liveness probe
```bash
curl http://localhost:8080/v1/live
```

### Readiness probe
```bash
curl http://localhost:8080/v1/ready
```

### Login (usuário)
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "senha123"}'
```

### Login (API Key)
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"api_key": "sk_live_abc123..."}'
```

### Dados do usuário autenticado
```bash
curl http://localhost:8080/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

### Refresh de token
```bash
curl -X POST http://localhost:8080/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "$REFRESH_TOKEN"}'
```

### Extraindo tokens com jq
```bash
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "senha123"}')

ACCESS_TOKEN=$(echo $RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $RESPONSE | jq -r '.refresh_token')

curl http://localhost:8080/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```
