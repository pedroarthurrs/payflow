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
  "name": "PayFlow"
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

A conta global do usuário é buscada pelo email em `user`. Se o usuário for membro de exatamente uma organization, o token é emitido diretamente. Se for membro de múltiplas organizations, um `session_token` de 5 minutos é retornado — o cliente deve chamar `POST /v1/auth/select-organization` com esse token para obter o par completo.

O token gerado inclui `auth_type: "user"` e o `role` do membro na organization selecionada.

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

Remove um produto (soft delete). Para remoção permanente use `?hard=true`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "product not found"
}
```

---

### POST /v1/products/:product_id/plans

Cria um plano de preço vinculado a um produto. Requer role `admin` ou superior.

**Requisição:**
```json
{
  "name": "Pro Mensal",
  "amount_cents": 4900,
  "currency": "BRL",
  "interval": "month",
  "interval_count": 1,
  "active": true
}
```

`interval` deve ser um de: `day`, `week`, `month`, `year`. `slug` é opcional.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440050",
  "product_id": "550e8400-e29b-41d4-a716-446655440020",
  "name": "Pro Mensal",
  "amount_cents": 4900,
  "currency": "BRL",
  "interval": "month",
  "interval_count": 1,
  "active": true,
  "version": 1,
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

---

### GET /v1/products/:product_id/plans

Lista os planos de um produto.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Padrão: 20, máximo: 100 |
| `offset` | int | Padrão: 0 |
| `active` | bool | Filtra por `active=true` ou `active=false` |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440050",
      "product_id": "550e8400-e29b-41d4-a716-446655440020",
      "name": "Pro Mensal",
      "amount_cents": 4900,
      "currency": "BRL",
      "interval": "month",
      "interval_count": 1,
      "active": true,
      "version": 1,
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

### GET /v1/products/:product_id/plans/:plan_id

Retorna um plano pelo ID.

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

**Resposta (404 Not Found):**
```json
{ "error": "plan not found" }
```

---

### PUT /v1/products/:product_id/plans/:plan_id

Atualiza um plano. Requer role `admin` ou superior. Alterações em `amount_cents`, `interval` ou `interval_count` incrementam `version` automaticamente.

**Requisição:** mesmo formato do `POST /v1/products/:product_id/plans`.

**Resposta (200 OK):** mesmo formato do `GET /v1/products/:product_id/plans/:plan_id`.

---

### DELETE /v1/products/:product_id/plans/:plan_id

Soft delete do plano. Para remoção permanente use `?hard=true`. Requer role `admin`.

**Resposta (204 No Content):** sem body.

**Resposta (409 Conflict)** — hard delete com assinaturas ativas:
```json
{ "error": "plan is referenced by active subscriptions" }
```

---

### POST /v1/subscriptions

Cria uma nova assinatura para um cliente.

**Requisição:**
```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "trial_end": "2026-04-20T00:00:00Z",
  "metadata": { "source": "checkout" },
  "items": [
    { "plan_id": "550e8400-e29b-41d4-a716-446655440050", "quantity": 1 }
  ]
}
```

`trial_end` é opcional. Se fornecido e no futuro, o status inicial é `trialing`; caso contrário, `incomplete`.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440060",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "status": "trialing",
  "current_period_start": "2026-04-13T00:00:00Z",
  "current_period_end": "2026-05-13T00:00:00Z",
  "cancel_at_period_end": false,
  "metadata": { "source": "checkout" },
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440061",
      "plan_id": "550e8400-e29b-41d4-a716-446655440050",
      "quantity": 1,
      "created_at": "2026-04-13T10:00:00Z"
    }
  ],
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

---

### GET /v1/subscriptions

Lista assinaturas da organization.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Padrão: 20, máximo: 100 |
| `offset` | int | Padrão: 0 |
| `customer_id` | UUID | Filtra por cliente |
| `status` | string (repetível) | `?status=active&status=past_due` |
| `from` | RFC3339 | Início do intervalo de criação |
| `to` | RFC3339 | Fim do intervalo de criação |

**Resposta (200 OK):**
```json
{
  "data": [ /* array de subscriptions */ ],
  "limit": 20,
  "offset": 0,
  "total": 5
}
```

---

### GET /v1/subscriptions/:subscription_id

Retorna uma assinatura pelo ID, com seus itens.

**Resposta (200 OK):** mesmo formato da resposta do `POST /v1/subscriptions`.

**Resposta (404 Not Found):**
```json
{ "error": "subscription not found" }
```

---

### POST /v1/subscriptions/:subscription_id/cancel

Cancela uma assinatura. Requer role `member` ou superior.

**Requisição:**
```json
{
  "reason": "customer request",
  "cancel_at_period_end": false
}
```

Se `cancel_at_period_end = true`, a assinatura permanece `active` até o fim do período antes de ser cancelada.

**Resposta (200 OK):** subscription atualizada.

**Resposta (422 Unprocessable Entity)** — já cancelada:
```json
{ "error": "subscription is already canceled" }
```

---

### POST /v1/subscriptions/:subscription_id/pause

Pausa uma assinatura ativa. Requer role `member` ou superior.

**Requisição (opcional):**
```json
{ "reason": "customer request" }
```

**Resposta (200 OK):** subscription com `status: "paused"`.

**Resposta (409 Conflict)** — transição inválida pela FSM:
```json
{ "error": "invalid subscription status transition: active → paused" }
```

---

### POST /v1/subscriptions/:subscription_id/resume

Retoma uma assinatura pausada. Requer role `member` ou superior.

**Resposta (200 OK):** subscription com `status: "active"`.

---

### PUT /v1/subscriptions/:subscription_id/items

Substitui todos os itens da assinatura atomicamente. Requer role `admin` ou superior.

**Requisição:**
```json
[
  { "plan_id": "uuid", "quantity": 1 },
  { "plan_id": "uuid", "quantity": 2 }
]
```

**Resposta (200 OK):** array com os novos itens.

---

### GET /v1/subscriptions/:subscription_id/items

Lista os itens de uma assinatura.

**Resposta (200 OK):**
```json
[
  {
    "id": "uuid",
    "plan_id": "uuid",
    "quantity": 1,
    "created_at": "2026-04-13T10:00:00Z",
    "updated_at": "2026-04-13T10:00:00Z"
  }
]
```

---

### GET /v1/subscriptions/:subscription_id/history

Retorna o histórico de mudanças de status da assinatura. Requer role `admin` ou superior.

**Resposta (200 OK):**
```json
[
  {
    "id": "uuid",
    "old_status": "trialing",
    "new_status": "active",
    "reason": "",
    "created_at": "2026-04-13T10:05:00Z"
  }
]
```

---

### POST /v1/subscriptions/:subscription_id/schedules

Cria um agendamento de mudança futura. Requer role `admin` ou superior. Apenas um agendamento pendente por assinatura.

**Requisição:**
```json
{
  "scheduled_at": "2026-05-01T00:00:00Z",
  "target_status": "canceled"
}
```

Exatamente um de `target_status` ou `target_plan_id` deve ser fornecido.

**Resposta (201 Created):**
```json
{
  "id": "uuid",
  "subscription_id": "uuid",
  "scheduled_at": "2026-05-01T00:00:00Z",
  "target_status": "canceled",
  "created_at": "2026-04-13T10:00:00Z"
}
```

**Resposta (409 Conflict)** — agendamento pendente já existe:
```json
{ "error": "subscription already has a pending schedule" }
```

---

### DELETE /v1/subscriptions/:subscription_id/schedules/pending

Cancela o agendamento pendente da assinatura. Requer role `admin` ou superior.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{ "error": "no pending schedule found" }
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

### GET /v1/members/logs

Lista os logs de atividade dos membros da organization. Requer role `admin` ou superior.

Registra toda ação autenticada via JWT (`auth_type: "user"`), incluindo chamadas a endpoints protegidos.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `cursor` | string | Cursor opaco da página anterior |
| `user_id` | UUID | Filtrar por usuário específico |
| `action` | string | Filtrar por action (ex: `GET /v1/customers`) |
| `from` | string | Timestamp ISO 8601 (início do intervalo) |
| `to` | string | Timestamp ISO 8601 (fim do intervalo) |
| `status_code` | int | Filtrar por código HTTP |
| `method` | string | Filtrar por método HTTP (`GET`, `POST`, etc.) |
| `path` | string | Filtrar por prefixo de path |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440080",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_id": "550e8400-e29b-41d4-a716-446655440010",
      "action": "POST /v1/customers",
      "method": "POST",
      "path": "/v1/customers",
      "status_code": 201,
      "latency_ms": 45,
      "ip_address": "203.0.113.42",
      "user_agent": "Mozilla/5.0 ...",
      "created_at": "2026-04-15T10:00:00Z"
    }
  ],
  "next_cursor": "eyJ0IjoiMjAyNi0wNC0xNVQxMDowMDowMFoiLCJpIjoiNTUwZTg0MDAifQ",
  "limit": 20
}
```

