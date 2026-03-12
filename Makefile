# Carrega as variáveis do arquivo .env
include .env
export $(shell sed 's/=.*//' .env)

# Variáveis de caminhos
DOCKER_COMPOSE_DEV = docker/dev/docker-compose.yml
MIGRATIONS_PATH = migrations
DATABASE_URL = postgres://$(DB_USER):$(DB_PASSWORD)@localhost:5433/$(DB_NAME)?sslmode=disable

.PHONY: dev down build-dev sqlc migrate-up migrate-down migrate-create tidy test help

# --- DOCKER COMMANDS ---

## dev: Sobe os containers de desenvolvimento (com Air)
dev:
	docker compose -f $(DOCKER_COMPOSE_DEV) up

## down: Para os containers
down:
	docker compose -f $(DOCKER_COMPOSE_DEV) down

## remove: Remove os containers
remove:
	docker compose -f $(DOCKER_COMPOSE_DEV) down -v --remove-orphans

## build-dev: Força o build das imagens de dev
build-dev:
	docker compose -f $(DOCKER_COMPOSE_DEV) up --build

# --- SQLC & GO COMMANDS ---

## sqlc: Gera o código Go a partir das queries SQL
sqlc:
	sqlc generate

## tidy: Limpa e atualiza as dependências do Go
tidy:
	go mod tidy

## test: Roda os testes do projeto
test:
	go test -v ./...

# --- MIGRATIONS COMMANDS ---

## migrate-up: Aplica todas as migrations no banco de dados
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

## migrate-down: Reverte a última migration aplicada
migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

## migrate-create name=exemplo: Cria um novo par de arquivos de migration
migrate-create:
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(name)

# --- HELP ---

## help: Mostra esta ajuda
help:
	@echo "Comandos disponíveis:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' |  sed -e 's/^/ /'