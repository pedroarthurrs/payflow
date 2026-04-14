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

## Idempotência

Para operações de escrita (`POST` e `PATCH`), envie o header `X-Idempotency-Key` com um UUID único gerado pelo cliente. Se a requisição for reenviada com a mesma chave dentro de 24 horas, a resposta original armazenada em cache será retornada sem reprocessar a operação.

```
X-Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

O header é ignorado em métodos de leitura (`GET`, `DELETE`).

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

### GET /v1/status

Retorna o estado atual do modo manutenção.

**Resposta (200 OK):**
```json
{
  "maintenance": false
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

### POST /v1/products

Cria um novo produto para a organization autenticada.

**Requisição:**
```json
{
  "name": "Plano Básico",
  "active": true
}
```

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440020",
  "name": "Plano Básico",
  "active": true,
  "created_at": "2026-04-06T10:00:00Z",
  "updated_at": "2026-04-06T10:00:00Z"
}
```

---

### GET /v1/products

Lista todos os produtos ativos da organization autenticada.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440020",
    "name": "Plano Básico",
    "active": true,
    "created_at": "2026-04-06T10:00:00Z",
    "updated_at": "2026-04-06T10:00:00Z"
  }
]
```

---

### GET /v1/products/:product_id

Retorna um produto pelo ID.

**Resposta (200 OK):** mesmo formato do item da listagem acima.

**Resposta (404 Not Found):**
```json
{
  "error": "product not found"
}
```

---

### PUT /v1/products/:product_id

Atualiza um produto. Todos os campos obrigatórios devem ser enviados.

**Requisição:**
```json
{
  "name": "Plano Básico Atualizado",
  "active": false
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/products/:product_id`.

**Resposta (404 Not Found):**
```json
{
  "error": "product not found"
}
```

---

### DELETE /v1/products/:product_id

Remove um produto (soft delete).

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "product not found"
}
```

---

### POST /v1/credentials

Cria uma nova API Key para a organization autenticada. Requer role `admin` ou superior.

A chave bruta (`raw_key`) é retornada **apenas nesta resposta** e nunca é armazenada. Guarde-a imediatamente.

**Requisição:**
```json
{
  "name": "Integração ERP",
  "environment": "production",
  "expires_at": "2027-01-01T00:00:00Z"
}
```

`environment` deve ser `test` ou `production`. `expires_at` é opcional.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440030",
  "name": "Integração ERP",
  "key_prefix": "I9LIVE_abc",
  "environment": "production",
  "expires_at": "2027-01-01T00:00:00Z",
  "created_at": "2026-04-06T10:00:00Z",
  "raw_key": "I9LIVE_abc123...",
  "warning": "Guarde esta chave agora. Ela não poderá ser recuperada novamente."
}
```

---

### GET /v1/credentials

Lista todas as credenciais ativas da organization. Requer role `admin` ou superior.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440030",
    "name": "Integração ERP",
    "key_prefix": "I9LIVE_abc",
    "environment": "production",
    "last_used_at": "2026-04-06T09:00:00Z",
    "expires_at": "2027-01-01T00:00:00Z",
    "created_at": "2026-04-06T10:00:00Z"
  }
]
```

---

### GET /v1/credentials/:credential_id

Retorna uma credencial pelo ID. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

**Resposta (404 Not Found):**
```json
{
  "error": "credential not found"
}
```

---

### DELETE /v1/credentials/:credential_id

Revoga uma credencial (soft delete). Requer role `owner`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "credential not found"
}
```

---

### POST /v1/credentials/:credential_id/rotate

Revoga a credencial atual e emite uma nova com os mesmos metadados, atomicamente. Requer role `owner`.

**Resposta (201 Created):** mesmo formato do `POST /v1/credentials`, incluindo `raw_key` e `warning`.

**Resposta (404 Not Found):**
```json
{
  "error": "credential not found"
}
```

---

### GET /v1/credentials/logs

Lista os logs de requisições autenticadas via API Key da organization. Requer role `admin` ou superior.