---

### GET /v1/members/logs/:log_id

Retorna um log de atividade específico pelo ID. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item do array `data` acima.

**Resposta (404 Not Found):**
```json
{
  "error": "user log not found"
}
```

---

### GET /v1/members

Lista os membros da organization. Requer role `admin` ou superior.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440090",
      "user_id": "550e8400-e29b-41d4-a716-446655440010",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "alice@example.com",
      "name": "Alice",
      "role": "developer",
      "joined_at": "2026-03-17T10:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0
}
```

---

### POST /v1/members

Convida um usuário para a organization pelo email. Se o email não possuir uma conta, uma conta é criada automaticamente com uma senha temporária retornada na resposta. Requer role `owner`.

Não é possível convidar um segundo `owner`.

**Requisição:**
```json
{
  "email": "alice@example.com",
  "name": "Alice",
  "role": "developer"
}
```

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440090",
  "user_id": "550e8400-e29b-41d4-a716-446655440010",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "alice@example.com",
  "name": "Alice",
  "role": "developer",
  "joined_at": "2026-03-17T10:00:00Z",
  "temp_password": "x7kR2mNp"
}
```

`temp_password` só é retornado quando uma nova conta é criada. Quando o usuário já possui conta, o campo é omitido.

**Resposta (403 Forbidden):**
```json
{
  "error": "forbidden"
}
```

