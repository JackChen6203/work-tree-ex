.PHONY: build run-api run-worker test lint fmt \
       migrate-up migrate-down migrate-create \
       docker-up docker-down docker-build docker-logs \
       help

# ── Variables ───────────────────────────────────────────────
BINARY_API    = backend/api
BINARY_WORKER = backend/worker
MIGRATE_DIR   = backend/migrations
DB_DSN       ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# Load .env if it exists
-include .env
export

DB_HOST      ?= localhost
DB_PORT      ?= 5432
DB_USER      ?= travel
DB_PASSWORD  ?= travel
DB_NAME      ?= travel_planner
DB_SSLMODE   ?= disable

# ── Build ───────────────────────────────────────────────────
build: ## Build API and Worker binaries
	cd backend && go build -o api ./cmd/api
	cd backend && go build -o worker ./cmd/worker

# ── Run ─────────────────────────────────────────────────────
run-api: ## Run API server locally
	cd backend && go run ./cmd/api

run-worker: ## Run Worker locally
	cd backend && go run ./cmd/worker

# ── Test & Lint ─────────────────────────────────────────────
test: ## Run all Go tests
	cd backend && go test ./... -v -race -count=1

lint: ## Run golangci-lint
	cd backend && golangci-lint run ./...

fmt: ## Format Go code
	cd backend && gofmt -s -w .

# ── Migrations ──────────────────────────────────────────────
migrate-up: ## Run all pending migrations
	migrate -path $(MIGRATE_DIR) -database "$(DB_DSN)" up

migrate-down: ## Rollback last migration
	migrate -path $(MIGRATE_DIR) -database "$(DB_DSN)" down 1

migrate-down-all: ## Rollback all migrations
	migrate -path $(MIGRATE_DIR) -database "$(DB_DSN)" down -all

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_foo)
	migrate create -ext sql -dir $(MIGRATE_DIR) -seq $(NAME)

# ── Docker ──────────────────────────────────────────────────
docker-build: ## Build Docker images
	docker compose build

docker-up: ## Start all services
	docker compose up -d

docker-down: ## Stop all services
	docker compose down

docker-logs: ## Tail logs from all services
	docker compose logs -f

docker-migrate: ## Run migrations via Docker
	docker compose run --rm migrate

# ── Help ────────────────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
