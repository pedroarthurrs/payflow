# Comandos

## Make

Todos os comandos estão no [Makefile](../Makefile). Para ver a lista completa com descrições:

```bash
make help
```

---

## Docker

### `make dev`

Sobe os containers de desenvolvimento com hot-reload (Air).

```bash
make dev
```

Inicializa:
- PostgreSQL em `localhost:5433`
- API em `http://localhost:8080`

### `make build-dev`

Força o rebuild das imagens Docker.

```bash
make build-dev
```

Use quando mudar o Dockerfile ou dependências.

### `make down`

Para os containers sem removê-los.

```bash
make down
```

### `make remove`

Para e remove containers, volumes e networks.

```bash
make remove
```

Use quando quiser limpar completamente e recomeçar do zero.

---

## Banco de Dados

### `make migrate-up`

Aplica todas as migrations pendentes.

```bash
make migrate-up
```

Requer PostgreSQL rodando e `DATABASE_URL` configurada.

### `make migrate-down`

Reverte a última migration aplicada.

```bash
make migrate-down
```

### `make migrate-create`

Cria um novo par de arquivos de migration.

```bash
make migrate-create name=add_products_table
```

Gera:
- `deploy/migrations/000004_add_products_table.up.sql`
- `deploy/migrations/000004_add_products_table.down.sql`

### `make sqlc`

Gera código Go a partir das queries SQL em `internal/database/queries.sql`.

```bash
make sqlc
```

Atualiza `internal/database/dbgen/*.go`. Execute sempre que escrever uma nova query ou alterar o schema.

---

## Go

### `make tidy`

Limpa dependências e atualiza `go.mod` e `go.sum`.

```bash
make tidy
```

### `make test`

Roda todos os testes com saída verbose.

```bash
make test
```

### `make vet`

Roda o `go vet` no projeto.

```bash
make vet
```

### `make lint`

Roda o `golangci-lint` no projeto.

```bash
make lint
```

### `make fmt`

Roda o `gofmt` no projeto.

```bash
make fmt
```

---

## Qualidade

### `make check`

Roda `fmt`, `tidy`, `vet`, `lint` e `test` em sequência. Recomendado antes de commits.

```bash
make check
```

### `make setup-hooks`

Configura o Git para usar os hooks do projeto (`.githooks/`).

```bash
make setup-hooks
```

Execute uma vez após clonar o repositório.

---

## Comandos Go Diretos

### Compilar sem executar
```bash
go build ./cmd/server/main.go
```

### Executar sem Docker
```bash
go run ./cmd/server/main.go
```

### Atualizar dependência específica
```bash
go get -u github.com/gin-gonic/gin
go mod tidy
```

### Árvore de dependências
```bash
go mod graph
```

### Testes com cobertura
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Detectar race conditions
```bash
go test -race ./...
```

---

## Debugging

### Delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server/main.go
```

### Logs estruturados

A aplicação usa `log/slog` nativo do Go 1.21+. Em desenvolvimento os logs saem em texto; em produção saem em JSON. O logger é inicializado em `cmd/server/main.go` via `SetupLogger(env)`.

---

## Fluxo Completo de Desenvolvimento

```bash
# 1. Suba o stack
make dev

# 2. Em outro terminal, crie a migration
make migrate-create name=add_products_table

# 3. Edite migrations/000004_add_products_table.up.sql

# 4. Aplique a migration
make migrate-up

# 5. Escreva a query em internal/database/queries.sql

# 6. Gere o código Go
make sqlc

# 7. Implemente service e handler

# 8. Rode os checks de qualidade
make check

# 9. Commit
git add .
git commit -m "feat: add products feature"

# 10. Limpe o ambiente
make remove
```

---

## Integração com CI/CD

```bash
# Lint
golangci-lint run ./...

# Testes com cobertura
go test -v -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Build
go build -o build/server ./cmd/server
```

---

## Troubleshooting

### `make: command not found`

```bash
# Linux
sudo apt install make

# macOS
brew install make
```

### `migrate: command not found`

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### `sqlc: command not found`

```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### Docker recusando conexão

```bash
# Adicione seu usuário ao grupo docker (Linux)
sudo usermod -aG docker $USER
newgrp docker
```

---

## Referências

- [SETUP.md](SETUP.md) — configuração do ambiente
- [ARCHITECTURE.md](ARCHITECTURE.md) — estrutura e fluxo
- [API.md](API.md) — rotas disponíveis
- [DEVELOPMENT.md](DEVELOPMENT.md) — guia de contribuição