Retornado ao tentar convidar um segundo `owner`.

**Resposta (409 Conflict):**
```json
{
  "error": "user is already a member"
}
```

---

### GET /v1/members/:user_id

Retorna um membro pelo `user_id`. Requer role `admin` ou superior.

**Resposta (200 OK):** mesmo formato de um item da listagem de membros.

**Resposta (404 Not Found):**
```json
{
  "error": "member not found"
}
```

---

### PUT /v1/members/:user_id/role

Atualiza o role de um membro. Requer role `owner`.

**Requisição:**
```json
{
  "role": "admin"
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/members/:user_id`.

**Resposta (403 Forbidden):**
```json
{
  "error": "forbidden"
}
```

**Resposta (404 Not Found):**
```json
{
  "error": "member not found"
}
```

---

### DELETE /v1/members/:user_id

Remove um membro da organization. Requer role `owner`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{
  "error": "member not found"
}
```

---

### GET /v1/members/:user_id/permissions

Retorna as permissões efetivas de um membro, com anotação de origem (`role`, `grant` ou `revoke`). Requer role `admin` ou superior.

Permissões efetivas = padrões do role ± overrides individuais.

**Resposta (200 OK):**
```json
{
  "role": "developer",
  "permissions": [
    { "permission": "customers:read",    "source": "role" },
    { "permission": "customers:write",   "source": "role" },
    { "permission": "payments:refund",   "source": "grant" }
  ],
  "revoked": [
    { "permission": "credentials:manage", "source": "revoke" }
  ]
}
```

`source` pode ser:
- `"role"` — concedida pelo role padrão
- `"grant"` — concedida individualmente (override)
- `"revoke"` — revogada individualmente (override que remove permissão do role)

---

### PUT /v1/members/:user_id/permissions

Define overrides de permissão em lote para um membro. Requer role `owner`.

Grants e revocations são aplicados atomicamente. Qualquer permissão não mencionada preserva seu estado atual.

**Requisição:**
```json
{
  "grants": ["payments:refund"],
  "revocations": ["credentials:manage"]
}
```

**Resposta (200 OK):** mesmo formato do `GET /v1/members/:user_id/permissions`.

---

### DELETE /v1/members/:user_id/permissions/:permission

Remove um override individual de permissão, revertendo o membro ao padrão do role. Requer role `owner`.

Exemplo: `DELETE /v1/members/:user_id/permissions/credentials:manage`

**Resposta (204 No Content):** sem body.

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

### POST /v1/invoices

Cria uma nova nota fiscal para a organization autenticada. O valor total (`amount_cents`) é calculado automaticamente a partir dos itens.

Requer permissão `invoices:write`.

**Requisição:**
```json
{
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "invoice_type": "national",
  "items": [
    {
      "description": "Plano Pro - Abril 2026",
      "quantity": "1.0",
      "unit_price_cents": 4900
    }
  ]
}
```

`invoice_type` deve ser `national` ou `international`. `items` deve conter ao menos um item.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440200",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "customer_id": "550e8400-e29b-41d4-a716-446655440001",
  "invoice_type": "national",
  "operation_type": 3,
  "amount_cents": 4900,
  "currency": "BRL",
  "status": "pending",
  "items": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440201",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "invoice_id": "550e8400-e29b-41d4-a716-446655440200",
      "description": "Plano Pro - Abril 2026",
      "quantity": "1.0",
      "unit_price_cents": 4900,
      "total_price_cents": 4900,
      "created_at": "2026-04-17T10:00:00Z",
      "updated_at": "2026-04-17T10:00:00Z"
    }
  ],
  "created_at": "2026-04-17T10:00:00Z",
  "updated_at": "2026-04-17T10:00:00Z"
}
```