Suporta paginação por cursor e filtros opcionais.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `cursor` | string | Cursor opaco da página anterior |
| `from` | string | Timestamp ISO 8601 (início do intervalo) |
| `to` | string | Timestamp ISO 8601 (fim do intervalo) |
| `status_code` | int | Filtrar por código de status HTTP |
| `method` | string | Filtrar por método HTTP (GET, POST, etc.) |
| `path` | string | Filtrar por prefixo de path |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440040",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "credential_id": "550e8400-e29b-41d4-a716-446655440030",
      "method": "POST",
      "path": "/v1/customers",
      "request_body": { "name": "João" },
      "response_body": { "id": "..." },
      "status_code": 201,
      "latency_ms": 42,
      "ip_address": "203.0.113.1",
      "user_agent": "MyApp/1.0",
      "created_at": "2026-04-06T10:05:00Z"
    }
  ],
  "next_cursor": "eyJ0IjoiMjAyNi0wNC0wNlQxMDowNTowMFoiLCJpIjoiNTUwZTg0MDAifQ",
  "limit": 20
}
```

Quando não há próxima página, `next_cursor` é `null`.

---

### GET /v1/credentials/logs/:log_id

Retorna um log específico pelo ID. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item do array `data` acima.

**Resposta (404 Not Found):**
```json
{
  "error": "credential log not found"
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

### POST /v1/payments

Cria um pagamento e executa a cobrança no gateway imediatamente.

**Requisição:**
```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "payment_method": "card",
  "amount_cents": 9900,
  "currency": "BRL"
}
```

`currency` é opcional e usa `BRL` como padrão. `amount_cents` deve ser positivo.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440050",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "payment_method": "card",
  "amount_cents": 9900,
  "status": "succeeded",
  "currency": "BRL",
  "gateway_name": "stripe",
  "gateway_reference": "pi_abc123",
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

**Resposta (402 Payment Required):** cobrança rejeitada pelo gateway.

**Resposta (503 Service Unavailable):** gateway indisponível.

---

### GET /v1/payments

Lista pagamentos da organization com paginação por offset e filtros opcionais.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset para paginação (padrão: 0) |
| `status` | string | Filtrar por status (`pending`, `authorized`, `succeeded`, `failed`, `canceled`, `disputed`, `refunded`, `partially_refunded`) |
| `customer_id` | UUID | Filtrar por cliente |
| `payment_method` | string | Filtrar por método de pagamento |
| `from` | string | Timestamp ISO 8601 (início do intervalo de criação) |
| `to` | string | Timestamp ISO 8601 (fim do intervalo de criação) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440050",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "customer_id": "550e8400-e29b-41d4-a716-446655440001",
      "payment_method": "card",
      "amount_cents": 9900,
      "status": "succeeded",
      "currency": "BRL",
      "created_at": "2026-04-13T10:00:00Z",
      "updated_at": "2026-04-13T10:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "total": 1
}
```

---

### GET /v1/payments/:payment_id

Retorna um pagamento pelo ID.

**Resposta (200 OK):** mesmo formato de um item do array `data` acima.

**Resposta (404 Not Found):**
```json
{
  "error": "payment not found"
}
```

---

### GET /v1/payments/:payment_id/history

Lista o histórico de transições de status de um pagamento (FSM).

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440051",
    "payment_id": "550e8400-e29b-41d4-a716-446655440050",
    "old_status": "pending",
    "new_status": "succeeded",
    "event_source": "gateway",
    "reason": "charge completed",
    "created_at": "2026-04-13T10:00:00Z"
  }
]
```

**Resposta (404 Not Found):**
```json
{
  "error": "payment not found"
}
```

---

### POST /v1/payments/:payment_id/refund

Inicia um reembolso para o pagamento. Requer role `member` ou superior.

**Requisição (opcional):**
```json
{
  "reason": "customer request"
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/payments/:payment_id` com status atualizado.

**Resposta (404 Not Found):**
```json
{
  "error": "payment not found"
}
```

---

### POST /v1/payments/:payment_id/cancel

Cancela um pagamento pendente. Requer role `member` ou superior.

**Requisição (opcional):**
```json
{
  "reason": "duplicate order"
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/payments/:payment_id` com status atualizado.

**Resposta (404 Not Found):**
```json
{
  "error": "payment not found"
}
```

---

### POST /v1/webhooks

Cria um novo endpoint de webhook. Requer role `admin` ou superior.

A URL deve usar HTTPS.

**Requisição:**
```json
{
  "url": "https://meu-sistema.com/webhooks/i9pay",
  "description": "Notificações de pagamento",
  "enabled_events": ["invoice.paid", "invoice.payment_failed"],
  "environment": "production"
}
```

`description` e `environment` são opcionais. `environment` deve ser `test` ou `production` (padrão: `production`).

**Event types válidos:**
```
customer.created    customer.updated    customer.deleted
invoice.created     invoice.paid        invoice.payment_failed
subscription.created  subscription.updated  subscription.canceled
```

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440060",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://meu-sistema.com/webhooks/i9pay",
  "description": "Notificações de pagamento",
  "enabled_events": ["invoice.paid", "invoice.payment_failed"],
  "environment": "production",
  "is_active": true,
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

---

### GET /v1/webhooks

