# ─────────────────────────────────────────────────────────────────────────────
# GoERP Makefile
# Usage: make <target>
# ─────────────────────────────────────────────────────────────────────────────

APP_NAME    := goerp-server
BUILD_DIR   := .
CMD_DIR     := ./cmd/server
GO          := go
GOFLAGS     := CGO_ENABLED=0 GOOS=linux GOARCH=amd64
LDFLAGS     := -ldflags="-w -s"

.PHONY: all build run dev clean test docker-build docker-up docker-down docker-logs docker-clean help

# Default target
all: build

## build: Compile the Go binary
build:
	@echo "Building $(APP_NAME)..."
	$(GOFLAGS) $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

## run: Build and run the server
run: build
	./$(APP_NAME)

## dev: Run with live reload (requires 'air' — go install github.com/cosmtrek/air@latest)
dev:
	air -c .air.toml || $(GO) run $(CMD_DIR)

## test: Run all tests
test:
	$(GO) test ./... -v -race -timeout 30s

## test-cover: Run tests with coverage report
test-cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## tidy: Tidy go modules
tidy:
	$(GO) mod tidy

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(APP_NAME) coverage.out coverage.html
	$(GO) clean -cache

# ─── Docker targets ───────────────────────────────────────────────────────────

## docker-build: Build the Docker image
docker-build:
	docker build -t goerp:latest .

## docker-up: Start all services (GoERP + PostgreSQL + Redis)
docker-up:
	docker compose up -d
	@echo "Services started. Access GoERP at http://localhost:8080"
	@echo "Default login: admin@goerp.io / admin123"

## docker-up-dev: Start with dev profile (includes Adminer)
docker-up-dev:
	docker compose --profile dev up -d
	@echo "Services started."
	@echo "GoERP:    http://localhost:8080"
	@echo "Adminer:  http://localhost:8081"

## docker-down: Stop all services
docker-down:
	docker compose down

## docker-down-v: Stop all services and remove volumes (WARNING: deletes data)
docker-down-v:
	docker compose down -v

## docker-logs: Follow application logs
docker-logs:
	docker compose logs -f app

## docker-logs-db: Follow database logs
docker-logs-db:
	docker compose logs -f db

## docker-clean: Remove containers, images, and volumes
docker-clean:
	docker compose down -v --rmi local

## docker-shell: Open shell in the running app container
docker-shell:
	docker compose exec app /bin/sh

## docker-psql: Open psql in the database container
docker-psql:
	docker compose exec db psql -U $${DB_USER:-goerp} -d $${DB_NAME:-goerp}

# ─── Setup targets ────────────────────────────────────────────────────────────

## setup: Create .env from .env.example
setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env created from .env.example. Edit it before starting."; \
	else \
		echo ".env already exists."; \
	fi

## help: Show this help message
help:
	@echo "GoERP Build Targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