**Resposta (404 Not Found):**
```json
{ "error": "customer not found" }
```

**Resposta (422 Unprocessable Entity):**
```json
{ "error": "invoice must have at least one item" }
```

---

### GET /v1/invoices

Lista as notas fiscais da organization com paginação e filtros.

Requer permissão `invoices:read`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |
| `customer_id` | UUID | Filtrar por cliente |
| `subscription_id` | UUID | Filtrar por assinatura |
| `status` | string | Filtrar por status (`pending`, `sent`, `paid`, `error`, `canceled`) |
| `invoice_type` | string | Filtrar por tipo (`national`, `international`) |
| `from` | RFC3339 | Início do intervalo de criação |
| `to` | RFC3339 | Fim do intervalo de criação |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440200",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "customer_id": "550e8400-e29b-41d4-a716-446655440001",
      "invoice_type": "national",
      "operation_type": 3,
      "amount_cents": 4900,
      "currency": "BRL",
      "status": "pending",
      "created_at": "2026-04-17T10:00:00Z",
      "updated_at": "2026-04-17T10:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "total": 1
}
```

---

### GET /v1/invoices/:invoice_id

Retorna uma nota fiscal pelo ID, incluindo itens e snapshots.

Requer permissão `invoices:read`.

**Resposta (200 OK):** mesmo formato da resposta do `POST /v1/invoices`, incluindo `customer_snapshot`, `items_snapshot` e `items`.

**Resposta (404 Not Found):**
```json
{ "error": "invoice not found" }
```

---

### POST /v1/invoices/:invoice_id/cancel

Cancela uma nota fiscal. Apenas invoices em estado não-terminal (`pending`, `sent`, `error`) podem ser canceladas.

Requer permissão `invoices:write`.

**Resposta (200 OK):** invoice atualizada com `status: "canceled"`.

**Resposta (404 Not Found):**
```json
{ "error": "invoice not found" }
```

**Resposta (422 Unprocessable Entity):**
```json
{ "error": "invoice is already canceled" }
```

---

### POST /v1/invoices/:invoice_id/transition

Transição manual de status da nota fiscal via FSM.

Requer permissão `invoices:write`.

**Requisição:**
```json
{
  "status": "sent",
  "message": "Enviada ao ERP com sucesso"
}
```

`message` é opcional. Transições válidas:

| De | Para |
|----|------|
| `pending` | `sent`, `canceled` |
| `sent` | `paid`, `error`, `canceled` |
| `error` | `sent`, `canceled` |

Estados terminais (`paid`, `canceled`) não permitem transição.

**Resposta (200 OK):** invoice atualizada.

**Resposta (422 Unprocessable Entity):**
```json
{ "error": "invalid invoice transition: pending → paid" }
```

---

### GET /v1/invoices/:invoice_id/logs

Lista os logs de operações ERP de uma nota fiscal.

Requer permissão `invoices:read`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440210",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "invoice_id": "550e8400-e29b-41d4-a716-446655440200",
      "event_type": "status_transition",
      "invoice_status": "sent",
      "attempt_number": 1,
      "success": true,
      "message": "Enviada ao ERP com sucesso",
      "created_at": "2026-04-17T10:05:00Z",
      "updated_at": "2026-04-17T10:05:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "total": 1
}
```

