# API Documentation

## Base URL

```
http://localhost:8080
```

## Authentication

Rotas protegidas requerem um header `Authorization`:

```
Authorization: Bearer <access_token>
```

Onde `access_token` é obtido via `/v1/auth/login` ou `/v1/auth/refresh`.

## Response Format

Todas as respostas são JSON. Erros seguem o formato:

```json
{
  "error": "mensagem de erro descritivo"
}
```

---

## Public Endpoints

### GET /

Informações básicas sobre a API.

**Response (200 OK):**
```json
{
  "name": "i9pay"
}
```

---

### GET /v1/health

Health check da aplicação e dependências.

**Response (200 OK):**
```json
{
  "status": "ok",
  "services": {
    "database": "ok"
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "error": "database unavailable: connection refused"
}
```

---

### POST /v1/auth/login

Autentica um tenant e emite um par de tokens.

**Request:**
```json
{
  "email": "tenant@example.com",
  "password": "senha123"
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2026-03-13T10:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2026-03-20T10:00:00Z"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "email and password are required"
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "invalid credentials"
}
```

---

### POST /v1/auth/refresh

Emite um novo par de tokens usando um refresh token válido.

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2026-03-13T10:15:00Z",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token_expires_at": "2026-03-20T10:00:00Z"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "refresh_token is required"
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "invalid or expired refresh token"
}
```

---

## Protected Endpoints

Todas as rotas abaixo requerem `Authorization: Bearer <access_token>`.

### GET /v1/me

Retorna informações do tenant autenticado extraídas do JWT.

**Request Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (200 OK):**
```json
{
  "tenant_id": "550e8400-e29b-41d4-a716-446655440000",
  "issued_at": "2026-03-13T10:00:00Z",
  "expires_at": "2026-03-13T10:15:00Z"
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "authorization header required"
}
```

**Response (401 Unauthorized):**
```json
{
  "error": "invalid or expired token"
}
```

---

## HTTP Status Codes

| Code | Meaning | Exemplo |
|------|---------|---------|
| 200 | OK | Requisição bem-sucedida |
| 400 | Bad Request | Validação falhou (payload inválido) |
| 401 | Unauthorized | Token ausente ou inválido |
| 403 | Forbidden | Acesso proibido |
| 404 | Not Found | Recurso não encontrado |
| 409 | Conflict | Conflito (ex: email já existe) |
| 500 | Internal Server Error | Erro não esperado |
| 503 | Service Unavailable | Dependência indisponível (ex: DB) |

---

## JWT Token Details

### Access Token

- **Tipo**: JWT (HS256)
- **TTL**: 15 minutos
- **Claims**:
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

- **Tipo**: JWT (HS256)
- **TTL**: 7 dias
- **Claims**:
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

## cURL Examples

### Health Check
```bash
curl http://localhost:8080/v1/health
```

### Login
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "tenant@example.com",
    "password": "senha123"
  }'
```

### Get Current User
```bash
curl http://localhost:8080/v1/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

### Refresh Token
```bash
curl -X POST http://localhost:8080/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

### Using jq to Extract Tokens
```bash
# Login e salva tokens em variáveis
RESPONSE=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "tenant@example.com", "password": "senha123"}')

ACCESS_TOKEN=$(echo $RESPONSE | jq -r '.access_token')
REFRESH_TOKEN=$(echo $RESPONSE | jq -r '.refresh_token')

# Use o token
curl http://localhost:8080/v1/me \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

---

## API Collections

Você pode importar as rotas em ferramentas como **Postman**, **Insomnia** ou **Thunder Client**.

### Postman Collection (manual)

1. Crie nova Collection "i9pay"
2. Adicione requests POST e GET com os URLs acima
3. Configure headers de autenticação
4. Use environment variables para armazenar tokens

### Example com curl (script)

```bash
#!/bin/bash

API_BASE="http://localhost:8080"

# 1. Health Check
echo "=== Health Check ==="
curl $API_BASE/v1/health

# 2. Login
echo "\n=== Login ==="
TOKENS=$(curl -s -X POST "$API_BASE/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "test123"}')

echo "$TOKENS" | jq .

ACCESS_TOKEN=$(echo "$TOKENS" | jq -r '.access_token')

# 3. Get Current User
echo "\n=== Get Current User ==="
curl "$API_BASE/v1/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" | jq .
```

---

## Rate Limiting

Atualmente não implementado. Roadmap para futuras versões.

## Webhooks

Atualmente não implementado. Roadmap para futuras versões.