Lista os endpoints de webhook da organization. Requer role `admin` ou superior.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440060",
    "organization_id": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://meu-sistema.com/webhooks/i9pay",
    "description": "Notificações de pagamento",
    "enabled_events": ["invoice.paid", "invoice.payment_failed"],
    "environment": "production",
    "is_active": true,
    "created_at": "2026-04-13T10:00:00Z",
    "updated_at": "2026-04-13T10:00:00Z"
  }
]
```

---

### GET /v1/webhooks/events

Lista todos os event types suportados. Requer role `admin` ou superior.

**Resposta (200 OK):**
```json
[
  "customer.created",
  "customer.updated",
  "customer.deleted",
  "invoice.created",
  "invoice.paid",
  "invoice.payment_failed",
  "subscription.created",
  "subscription.updated",
  "subscription.canceled"
]
```

---

### GET /v1/webhooks/:webhook_id

Retorna um endpoint pelo ID. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

**Resposta (404 Not Found):**
```json
{
  "error": "webhook endpoint not found"
}
```

---

### PUT /v1/webhooks/:webhook_id

Atualiza um endpoint de webhook. Requer role `admin` ou superior. Todos os campos são opcionais.

**Requisição:**
```json
{
  "url": "https://meu-sistema.com/webhooks/v2",
  "enabled_events": ["invoice.paid"],
  "is_active": false
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/webhooks/:webhook_id`.

---

### DELETE /v1/webhooks/:webhook_id

Remove um endpoint de webhook. Requer role `owner`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "webhook endpoint not found"
}
```

---

### GET /v1/webhooks/logs

Lista os logs de entrega de webhooks da organization. Requer role `admin` ou superior.

Suporta paginação por cursor e filtros opcionais.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `cursor` | string | Cursor opaco da página anterior |
| `from` | string | Timestamp ISO 8601 (início do intervalo) |
| `to` | string | Timestamp ISO 8601 (fim do intervalo) |
| `endpoint_id` | UUID | Filtrar por endpoint |
| `event_type` | string | Filtrar por event type |
| `success` | bool | Filtrar por resultado (`true`/`false`) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440070",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "endpoint_id": "550e8400-e29b-41d4-a716-446655440060",
      "event_type": "invoice.paid",
      "payload": { "invoice_id": "..." },
      "response_body": "OK",
      "response_status": 200,
      "duration_ms": 120,
      "attempt_number": 1,
      "success": true,
      "created_at": "2026-04-13T10:05:00Z"
    }
  ],
  "next_cursor": "eyJ0IjoiMjAyNi0wNC0xM1QxMDowNTowMFoiLCJpIjoiNTUwZTg0MDAifQ",
  "limit": 20
}
```

---

### GET /v1/webhooks/logs/:log_id

Retorna um log de entrega específico pelo ID. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item do array `data` acima.

**Resposta (404 Not Found):**
```json
{
  "error": "webhook log not found"
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
| `POST /v1/credentials` | admin |
| `GET /v1/credentials` | admin |
| `GET /v1/credentials/:credential_id` | admin |
| `DELETE /v1/credentials/:credential_id` | owner |
| `POST /v1/credentials/:credential_id/rotate` | owner |
| `GET /v1/credentials/logs` | admin |
| `GET /v1/credentials/logs/:log_id` | admin |
| `POST /v1/products` | Qualquer role |
| `GET /v1/products` | Qualquer role |
| `GET /v1/products/:product_id` | Qualquer role |
| `PUT /v1/products/:product_id` | Qualquer role |
| `DELETE /v1/products/:product_id` | Qualquer role |
| `POST /v1/payments` | Qualquer role |
| `GET /v1/payments` | Qualquer role |
| `GET /v1/payments/:payment_id` | Qualquer role |
| `GET /v1/payments/:payment_id/history` | Qualquer role |
| `POST /v1/payments/:payment_id/refund` | member |
| `POST /v1/payments/:payment_id/cancel` | member |
| `POST /v1/webhooks` | admin |
| `GET /v1/webhooks` | admin |
| `GET /v1/webhooks/events` | admin |
| `GET /v1/webhooks/:webhook_id` | admin |
| `PUT /v1/webhooks/:webhook_id` | admin |
| `DELETE /v1/webhooks/:webhook_id` | owner |
| `GET /v1/webhooks/logs` | admin |
| `GET /v1/webhooks/logs/:log_id` | admin |
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

### Criar credencial
```bash
curl -X POST http://localhost:8080/v1/credentials \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "Integração ERP", "environment": "production"}'
```

### Listar logs de credencial
```bash
curl "http://localhost:8080/v1/credentials/logs?limit=20" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
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