---

### PUT /v1/invoice-settings/:type

Cria ou atualiza a configuração de emissão fiscal para um tipo de invoice (`national` ou `international`).

Requer permissão `invoices:manage`.

**Requisição:**
```json
{
  "erp_api_key": "key_abc123",
  "erp_api_secret": "secret_xyz",
  "erp_account_id": "acc_001",
  "nfse_service_code": "1.05",
  "nfse_item_list_service": "01.05",
  "tax_rate": "5.00",
  "invoice_series": "NF",
  "iso_currency": "BRL",
  "issue_on": "payment_confirmed",
  "is_active": true
}
```

Todos os campos são opcionais. `issue_on` padrão: `payment_confirmed`. `is_active` padrão: `true`.

**Resposta (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440220",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "national",
  "erp_api_key": "key_abc123",
  "erp_api_secret": "secret_xyz",
  "erp_account_id": "acc_001",
  "nfse_service_code": "1.05",
  "nfse_item_list_service": "01.05",
  "tax_rate": "5.00",
  "invoice_series": "NF",
  "iso_currency": "BRL",
  "issue_on": "payment_confirmed",
  "is_active": true,
  "created_at": "2026-04-17T10:00:00Z",
  "updated_at": "2026-04-17T10:00:00Z"
}
```

---

### GET /v1/invoice-settings

Lista todas as configurações de emissão fiscal da organization.

Requer permissão `invoices:read`.

**Resposta (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440220",
    "organization_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "national",
    "erp_api_key": "key_abc123",
    "issue_on": "payment_confirmed",
    "is_active": true,
    "created_at": "2026-04-17T10:00:00Z",
    "updated_at": "2026-04-17T10:00:00Z"
  }
]
```

---

### GET /v1/invoice-settings/:type

Retorna a configuração de emissão fiscal para um tipo específico.

Requer permissão `invoices:read`.

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

**Resposta (404 Not Found):**
```json
{ "error": "invoice setting not found" }
```

---

### DELETE /v1/invoice-settings/:type

Remove a configuração de emissão fiscal de um tipo.

