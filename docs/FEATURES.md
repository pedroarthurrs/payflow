# Funcionalidades a Implementar

Documento de requisitos funcionais e técnicos para os módulos ainda não implementados.

Cada módulo descreve:
- **Estado atual** — o que já existe (schema, domínio, etc.)
- **O que implementar** — camadas que precisam ser criadas
- **Regras de negócio** — comportamento esperado
- **Requisitos técnicos** — decisões de implementação
- **Referências de API** — contratos HTTP esperados

---

## Módulos

- [User Permissions & Multi-Organization](#user-permissions--multi-organization)
- [Plans](#plans)
- [Subscriptions](#subscriptions)

- [Subscriptions](#subscriptions)

---

## User Permissions & Multi-Organization

### Visão Geral

Esta feature envolve duas mudanças fundamentais interligadas:

1. **Identidade global de usuário** — separar credenciais de acesso (email + senha) do vínculo com uma organization específica, permitindo que um mesmo login acesse múltiplas organizations
2. **Sistema de permissões granular** — substituir a hierarquia simples de roles por um modelo de roles expandido (`owner > admin > developer > support`) com permissões individuais configuráveis por usuário

Ambas as mudanças constituem uma **refatoração breaking** na camada de banco e de autenticação. Devem ser implementadas juntas e incluem migração de dados dos registros existentes.

---

### Estado atual

| Camada | Status |
|--------|--------|
| Tabela `users` (org-scoped) | ✅ Existe |
| Roles `owner / admin / member` | ✅ Existe |
| Login por email (single-org) | ✅ Implementado |
| Tabela `user_accounts` (global) | ❌ Não existe |
| Tabela `organization_members` | ❌ Não existe |
| Tabela `user_permissions` | ❌ Não existe |
| Roles `developer / support` | ❌ Não existe |
| Login multi-org com seleção | ❌ Não implementado |
| Middleware `RequirePermission` | ❌ Não existe |

**O que muda no contrato atual:**
- `users` é substituído por `user_accounts` (identidade) + `organization_members` (vínculo)
- `role` em `users` passa para `organization_members`
- `GET /v1/auth/login` ganha resposta condicional dependendo do número de orgs vinculadas
- `users:member` é renomeado para `developer` e `support` (novo papel)

---

### Modelo de Dados

#### Migração `000004_user_accounts_and_permissions`

```sql
-- 1. Identidade global de usuário (desacoplada de org)
CREATE TABLE user_accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email        VARCHAR(255) NOT NULL UNIQUE,
    name         VARCHAR(255) NOT NULL,
    password     VARCHAR(255) NOT NULL,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at   TIMESTAMP WITH TIME ZONE
);

-- 2. Vínculo usuário ↔ organization com role
CREATE TABLE organization_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
    role            VARCHAR(20) NOT NULL DEFAULT 'support'
                    CHECK (role IN ('owner', 'admin', 'developer', 'support')),
    invited_by      UUID REFERENCES user_accounts(id),
    joined_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    CONSTRAINT unique_member_per_org UNIQUE (organization_id, user_account_id)
);

-- 3. Permissões individuais (grants e revocations sobre os defaults do role)
CREATE TABLE user_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES user_accounts(id) ON DELETE CASCADE,
    permission      VARCHAR(50) NOT NULL,
    granted         BOOLEAN NOT NULL DEFAULT true, -- false = revogação explícita
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT unique_user_permission UNIQUE (organization_id, user_account_id, permission)
);
```

**Migração de dados existentes** (dentro da mesma migration):

```sql
-- 1. Criar user_accounts a partir de users (dedupado por email)
INSERT INTO user_accounts (id, email, name, password, created_at, updated_at, deleted_at)
SELECT DISTINCT ON (email)
    id, email, name, password, created_at, updated_at, deleted_at
FROM users
WHERE deleted_at IS NULL
ORDER BY email, created_at ASC;

-- 2. Criar organization_members a partir de users
INSERT INTO organization_members (organization_id, user_account_id, role, joined_at, created_at, updated_at)
SELECT u.organization_id, ua.id,
    CASE WHEN u.role = 'member' THEN 'support' ELSE u.role END,
    u.created_at, u.created_at, u.updated_at
FROM users u
JOIN user_accounts ua ON ua.email = u.email
WHERE u.deleted_at IS NULL;

-- 3. A tabela users pode ser mantida como view de compatibilidade ou removida
--    na migration seguinte após verificação
```

**Índices:**

```sql
CREATE INDEX idx_user_accounts_email     ON user_accounts (email);
CREATE INDEX idx_org_members_org         ON organization_members (organization_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_members_user        ON organization_members (user_account_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_user_permissions_user   ON user_permissions (organization_id, user_account_id);
```

---

### Novo Modelo de Roles

```
owner (4) > admin (3) > developer (2) > support (1)
```

| Role | Descrição |
|------|-----------|
| `owner` | Proprietário da organization. Único por org. Acesso total irrestrito. |
| `admin` | Gerencia usuários, configura integrações, acesso operacional completo exceto transferência de ownership. |
| `developer` | Foco técnico: credenciais, webhooks, logs, leitura de dados financeiros. Não gerencia usuários. |
| `support` | Foco operacional: clientes, assinaturas, reembolsos. Sem acesso a configurações técnicas. |

**Compatibilidade:** o valor `member` deixa de ser aceito no CHECK constraint. A migração converte `member` → `support` automaticamente.

---

### Catálogo de Permissões

Permissões são strings no formato `{recurso}:{ação}`. Definidas em código (`domain/permission.go`), não em banco.

| Permissão | owner | admin | developer | support |
|-----------|:-----:|:-----:|:---------:|:-------:|
| `users:read` | ✅ | ✅ | ❌ | ❌ |
| `users:invite` | ✅ | ✅ | ❌ | ❌ |
| `users:manage` | ✅ | ❌ | ❌ | ❌ |
| `credentials:read` | ✅ | ✅ | ✅ | ❌ |
| `credentials:manage` | ✅ | ✅ | ✅ | ❌ |
| `customers:read` | ✅ | ✅ | ✅ | ✅ |
| `customers:write` | ✅ | ✅ | ✅ | ✅ |
| `customers:delete` | ✅ | ✅ | ❌ | ❌ |
| `products:read` | ✅ | ✅ | ✅ | ✅ |
| `products:write` | ✅ | ✅ | ❌ | ❌ |
| `plans:read` | ✅ | ✅ | ✅ | ✅ |
| `plans:write` | ✅ | ✅ | ❌ | ❌ |
| `subscriptions:read` | ✅ | ✅ | ✅ | ✅ |
| `subscriptions:write` | ✅ | ✅ | ❌ | ✅ |
| `subscriptions:items` | ✅ | ✅ | ❌ | ❌ |
| `payments:read` | ✅ | ✅ | ✅ | ✅ |
| `payments:charge` | ✅ | ✅ | ✅ | ❌ |
| `payments:refund` | ✅ | ✅ | ❌ | ✅ |
| `payments:cancel` | ✅ | ✅ | ❌ | ✅ |
| `webhooks:read` | ✅ | ✅ | ✅ | ❌ |
| `webhooks:manage` | ✅ | ✅ | ✅ | ❌ |
| `logs:read` | ✅ | ✅ | ✅ | ❌ |

Permissões individuais em `user_permissions` sobrescrevem os defaults da tabela acima:
- `granted = true` → concede permissão não prevista no role (ex: dar `credentials:read` a um `support`)
- `granted = false` → revoga permissão prevista no role (ex: proibir `payments:charge` para um `developer` específico)

---

### Fluxo de Login Multi-Organization

O login passa a suportar dois passos quando o usuário pertence a múltiplas organizations.

#### Passo 1 — Autenticar

`POST /v1/auth/login` com `email + password` (sem `organization_id`):

- **1 org**: emite `access_token` + `refresh_token` normalmente (backward compatible)
- **N orgs**: retorna `200` com lista de organizations e um `session_token` temporário (TTL: 5 min, uso único)
- **0 orgs**: `401 Unauthorized`

**Resposta multi-org (200 OK):**
```json
{
  "requires_org_selection": true,
  "session_token": "eyJ...",
  "organizations": [
    { "id": "uuid-org-a", "name": "Acme Corp" },
    { "id": "uuid-org-b", "name": "Beta Ltda" }
  ]
}
```

O `session_token` é um JWT de curta duração com claims `{ user_account_id, token_type: "org_selection" }`. Não é um access token e não autoriza nenhuma operação além de `POST /v1/auth/select-organization`.

#### Passo 2 — Selecionar Organization

`POST /v1/auth/select-organization`:

```json
{
  "session_token": "eyJ...",
  "organization_id": "uuid-org-a"
}
```

**Resposta (200 OK):** mesmo formato atual de `POST /v1/auth/login` (access + refresh tokens)

**Erros:**
- `401` — session_token inválido ou expirado
- `403` — usuário não é membro da organization informada

#### Login com `organization_id` direto (opcional)

Para clients que já conhecem a org (ex: frontend que recorda a última org usada), `organization_id` pode ser enviado no passo 1:

```json
{
  "email": "user@example.com",
  "password": "senha123",
  "organization_id": "uuid-org-a"
}
```

Se `organization_id` for fornecido e o usuário for membro, emite token diretamente (sem segundo passo).

---

### Requisitos Técnicos

#### `internal/domain/permission.go`

```go
// Permission representa uma capability granular no sistema.
type Permission string

// Catálogo completo de permissões disponíveis.
const (
    PermUsersRead         Permission = "users:read"
    PermUsersInvite       Permission = "users:invite"
    PermUsersManage       Permission = "users:manage"
    PermCredentialsRead   Permission = "credentials:read"
    PermCredentialsManage Permission = "credentials:manage"
    PermCustomersRead     Permission = "customers:read"
    PermCustomersWrite    Permission = "customers:write"
    PermCustomersDelete   Permission = "customers:delete"
    PermProductsRead      Permission = "products:read"
    PermProductsWrite     Permission = "products:write"
    PermPlansRead         Permission = "plans:read"
    PermPlansWrite        Permission = "plans:write"
    PermSubscriptionsRead  Permission = "subscriptions:read"
    PermSubscriptionsWrite Permission = "subscriptions:write"
    PermSubscriptionsItems Permission = "subscriptions:items"
    PermPaymentsRead      Permission = "payments:read"
    PermPaymentsCharge    Permission = "payments:charge"
    PermPaymentsRefund    Permission = "payments:refund"
    PermPaymentsCancel    Permission = "payments:cancel"
    PermWebhooksRead      Permission = "webhooks:read"
    PermWebhooksManage    Permission = "webhooks:manage"
    PermLogsRead          Permission = "logs:read"
)

// roleDefaultPermissions mapeia cada role ao conjunto de permissões padrão.
var roleDefaultPermissions = map[Role][]Permission{
    RoleOwner:     { /* todas */ },
    RoleAdmin:     { PermUsersRead, PermUsersInvite, PermCredentialsRead, PermCredentialsManage, /* ... */ },
    RoleDeveloper: { PermCredentialsRead, PermCredentialsManage, PermCustomersRead, /* ... */ },
    RoleSupport:   { PermCustomersRead, PermCustomersWrite, PermSubscriptionsRead, PermSubscriptionsWrite, /* ... */ },
}

// DefaultPermissionsFor returns the default permission set for a given role.
func DefaultPermissionsFor(r Role) []Permission

// HasDefaultPermission returns true if the role has a given permission by default.
func HasDefaultPermission(r Role, p Permission) bool
```

#### `internal/domain/user.go` (atualizado)

```go
// Roles expandidos
const (
    RoleOwner     Role = "owner"
    RoleAdmin     Role = "admin"
    RoleDeveloper Role = "developer"
    RoleSupport   Role = "support"
)

func RoleLevel(r Role) int {
    switch r {
    case RoleOwner:     return 4
    case RoleAdmin:     return 3
    case RoleDeveloper: return 2
    case RoleSupport:   return 1
    default:            return 0
    }
}

// UserAccount representa a identidade global de um usuário (desacoplada de org).
type UserAccount struct {
    ID        uuid.UUID
    Email     string
    Name      string
    CreatedAt time.Time
    UpdatedAt time.Time
}

// OrganizationMember representa o vínculo entre um UserAccount e uma Organization.
type OrganizationMember struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    UserAccountID  uuid.UUID
    Email          string  // desnormalizado para exibição
    Name           string  // desnormalizado para exibição
    Role           Role
    JoinedAt       time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// UserPermission representa um grant ou revocation individual.
type UserPermission struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    UserAccountID  uuid.UUID
    Permission     Permission
    Granted        bool // true=grant, false=revoke
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// UserAccountRepository — operações na identidade global
type UserAccountRepository interface {
    Create(ctx context.Context, input CreateUserAccountInput) (*UserAccount, error)
    GetByID(ctx context.Context, id uuid.UUID) (*UserAccount, error)
    GetByEmail(ctx context.Context, email string) (*UserAccount, error)
    Update(ctx context.Context, id uuid.UUID, input UpdateUserAccountInput) (*UserAccount, error)
    SoftDelete(ctx context.Context, id uuid.UUID) error
}

// MemberRepository — operações de membership
type MemberRepository interface {
    AddMember(ctx context.Context, orgID, userAccountID uuid.UUID, role Role, invitedBy *uuid.UUID) (*OrganizationMember, error)
    GetMember(ctx context.Context, orgID, userAccountID uuid.UUID) (*OrganizationMember, error)
    ListMembers(ctx context.Context, orgID uuid.UUID, params ListParams) ([]OrganizationMember, int32, error)
    UpdateMemberRole(ctx context.Context, orgID, userAccountID uuid.UUID, role Role) (*OrganizationMember, error)
    RemoveMember(ctx context.Context, orgID, userAccountID uuid.UUID) error
    ListOrganizationsForUser(ctx context.Context, userAccountID uuid.UUID) ([]OrganizationRef, error)
}

// PermissionRepository — operações de permissões individuais
type PermissionRepository interface {
    Upsert(ctx context.Context, orgID, userAccountID uuid.UUID, permission Permission, granted bool) error
    Delete(ctx context.Context, orgID, userAccountID uuid.UUID, permission Permission) error
    ListForUser(ctx context.Context, orgID, userAccountID uuid.UUID) ([]UserPermission, error)
}

// AuthRepository (atualizado)
type AuthRepository interface {
    GetUserAccountByEmail(ctx context.Context, email string) (*AuthUserAccount, error)
    ListMembershipsForUser(ctx context.Context, userAccountID uuid.UUID) ([]AuthMembership, error)
    GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
    UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error
}

type AuthUserAccount struct {
    ID       uuid.UUID
    Password string
}

type AuthMembership struct {
    OrganizationID uuid.UUID
    OrgName        string
    Role           Role
}

type OrganizationRef struct {
    ID   uuid.UUID
    Name string
}
```

#### Novos erros de domínio

```go
var ErrOrgSelectionRequired = errors.New("organization selection required")
var ErrInvalidSessionToken  = errors.New("invalid or expired session token")
var ErrAlreadyMember        = errors.New("user is already a member of this organization")
var ErrOwnerCannotLeave     = errors.New("owner cannot be removed from organization")
var ErrPermissionNotFound   = errors.New("permission not found")
```

#### Atualização do Token Service (`auth/token/token.go`)

Adicionar `TokenType` para seleção de org:

```go
const (
    TypeAccess      TokenType = "access"
    TypeRefresh     TokenType = "refresh"
    TypeOrgSelect   TokenType = "org_selection" // curta duração, uso único
)

// Claims — sem mudanças de struct; UserID passa a referenciar user_account_id
```

O `session_token` de seleção de org usa `TypeOrgSelect` com TTL de 5 minutos.

#### Novo Middleware `RequirePermission`

```go
// RequirePermission retorna middleware que valida permissão granular.
// Resolve a permissão efetiva combinando defaults do role + overrides individuais.
// Usa cache in-memory com TTL de 5 minutos por (org_id, user_id).
func RequirePermission(p domain.Permission) gin.HandlerFunc

// PermissionResolver — interface para injeção no middleware
type PermissionResolver interface {
    HasPermission(ctx context.Context, orgID, userAccountID uuid.UUID, p domain.Permission) (bool, error)
}
```

**Lógica de resolução:**
```
1. Verificar cache (orgID + userAccountID + permission)
2. Se cache MISS:
   a. Obter role do membro em organization_members
   b. Verificar HasDefaultPermission(role, permission)
   c. Carregar user_permissions para o (orgID, userAccountID)
   d. Aplicar overrides: grants adicionam, revocations removem
   e. Armazenar resultado em cache com TTL=5min
3. Se false → 403 "insufficient permissions"
```

O middleware existente `RequireRole(minRole)` é mantido e continua funcionando para endpoints que não precisam de granularidade.

#### `internal/service/auth.go` (atualizado)

```go
// Login dinâmico:
// - API Key → system token (sem mudança)
// - Email + Password + organization_id → token direto
// - Email + Password (sem org) + 1 membership → token direto
// - Email + Password (sem org) + N memberships → session_token + lista de orgs

type LoginResult struct {
    Pair                 *token.Pair       // nil se requer seleção de org
    RequiresOrgSelection bool
    SessionToken         string            // preenchido se RequiresOrgSelection=true
    Organizations        []domain.OrganizationRef // preenchido se RequiresOrgSelection=true
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*LoginResult, error)
func (s *AuthService) SelectOrganization(ctx context.Context, sessionToken, orgID string) (*token.Pair, error)
```

#### `internal/service/member.go` (novo)

```go
type MemberService struct {
    accounts     domain.UserAccountRepository
    members      domain.MemberRepository
    permissions  domain.PermissionRepository
    passwordHasher *crypto.PasswordHasher
    txMgr        domain.TxManager
}

// InviteMember — cria user_account (se não existir) + organization_member atomicamente
func (s *MemberService) InviteMember(ctx context.Context, orgID uuid.UUID, input InviteMemberInput) (*domain.OrganizationMember, *string, error)
// Retorna (member, rawTemporaryPassword, error)
// Se email não existir em user_accounts, cria conta com senha temporária aleatória

// ListMembers, GetMember, UpdateMemberRole, RemoveMember
// GetPermissions, SetPermission, DeletePermission
// ResolvePermissions — retorna mapa de todos as permissões efetivas do usuário na org
```

---

### API Esperada

#### Autenticação

| Método | Rota | Descrição | Auth |
|--------|------|-----------|------|
| `POST` | `/v1/auth/login` | Login (retorno condicional) | público |
| `POST` | `/v1/auth/select-organization` | Selecionar org após login multi-org | session_token |
| `POST` | `/v1/auth/refresh` | Renovar tokens | sem mudança |

**POST /v1/auth/login — Resposta single-org (200 OK):**
```json
{
  "access_token": "eyJ...",
  "access_token_expires_at": "2026-04-13T10:15:00Z",
  "refresh_token": "eyJ...",
  "refresh_token_expires_at": "2026-04-20T10:00:00Z"
}
```

**POST /v1/auth/login — Resposta multi-org (200 OK):**
```json
{
  "requires_org_selection": true,
  "session_token": "eyJ...",
  "organizations": [
    { "id": "uuid-a", "name": "Acme Corp" },
    { "id": "uuid-b", "name": "Beta Ltda" }
  ]
}
```

**POST /v1/auth/select-organization — Requisição:**
```json
{
  "session_token": "eyJ...",
  "organization_id": "uuid-a"
}
```
**Resposta (200 OK):** mesmo formato da resposta single-org.

---

#### Membros da Organization

| Método | Rota | Descrição | Role Mínimo |
|--------|------|-----------|-------------|
| `POST` | `/v1/members` | Convidar membro | admin (`users:invite`) |
| `GET` | `/v1/members` | Listar membros | admin (`users:read`) |
| `GET` | `/v1/members/:member_id` | Buscar membro | admin (`users:read`) |
| `PUT` | `/v1/members/:member_id/role` | Atualizar role | owner (`users:manage`) |
| `DELETE` | `/v1/members/:member_id` | Remover membro | owner (`users:manage`) |
| `GET` | `/v1/members/:member_id/permissions` | Listar permissões efetivas | owner (`users:manage`) |
| `PUT` | `/v1/members/:member_id/permissions` | Atualizar permissões individuais | owner (`users:manage`) |

> Os endpoints de `/v1/users` existentes são mantidos durante a migração e depois deprecados em favor de `/v1/members`.

**POST /v1/members — Requisição:**
```json
{
  "email": "dev@empresa.com",
  "name": "Carlos Dev",
  "role": "developer"
}
```

Se o email já existir em `user_accounts`, vincula à org sem criar nova conta. Se não existir, cria `user_account` com senha temporária aleatória.

**Resposta (201 Created):**
```json
{
  "id": "uuid-member",
  "user_account_id": "uuid-account",
  "email": "dev@empresa.com",
  "name": "Carlos Dev",
  "role": "developer",
  "joined_at": "2026-04-13T10:00:00Z",
  "temporary_password": "Tmp@X9z2k..."
}
```

`temporary_password` só aparece quando a conta foi criada nesta requisição. Em reuso de conta existente, o campo é `null`.

**PUT /v1/members/:member_id/permissions — Requisição:**
```json
{
  "grants": ["credentials:read", "logs:read"],
  "revocations": ["payments:charge"]
}
```

`grants` e `revocations` são aplicados sobre os defaults do role. Permissões não listadas permanecem com o valor default. Enviar uma permissão em ambos é erro de validação.

**GET /v1/members/:member_id/permissions — Resposta (200 OK):**
```json
{
  "role": "developer",
  "effective_permissions": [
    { "permission": "credentials:read",   "source": "role" },
    { "permission": "credentials:manage", "source": "role" },
    { "permission": "logs:read",          "source": "grant" },
    { "permission": "payments:charge",    "source": "revoked" }
  ]
}
```

`source` pode ser `"role"` (default), `"grant"` (concedida individualmente) ou `"revoked"` (revogada individualmente).

---

#### Perfil do Usuário Autenticado (`/v1/me`)

`GET /v1/me` passa a retornar também o role e as permissões efetivas:

```json
{
  "user_account_id": "uuid",
  "organization_id": "uuid",
  "email": "user@example.com",
  "name": "Alice",
  "role": "developer",
  "permissions": ["credentials:read", "customers:read", "payments:read"],
  "issued_at": "2026-04-13T10:00:00Z",
  "expires_at": "2026-04-13T10:15:00Z"
}
```

---

### Mapeamento de Endpoints Existentes para Permissões

Os endpoints existentes devem migrar gradualmente de `RequireRole` para `RequirePermission`. Abaixo o mapeamento:

| Endpoint atual | RequireRole atual | RequirePermission futura |
|---------------|-------------------|--------------------------|
| `POST /v1/users` | owner | `users:invite` |
| `GET /v1/users` | admin | `users:read` |
| `GET /v1/users/:id` | admin | `users:read` |
| `PUT /v1/users/:id` | owner | `users:manage` |
| `DELETE /v1/users/:id` | owner | `users:manage` |
| `POST /v1/credentials` | admin | `credentials:manage` |
| `GET /v1/credentials` | admin | `credentials:read` |
| `GET /v1/credentials/:id` | admin | `credentials:read` |
| `DELETE /v1/credentials/:id` | owner | `credentials:manage` |
| `POST /v1/credentials/:id/rotate` | owner | `credentials:manage` |
| `GET /v1/credentials/logs` | admin | `logs:read` |
| `GET /v1/credentials/logs/:id` | admin | `logs:read` |
| `POST /v1/customers` | qualquer | `customers:write` |
| `GET /v1/customers` | qualquer | `customers:read` |
| `PUT /v1/customers/:id` | qualquer | `customers:write` |
| `DELETE /v1/customers/:id` | qualquer | `customers:delete` |
| `POST /v1/payments` | qualquer | `payments:charge` |
| `GET /v1/payments` | qualquer | `payments:read` |
| `POST /v1/payments/:id/refund` | member | `payments:refund` |
| `POST /v1/payments/:id/cancel` | member | `payments:cancel` |
| `POST /v1/webhooks` | admin | `webhooks:manage` |
| `GET /v1/webhooks` | admin | `webhooks:read` |
| `PUT /v1/webhooks/:id` | admin | `webhooks:manage` |
| `DELETE /v1/webhooks/:id` | owner | `webhooks:manage` |
| `GET /v1/webhooks/logs` | admin | `logs:read` |

---

### Ordem de Implementação Recomendada

Esta feature deve ser implementada em duas fases para minimizar risco:

**Fase 1 — Estrutura de dados e migração**
1. Escrever e aplicar migration `000004` (criar tabelas + migrar dados)
2. Implementar domain: `user_account.go`, `permission.go`, atualizar `user.go`
3. Implementar queries SQL + `make sqlc`
4. Implementar repositories: `UserAccountRepo`, `MemberRepo`, `PermissionRepo`
5. Atualizar `AuthRepo` para usar `user_accounts` + `organization_members`

**Fase 2 — Serviços, middleware e handlers**
1. Atualizar `AuthService.Login` para fluxo multi-org
2. Adicionar `AuthService.SelectOrganization`
3. Implementar `MemberService`
4. Implementar `PermissionResolver` com cache in-memory
5. Implementar middleware `RequirePermission`
6. Criar `MemberHandler` (`/v1/members`)
7. Atualizar `MeHandler` para retornar permissions
8. Migrar endpoints existentes de `RequireRole` para `RequirePermission` gradualmente

---

## Plans

### Estado atual

| Camada | Status |
|--------|--------|
| Migration (`plans`) | ✅ Existe |
| Index (`idx_plans_product_id`) | ✅ Existe |
| Domain (`domain/`) | ❌ Não existe |
| Service (`service/`) | ❌ Não existe |
| Handler (`api/handler/`) | ❌ Não existe |
| SQL Queries (`database/queries/`) | ❌ Não existe |
| Repository (`database/repository/`) | ❌ Não existe |

O modelo `Product` já está implementado. Plans são filhos de Products e devem ser gerenciados como sub-recursos de `/products/:product_id/plans`.

### Schema (referência)

```sql
CREATE TABLE plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    product_id      UUID NOT NULL,
    name            VARCHAR(255),
    slug            TEXT,
    amount_cents    INTEGER NOT NULL CHECK (amount_cents >= 0),
    currency        VARCHAR(3) DEFAULT 'BRL',
    interval        VARCHAR(20) CHECK (interval IN ('day', 'week', 'month', 'year')),
    interval_count  INTEGER DEFAULT 1,
    active          BOOLEAN DEFAULT true,
    version         INTEGER DEFAULT 1,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at      TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_plans_product_organization
        FOREIGN KEY (organization_id, product_id)
        REFERENCES products (organization_id, id) ON DELETE CASCADE,

    CONSTRAINT unique_plan_organization_id UNIQUE (organization_id, id)
);
```

### Regras de Negócio

**Criação**
- `product_id` deve existir e pertencer à mesma `organization_id`
- `amount_cents` deve ser `>= 0` (plano gratuito é permitido)
- `interval` é obrigatório: `day`, `week`, `month` ou `year`
- `interval_count` deve ser `>= 1` (padrão: `1`)
- `currency` deve ser um código ISO 4217 de 3 letras (padrão: `BRL`)
- `slug` é opcional; se fornecido, deve ser único por `organization_id`
- `version` começa em `1` e não é editável pelo usuário

**Atualização**
- Alterações de `amount_cents` ou `interval`/`interval_count` não retroagem em assinaturas ativas — o novo valor entra em vigor apenas em novas assinaturas ou renovações futuras
- Ao alterar campos de precificação (`amount_cents`, `interval`, `interval_count`), incrementar `version` automaticamente
- `name`, `active` e `slug` podem ser alterados sem incremento de versão

**Exclusão**
- Soft delete (`deleted_at`): plan não aparece em listagens, mas assinaturas existentes que referenciam o plan não são afetadas
- Hard delete: só permitido se não houver `subscription_items` ativos referenciando o plan; retornar `409 Conflict` caso contrário

**Listagem**
- Filtrar por `active` (query param `?active=true/false`), padrão: retornar todos
- Ordenar por `created_at DESC`
- Suportar paginação offset

### Requisitos Técnicos

**`internal/domain/plan.go`**
```go
type PlanRepository interface {
    Create(ctx context.Context, organizationID uuid.UUID, input PlanInput) (*Plan, error)
    GetByID(ctx context.Context, organizationID, planID uuid.UUID) (*Plan, error)
    List(ctx context.Context, organizationID, productID uuid.UUID, params PlanListParams) ([]Plan, int32, error)
    Update(ctx context.Context, organizationID, planID uuid.UUID, input PlanInput) (*Plan, error)
    SoftDelete(ctx context.Context, organizationID, planID uuid.UUID) error
    HardDelete(ctx context.Context, organizationID, planID uuid.UUID) error
}

type Plan struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    ProductID      uuid.UUID
    Name           string
    Slug           *string
    AmountCents    int32
    Currency       string
    Interval       string
    IntervalCount  int32
    Active         bool
    Version        int32
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type PlanInput struct {
    Name          string
    Slug          *string
    AmountCents   int32
    Currency      string
    Interval      string
    IntervalCount int32
    Active        bool
}

type PlanListParams struct {
    ListParams
    Active *bool
}
```

Novos erros de domínio a adicionar em `domain/errors.go`:
```go
var ErrPlanNotFound      = errors.New("plan not found")
var ErrPlanInUse         = errors.New("plan is referenced by active subscriptions")
```

**Lógica de versão no service**

No `PlanService.Update`, comparar os campos de precificação antes de persistir:
```go
if input.AmountCents != current.AmountCents ||
   input.Interval != current.Interval ||
   input.IntervalCount != current.IntervalCount {
    input.Version = current.Version + 1
}
```

**`internal/database/queries/plan.sql`**

Queries mínimas:
- `CreatePlan :one`
- `GetPlanByID :one` — filtra `deleted_at IS NULL`
- `ListPlansByProduct :many` — filtra por `organization_id`, `product_id`, `deleted_at IS NULL`, suporte a `active` opcional
- `CountPlansByProduct :one`
- `UpdatePlan :one`
- `SoftDeletePlan :exec`
- `HardDeletePlan :exec` — apenas se não há `subscription_items` ativos
- `PlanHasActiveSubscriptions :one` — usado antes do hard delete

**Middleware e RBAC**

Usar o mesmo padrão dos handlers existentes. Plans são sub-recursos de Products:
- `POST`, `PUT`, `DELETE`: role mínimo `admin`
- `GET`: qualquer role autenticado

### API Esperada

Rotas aninhadas sob `/products/:product_id/plans`:

| Método | Rota | Descrição | Role Mínimo |
|--------|------|-----------|-------------|
| `POST` | `/v1/products/:product_id/plans` | Criar plano | admin |
| `GET` | `/v1/products/:product_id/plans` | Listar planos do produto | qualquer |
| `GET` | `/v1/products/:product_id/plans/:plan_id` | Buscar plano | qualquer |
| `PUT` | `/v1/products/:product_id/plans/:plan_id` | Atualizar plano | admin |
| `DELETE` | `/v1/products/:product_id/plans/:plan_id` | Remover plano | owner |

**POST /v1/products/:product_id/plans — Requisição:**
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

**Resposta (201 Created):**
```json
{
  "id": "uuid",
  "product_id": "uuid",
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

**GET /v1/products/:product_id/plans — Resposta (200 OK):**
```json
{
  "data": [ /* array de plans */ ],
  "limit": 20,
  "offset": 0,
  "total": 3
}
```

**Resposta (404 Not Found)** — product_id inválido ou plan_id não encontrado:
```json
{ "error": "plan not found" }
```

**Resposta (409 Conflict)** — hard delete com subscriptions ativas:
```json
{ "error": "plan is referenced by active subscriptions" }
```

---

## Subscriptions

### Estado atual

| Camada | Status |
|--------|--------|
| Migration (`subscriptions`, `subscription_items`, `subscription_status_history`, `subscription_schedules`) | ✅ Existe |
| Trigger de histórico de status (`tr_subscriptions_status_history`) | ✅ Existe |
| Índices de performance | ✅ Existem |
| Domain (`domain/`) | ❌ Não existe |
| Service (`service/`) | ❌ Não existe |
| Handler (`api/handler/`) | ❌ Não existe |
| SQL Queries (`database/queries/`) | ❌ Não existe |
| Repository (`database/repository/`) | ❌ Não existe |

Plans, Customers e Products já estão implementados. Subscriptions dependem dos três.

### Schema (referência)

```sql
-- Status possíveis
CREATE TYPE subscription_status_type AS ENUM (
    'trialing',    -- Período de teste (acesso liberado)
    'active',      -- Ativo
    'past_due',    -- Pagamento vencido, acesso mantido
    'unpaid',      -- Pagamento vencido, acesso bloqueado
    'canceled',    -- Cancelado permanentemente
    'paused',      -- Pausado (acesso bloqueado, pode retomar)
    'incomplete'   -- Pagamento inicial em processamento
);

CREATE TABLE subscriptions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id      UUID NOT NULL REFERENCES organizations(id),
    customer_id          UUID NOT NULL,
    status               subscription_status_type NOT NULL,
    current_period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    current_period_end   TIMESTAMP WITH TIME ZONE NOT NULL,
    cancel_at_period_end BOOLEAN DEFAULT false,
    metadata             JSONB,
    created_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE subscription_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,
    plan_id         UUID NOT NULL,
    quantity        INTEGER DEFAULT 1 CHECK (quantity > 0),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE subscription_status_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,
    old_status      subscription_status_type,
    new_status      subscription_status_type NOT NULL,
    reason          VARCHAR(100),
    metadata        JSONB,
    event_source_id UUID,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE subscription_schedules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    subscription_id UUID NOT NULL,
    scheduled_at    TIMESTAMP WITH TIME ZONE NOT NULL,
    target_status   subscription_status_type,
    target_plan_id  UUID,
    applied_at      TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Ciclo de Vida e Máquina de Estados

O status da assinatura segue a FSM abaixo. Transições inválidas devem retornar `422 Unprocessable Entity`.

```
                ┌──────────┐
                │ trialing │
                └────┬─────┘
                     │ ativação (pagamento confirmado ou fim do trial)
                     ▼
  ┌──────────┐   ┌────────┐   ┌──────────┐
  │ canceled │◄──│ active │──►│ past_due │
  └──────────┘   └────┬───┘   └────┬─────┘
                      │            │ pagamento definitivamente não efetuado
                      │            ▼
                      │       ┌────────┐
                      │       │ unpaid │
                      │       └────────┘
                      │
                      ▼
                 ┌────────┐
                 │ paused │──► active (retomada)
                 └────────┘
```

**Transições válidas:**

| De | Para | Gatilho |
|----|------|---------|
| `trialing` | `active` | Pagamento do primeiro ciclo confirmado |
| `trialing` | `canceled` | Cancelamento durante trial |
| `active` | `past_due` | Falha na cobrança de renovação |
| `active` | `canceled` | Cancelamento imediato ou fim de período (`cancel_at_period_end=true`) |
| `active` | `paused` | Pausa solicitada pelo usuário ou schedule |
| `past_due` | `active` | Pagamento da fatura atrasada confirmado |
| `past_due` | `unpaid` | Expiração do prazo de tolerância |
| `past_due` | `canceled` | Cancelamento durante atraso |
| `unpaid` | `active` | Pagamento tardio confirmado |
| `unpaid` | `canceled` | Cancelamento por inadimplência definitiva |
| `paused` | `active` | Retomada |
| `paused` | `canceled` | Cancelamento enquanto pausado |
| `incomplete` | `active` | Pagamento inicial confirmado |
| `incomplete` | `canceled` | Pagamento inicial não concluído |

O trigger `tr_subscriptions_status_history` registra automaticamente cada mudança de status em `subscription_status_history`. **O service não precisa inserir no histórico manualmente.**

### Regras de Negócio

**Criação**
- `customer_id` deve existir e pertencer à mesma `organization_id`
- Ao menos um item (`items`) deve ser fornecido
- Cada item: `plan_id` deve existir, pertencer à mesma `organization_id` e estar `active = true`
- `quantity` deve ser `>= 1` (padrão: `1`)
- Status inicial: `trialing` se `trial_end` for fornecido e > `NOW()`; caso contrário, `incomplete` (aguarda cobrança do primeiro ciclo)
- `current_period_start` = `NOW()`
- `current_period_end` calculado a partir do `interval` e `interval_count` do plano do primeiro item
- Se a assinatura tiver múltiplos itens com intervalos diferentes, usar o plano de menor `interval_count` × `interval` como base do período

**Atualização de status (transição)**
- Validar a FSM antes de persistir
- `reason` é obrigatório para transições para `canceled`, `paused` e `unpaid`
- Transições `canceled` são irreversíveis

**`cancel_at_period_end`**
- Quando `true`, a assinatura permanece `active` até `current_period_end` e então transiciona para `canceled`
- O worker de renovação deve verificar esta flag antes de tentar renovar

**Renovação (lógica futura — escopo do worker)**
- Não é escopo desta iteração implementar a renovação automática
- O endpoint `POST /:id/renew` fica como placeholder para o próximo módulo (Invoices)

**Itens da assinatura**
- Na criação, os items são inseridos atomicamente junto com a subscription via `TxManager`
- Na atualização de itens (`PUT /:id/items`): substituição completa dos items (remover antigos e inserir novos) em uma única transação
- Não é permitido remover todos os items de uma assinatura ativa — manter ao menos um

**Schedules (agendamentos)**
- Um schedule define uma mudança de status futura ou troca de plano
- Um schedule pendente (`applied_at IS NULL`) por vez por subscription
- Se um novo schedule for criado e já existe um pendente, retornar `409 Conflict`
- O worker de schedules aplica as mudanças na data programada (escopo futuro)

**Filtros de listagem**
- Por `customer_id`, `status` (múltiplos valores via `?status=active&status=past_due`), `from`, `to` (por `created_at`)
- Paginação offset
- Ordenar por `created_at DESC`

### Requisitos Técnicos

**`internal/domain/subscription.go`**
```go
type SubscriptionStatus string

const (
    StatusTrialing   SubscriptionStatus = "trialing"
    StatusActive     SubscriptionStatus = "active"
    StatusPastDue    SubscriptionStatus = "past_due"
    StatusUnpaid     SubscriptionStatus = "unpaid"
    StatusCanceled   SubscriptionStatus = "canceled"
    StatusPaused     SubscriptionStatus = "paused"
    StatusIncomplete SubscriptionStatus = "incomplete"
)

type SubscriptionRepository interface {
    Create(ctx context.Context, organizationID uuid.UUID, input CreateSubscriptionInput) (*Subscription, error)
    GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*Subscription, error)
    List(ctx context.Context, organizationID uuid.UUID, params SubscriptionListParams) ([]Subscription, int32, error)
    UpdateStatus(ctx context.Context, organizationID, subscriptionID uuid.UUID, status SubscriptionStatus, reason string) (*Subscription, error)
    SetCancelAtPeriodEnd(ctx context.Context, organizationID, subscriptionID uuid.UUID, value bool) (*Subscription, error)
    UpdatePeriod(ctx context.Context, organizationID, subscriptionID uuid.UUID, start, end time.Time) error
}

type SubscriptionItemRepository interface {
    ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []SubscriptionItemInput) ([]SubscriptionItem, error)
    ListBySubscription(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionItem, error)
}

type SubscriptionScheduleRepository interface {
    Create(ctx context.Context, organizationID uuid.UUID, input CreateScheduleInput) (*SubscriptionSchedule, error)
    GetPending(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*SubscriptionSchedule, error)
    Cancel(ctx context.Context, organizationID, scheduleID uuid.UUID) error
    ListDue(ctx context.Context, before time.Time) ([]SubscriptionSchedule, error) // usado pelo worker
}

type SubscriptionHistoryRepository interface {
    List(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionStatusHistory, error)
}

type Subscription struct {
    ID                  uuid.UUID
    OrganizationID      uuid.UUID
    CustomerID          uuid.UUID
    Status              SubscriptionStatus
    CurrentPeriodStart  time.Time
    CurrentPeriodEnd    time.Time
    CancelAtPeriodEnd   bool
    Metadata            map[string]any
    Items               []SubscriptionItem
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

type SubscriptionItem struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    SubscriptionID uuid.UUID
    PlanID         uuid.UUID
    Quantity       int32
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type CreateSubscriptionInput struct {
    CustomerID  uuid.UUID
    TrialEnd    *time.Time
    Metadata    map[string]any
    Items       []SubscriptionItemInput
}

type SubscriptionItemInput struct {
    PlanID   uuid.UUID
    Quantity int32
}

type SubscriptionListParams struct {
    ListParams
    CustomerID *uuid.UUID
    Status     []SubscriptionStatus
    CreatedFrom *time.Time
    CreatedTo   *time.Time
}
```

Novos erros de domínio a adicionar em `domain/errors.go`:
```go
var ErrSubscriptionNotFound        = errors.New("subscription not found")
var ErrInvalidSubscriptionTransition = errors.New("invalid subscription status transition")
var ErrSubscriptionCanceled        = errors.New("subscription is already canceled")
var ErrNoSubscriptionItems         = errors.New("subscription must have at least one item")
var ErrScheduleAlreadyPending      = errors.New("subscription already has a pending schedule")
```

**FSM de Subscription**

Implementar em `internal/domain/subscription_fsm.go` seguindo o mesmo padrão de `payment_fsm.go`:
```go
// Transições válidas: map[de][]para
var validSubscriptionTransitions = map[SubscriptionStatus][]SubscriptionStatus{
    StatusTrialing:   {StatusActive, StatusCanceled},
    StatusActive:     {StatusPastDue, StatusCanceled, StatusPaused},
    StatusPastDue:    {StatusActive, StatusUnpaid, StatusCanceled},
    StatusUnpaid:     {StatusActive, StatusCanceled},
    StatusPaused:     {StatusActive, StatusCanceled},
    StatusIncomplete: {StatusActive, StatusCanceled},
}

func CanTransition(from, to SubscriptionStatus) bool
```

**`internal/service/subscription.go`**

Separar serviços por responsabilidade:
- `SubscriptionService` — CRUD e transições de status
- Injetar: `SubscriptionRepository`, `SubscriptionItemRepository`, `SubscriptionScheduleRepository`, `SubscriptionHistoryRepository`, `PlanRepository`, `CustomerRepository`, `TxManager`

Método de criação deve usar `TxManager.WithTx` para inserir subscription + items atomicamente.

**Cálculo de período**

Implementar helper `calculatePeriodEnd(start time.Time, interval string, intervalCount int) time.Time` em `service/subscription.go`:
```go
// interval="month", intervalCount=1 → start.AddDate(0, 1, 0)
// interval="year",  intervalCount=1 → start.AddDate(1, 0, 0)
// interval="week",  intervalCount=2 → start.AddDate(0, 0, 14)
// interval="day",   intervalCount=7 → start.AddDate(0, 0, 7)
```

**`internal/database/queries/subscription.sql`**

Queries mínimas:
- `CreateSubscription :one`
- `GetSubscriptionByID :one`
- `ListSubscriptions :many` (com filtros opcionais de status e customer)
- `CountSubscriptions :one`
- `UpdateSubscriptionStatus :one`
- `SetCancelAtPeriodEnd :one`
- `UpdateSubscriptionPeriod :exec`
- `DeleteSubscriptionItems :exec` (para replace atômico)
- `CreateSubscriptionItem :one`
- `ListSubscriptionItems :many`
- `CreateSubscriptionSchedule :one`
- `GetPendingSchedule :one`
- `CancelSubscriptionSchedule :exec`
- `ListDueSchedules :many` (para worker futuro)
- `ListSubscriptionHistory :many`

### API Esperada

| Método | Rota | Descrição | Role Mínimo |
|--------|------|-----------|-------------|
| `POST` | `/v1/subscriptions` | Criar assinatura | qualquer |
| `GET` | `/v1/subscriptions` | Listar assinaturas | qualquer |
| `GET` | `/v1/subscriptions/:subscription_id` | Buscar assinatura | qualquer |
| `POST` | `/v1/subscriptions/:subscription_id/cancel` | Cancelar assinatura | member |
| `POST` | `/v1/subscriptions/:subscription_id/pause` | Pausar assinatura | member |
| `POST` | `/v1/subscriptions/:subscription_id/resume` | Retomar assinatura | member |
| `PUT` | `/v1/subscriptions/:subscription_id/items` | Substituir itens | admin |
| `GET` | `/v1/subscriptions/:subscription_id/items` | Listar itens | qualquer |
| `GET` | `/v1/subscriptions/:subscription_id/history` | Histórico de status | admin |
| `POST` | `/v1/subscriptions/:subscription_id/schedules` | Agendar mudança | admin |
| `DELETE` | `/v1/subscriptions/:subscription_id/schedules/pending` | Cancelar agendamento | admin |

**POST /v1/subscriptions — Requisição:**
```json
{
  "customer_id": "uuid",
  "trial_end": "2026-04-20T00:00:00Z",
  "metadata": { "source": "checkout" },
  "items": [
    { "plan_id": "uuid", "quantity": 1 }
  ]
}
```

`trial_end` é opcional. Se fornecido e > `NOW()`, status inicial = `trialing`; caso contrário, `incomplete`.

**Resposta (201 Created):**
```json
{
  "id": "uuid",
  "organization_id": "uuid",
  "customer_id": "uuid",
  "status": "trialing",
  "current_period_start": "2026-04-13T00:00:00Z",
  "current_period_end": "2026-05-13T00:00:00Z",
  "cancel_at_period_end": false,
  "metadata": { "source": "checkout" },
  "items": [
    {
      "id": "uuid",
      "plan_id": "uuid",
      "quantity": 1,
      "created_at": "2026-04-13T10:00:00Z"
    }
  ],
  "created_at": "2026-04-13T10:00:00Z",
  "updated_at": "2026-04-13T10:00:00Z"
}
```

**POST /v1/subscriptions/:id/cancel — Requisição:**
```json
{
  "reason": "customer request",
  "cancel_at_period_end": false
}
```

Se `cancel_at_period_end = true`, a subscription permanece `active` e será cancelada ao fim do período. Se `false`, o cancelamento é imediato.

**POST /v1/subscriptions/:id/schedules — Requisição:**
```json
{
  "scheduled_at": "2026-05-01T00:00:00Z",
  "target_status": "canceled",
  "target_plan_id": null
}
```

Exatamente um de `target_status` ou `target_plan_id` deve ser fornecido.

**GET /v1/subscriptions — Parâmetros de query:**

| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `limit` | int | Padrão: 20, máximo: 100 |
| `offset` | int | Padrão: 0 |
| `customer_id` | UUID | Filtrar por cliente |
| `status` | string (repetível) | `?status=active&status=past_due` |
| `from` | RFC3339 | Início do intervalo de criação |
| `to` | RFC3339 | Fim do intervalo de criação |

**Erros esperados:**

| Status | Cenário |
|--------|---------|
| 404 | subscription_id ou plan_id não encontrado |
| 409 | Transição de status inválida pela FSM |
| 409 | Schedule pendente ao tentar criar novo schedule |
| 422 | Tentar cancelar assinatura já cancelada |
| 422 | Items vazio ou plan inativo |

---

## Ordem de Implementação Recomendada

1. **Plans** — menor escopo, sem dependências novas além de `Product`
2. **Subscriptions** — depende de Plans, Customers e Products já implementados

Seguir o fluxo padrão do projeto para cada módulo:

```
domain/{feature}.go
    ↓
database/queries/{feature}.sql  →  make sqlc
    ↓
database/repository/{feature}.go
    ↓
service/{feature}.go
    ↓
api/handler/{feature}.go
    ↓
api/router/router.go  (registrar handler)
```