Requer permissão `invoices:manage`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{ "error": "invoice setting not found" }
```

---

### GET /v1/wallets/me

Retorna a carteira da organization autenticada. Cria automaticamente se ainda não existir.

Requer permissão `wallets:read`.

**Resposta (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440100",
  "organization_id": "550e8400-e29b-41d4-a716-446655440000",
  "balance": 150000,
  "locked_balance": 50000,
  "available_balance": 100000,
  "currency": "BRL",
  "status": "active",
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

`available_balance = balance - locked_balance`. Valores em centavos.

---

### GET /v1/wallets/:wallet_id

Retorna uma carteira pelo ID, escopada à organization autenticada.

Requer permissão `wallets:read`.

**Resposta (200 OK):** mesmo formato do `GET /v1/wallets/me`.

**Resposta (404 Not Found):**
```json
{ "error": "wallet not found" }
```

---

### GET /v1/wallets/:wallet_id/entries

Lista as entradas do ledger de uma carteira com paginação e filtros.

Requer permissão `wallets:read`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |
| `category` | string | Filtrar por categoria (`payment_split`, `payout`, `refund`, `commission`, `manual_adjustment`) |
| `type` | string | Filtrar por tipo (`credit`, `debit`) |
| `created_from` | RFC3339 | Início do intervalo de criação |
| `created_to` | RFC3339 | Fim do intervalo de criação |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440101",
      "wallet_id": "550e8400-e29b-41d4-a716-446655440100",
      "type": "credit",
      "amount_cents": 9900,
      "category": "payment_split",
      "balance_after": 150000,
      "payment_id": "550e8400-e29b-41d4-a716-446655440050",
      "created_at": "2026-04-13T10:00:00Z"
    }
  ],
  "limit": 20,
  "offset": 0,
  "total": 1
}
```

---

### POST /v1/wallets/:wallet_id/payouts

Solicita um saque da carteira. O valor é reservado (`locked_balance`) imediatamente; o débito efetivo ocorre quando o status for atualizado para `paid`.

Requer permissão `wallets:payout`.

**Requisição:**
```json
{
  "amount_cents": 50000,
  "fee_cents": 500,
  "bank_info": { "bank": "341", "branch": "0001", "account": "12345-6" }
}
```

`fee_cents` é opcional (padrão: 0). `bank_info` é um objeto JSON livre.

**Resposta (201 Created):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440102",
  "wallet_id": "550e8400-e29b-41d4-a716-446655440100",
  "amount_cents": 50000,
  "fee_cents": 500,
  "status": "pending",
  "bank_info": { "bank": "341", "branch": "0001", "account": "12345-6" },
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

**Resposta (404 Not Found):**
```json
{ "error": "wallet not found" }
```

**Resposta (422 Unprocessable Entity):**
```json
{ "error": "insufficient balance" }
```

Também retorna 422 se a carteira estiver suspensa (`wallet is suspended`).

---

### GET /v1/wallets/:wallet_id/payouts

Lista os saques de uma carteira com paginação.

Requer permissão `wallets:read`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440102",
      "wallet_id": "550e8400-e29b-41d4-a716-446655440100",
      "amount_cents": 50000,
      "fee_cents": 500,
      "status": "pending",
      "bank_info": { "bank": "341", "branch": "0001", "account": "12345-6" },
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

### POST /v1/wallets/:wallet_id/adjust

Aplica um ajuste manual de crédito ou débito na carteira.

Requer permissão `wallets:adjust` (exclusiva do role `owner`).

**Requisição:**
```json
{
  "direction": "credit",
  "amount_cents": 1000,
  "reason": "Correção manual referente à fatura #X"
}
```

`direction` deve ser `credit` ou `debit`. `reason` é obrigatório.

**Resposta (200 OK):** mesmo formato do `GET /v1/wallets/me`.

**Resposta (422 Unprocessable Entity):**
```json
{ "error": "insufficient balance" }
```

Também retorna 422 para `invalid adjustment`.

---

### GET /v1/commissions

Lista as configurações de comissão da organization.

Requer permissão `commissions:read`.

**Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Itens por página (1–100, padrão: 20) |
| `offset` | int | Offset de paginação (padrão: 0) |

**Resposta (200 OK):**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440110",
      "organization_id": "550e8400-e29b-41d4-a716-446655440000",
      "target_customer_id": "550e8400-e29b-41d4-a716-446655440001",
      "type": "percentage",
      "value": 10.0,
      "is_active": true,
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

### PUT /v1/commissions/:customer_id

Cria ou atualiza a configuração de comissão para um cliente (upsert).

Requer permissão `commissions:manage`.

**Requisição:**
```json
{
  "type": "percentage",
  "value": 10.0,
  "is_active": true
}
```

`type` deve ser `percentage` (0–100%) ou `fixed` (valor em centavos). `is_active` é opcional (padrão: `true`).

**Resposta (200 OK):** mesmo formato de um item da listagem acima.

---

### DELETE /v1/commissions/:customer_id

Remove (soft delete) a configuração de comissão de um cliente.

Requer permissão `commissions:manage`.

**Resposta (204 No Content):** sem body.

**Resposta (404 Not Found):**
```json
{ "error": "commission setting not found" }
```

---

### POST /v1/webhooks

Cria um novo endpoint de webhook. Requer role `admin` ou superior.

A URL deve usar HTTPS.

**Requisição:**
```json
{
  "url": "https://meu-sistema.com/webhooks/payflow",
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
  "url": "https://meu-sistema.com/webhooks/payflow",
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
    "url": "https://meu-sistema.com/webhooks/payflow",
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
owner (4) > admin (3) > developer (2) > support (1)
```

| Endpoint | Role Mínimo |
|----------|-------------|
| **Usuários** | |
| `GET /v1/members/logs` | admin |
| `GET /v1/members/logs/:log_id` | admin |
| **Membros** | |
| `POST /v1/members` | owner |
| `PUT /v1/members/:user_id/role` | owner |
| `DELETE /v1/members/:user_id` | owner |
| `PUT /v1/members/:user_id/permissions` | owner |
| `DELETE /v1/members/:user_id/permissions/:permission` | owner |
| `GET /v1/members` | admin |
| `GET /v1/members/:user_id` | admin |
| `GET /v1/members/:user_id/permissions` | admin |
| **Credenciais** | |
| `POST /v1/credentials` | admin |
| `GET /v1/credentials` | admin |
| `GET /v1/credentials/:credential_id` | admin |
| `DELETE /v1/credentials/:credential_id` | owner |
| `POST /v1/credentials/:credential_id/rotate` | owner |
| `GET /v1/credentials/logs` | admin |
| `GET /v1/credentials/logs/:log_id` | admin |
| **Produtos e Planos** | |
| `POST /v1/products` | Qualquer role |
| `GET /v1/products` | Qualquer role |
| `GET /v1/products/:product_id` | Qualquer role |
| `PUT /v1/products/:product_id` | Qualquer role |
| `DELETE /v1/products/:product_id` | Qualquer role |
| `POST /v1/products/:product_id/plans` | admin |
| `GET /v1/products/:product_id/plans` | Qualquer role |
| `GET /v1/products/:product_id/plans/:plan_id` | Qualquer role |
| `PUT /v1/products/:product_id/plans/:plan_id` | admin |
| `DELETE /v1/products/:product_id/plans/:plan_id` | admin |
| **Assinaturas** | |
| `POST /v1/subscriptions` | Qualquer role |
| `GET /v1/subscriptions` | Qualquer role |
| `GET /v1/subscriptions/:subscription_id` | Qualquer role |
| `POST /v1/subscriptions/:subscription_id/cancel` | support |
| `POST /v1/subscriptions/:subscription_id/pause` | support |
| `POST /v1/subscriptions/:subscription_id/resume` | support |
| `PUT /v1/subscriptions/:subscription_id/items` | admin |
| `GET /v1/subscriptions/:subscription_id/items` | Qualquer role |
| `GET /v1/subscriptions/:subscription_id/history` | admin |
| `POST /v1/subscriptions/:subscription_id/schedules` | admin |
| `DELETE /v1/subscriptions/:subscription_id/schedules/pending` | admin |
| **Pagamentos** | |
| `POST /v1/payments` | Qualquer role |
| `GET /v1/payments` | Qualquer role |
| `GET /v1/payments/:payment_id` | Qualquer role |
| `GET /v1/payments/:payment_id/history` | Qualquer role |
| `POST /v1/payments/:payment_id/refund` | support |
| `POST /v1/payments/:payment_id/cancel` | support |
| **Invoices** | |
| `POST /v1/invoices` | `invoices:write` |
| `GET /v1/invoices` | `invoices:read` |
| `GET /v1/invoices/:invoice_id` | `invoices:read` |
| `POST /v1/invoices/:invoice_id/cancel` | `invoices:write` |
| `POST /v1/invoices/:invoice_id/transition` | `invoices:write` |
| `GET /v1/invoices/:invoice_id/logs` | `invoices:read` |
| **Invoice Settings** | |
| `PUT /v1/invoice-settings/:type` | `invoices:manage` |
| `GET /v1/invoice-settings` | `invoices:read` |
| `GET /v1/invoice-settings/:type` | `invoices:read` |
| `DELETE /v1/invoice-settings/:type` | `invoices:manage` |
| **Webhooks** | |
| `POST /v1/webhooks` | admin |
| `GET /v1/webhooks` | admin |
| `GET /v1/webhooks/events` | admin |
| `GET /v1/webhooks/:webhook_id` | admin |
| `PUT /v1/webhooks/:webhook_id` | admin |
| `DELETE /v1/webhooks/:webhook_id` | owner |
| `GET /v1/webhooks/logs` | admin |
| `GET /v1/webhooks/logs/:log_id` | admin |
| **Carteiras e Ledger** | |
| `GET /v1/wallets/me` | `wallets:read` |
| `GET /v1/wallets/:wallet_id` | `wallets:read` |
| `GET /v1/wallets/:wallet_id/entries` | `wallets:read` |
| `GET /v1/wallets/:wallet_id/payouts` | `wallets:read` |
| `POST /v1/wallets/:wallet_id/payouts` | `wallets:payout` |
| `POST /v1/wallets/:wallet_id/adjust` | `wallets:adjust` (owner only) |
| **Comissões** | |
| `GET /v1/commissions` | `commissions:read` |
| `PUT /v1/commissions/:customer_id` | `commissions:manage` |
| `DELETE /v1/commissions/:customer_id` | `commissions:manage` |
| Demais rotas protegidas | Qualquer role (autenticação JWT válida) |

Tokens gerados via API Key (`auth_type: "system"`) não possuem role e recebem `403 Forbidden` em endpoints que exigem role.

Além dos roles, o sistema oferece **permissões granulares** gerenciadas via `/v1/members/:id/permissions`. Cada role tem um conjunto padrão que pode ser estendido ou restringido individualmente por usuário:

| Permissão | Descrição |
|-----------|-----------|
| `users:read` | Listar e visualizar usuários |
| `users:invite` | Convidar novos membros |
| `users:manage` | Criar, atualizar e remover usuários |
| `credentials:read` | Visualizar credenciais |
| `credentials:manage` | Criar, rotacionar e remover credenciais |
| `customers:read` | Visualizar clientes |
| `customers:write` | Criar e atualizar clientes |
| `customers:delete` | Remover clientes |
| `products:read` | Visualizar produtos |
| `products:write` | Criar e atualizar produtos e planos |
| `plans:read` | Visualizar planos |
| `plans:write` | Criar e atualizar planos |
| `subscriptions:read` | Visualizar assinaturas |
| `subscriptions:write` | Criar, cancelar e alterar status de assinaturas |
| `subscriptions:items` | Substituir itens de assinatura |
| `payments:read` | Visualizar pagamentos |
| `payments:charge` | Executar cobranças |
| `payments:refund` | Executar reembolsos |
| `payments:cancel` | Cancelar pagamentos |
| `webhooks:read` | Visualizar webhooks e logs |
| `webhooks:manage` | Criar, atualizar e remover webhooks |
| `logs:read` | Visualizar logs de credenciais e usuários |
| `wallets:read` | Visualizar carteiras, entradas do ledger e saques |
| `wallets:payout` | Solicitar saques de carteira |
| `wallets:adjust` | Aplicar ajustes manuais de saldo (owner only) |
| `commissions:read` | Visualizar configurações de comissão |
| `commissions:manage` | Criar, atualizar e remover configurações de comissão |
| `invoices:read` | Visualizar notas fiscais, itens, logs e configurações |
| `invoices:write` | Criar notas fiscais, cancelar e transicionar status |
| `invoices:manage` | Criar, atualizar e remover configurações de emissão fiscal |

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
    "iss": "payflow",
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
    "iss": "payflow",
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
