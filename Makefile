# GoShield Makefile
# Usage: make <target>

SHELL := /bin/bash
.DEFAULT_GOAL := help

# Project settings
PROJECT := goshield
MODULE  := github.com/goshield
SERVICES := api-gateway auth-service claim-service ai-service-go notification
GO      := go
GOFLAGS := -trimpath

# Versioning
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_SHA  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TS := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
  -X $(MODULE)/pkg/config.Version=$(VERSION) \
  -X $(MODULE)/pkg/config.GitSHA=$(GIT_SHA) \
  -X $(MODULE)/pkg/config.BuildTime=$(BUILD_TS)

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' | sort

# ── Development ───────────────────────────────────────────────────────────────

.PHONY: run
run: ## Start all services with Docker Compose
	@echo "▶  Starting GoShield..."
	docker compose up -d
	@echo "✅ Services running. Dashboard: http://localhost:3000  API: http://localhost:8080"

.PHONY: run-infra
run-infra: ## Start only infrastructure (DB, Redis, Kafka, MinIO)
	docker compose up -d postgres redis kafka minio jaeger

.PHONY: down
down: ## Stop all services
	docker compose down

.PHONY: down-clean
down-clean: ## Stop all services and remove volumes
	docker compose down -v --remove-orphans

.PHONY: logs
logs: ## Tail logs from all services
	docker compose logs -f --tail=100

.PHONY: logs-svc
logs-svc: ## Tail logs from a specific service: make logs-svc SVC=claim-service
	docker compose logs -f --tail=100 $(SVC)

.PHONY: ps
ps: ## Show service status
	docker compose ps

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build all Go services
	@for svc in $(SERVICES); do \
	  echo "▶  Building $$svc..."; \
	  CGO_ENABLED=0 GOOS=linux $(GO) build $(GOFLAGS) \
	    -ldflags="$(LDFLAGS)" \
	    -o bin/$$svc \
	    ./services/$$svc/cmd/... || exit 1; \
	done
	@echo "✅ All services built → bin/"

.PHONY: build-svc
build-svc: ## Build a single service: make build-svc SVC=claim-service
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
	  -o bin/$(SVC) ./services/$(SVC)/cmd/...

.PHONY: docker-build
docker-build: ## Build all Docker images
	@for svc in $(SERVICES); do \
	  echo "▶  Building Docker image for $$svc..."; \
	  docker build \
	    --build-arg VERSION=$(VERSION) \
	    --build-arg GIT_SHA=$(GIT_SHA) \
	    -t $(PROJECT)/$$svc:$(VERSION) \
	    -t $(PROJECT)/$$svc:latest \
	    -f services/$$svc/Dockerfile . || exit 1; \
	done

.PHONY: docker-push
docker-push: ## Push images to registry: make docker-push REGISTRY=ghcr.io/org
	@for svc in $(SERVICES); do \
	  docker tag $(PROJECT)/$$svc:$(VERSION) $(REGISTRY)/$$svc:$(VERSION); \
	  docker push $(REGISTRY)/$$svc:$(VERSION); \
	done

# ── Database ──────────────────────────────────────────────────────────────────

.PHONY: migrate
migrate: ## Apply all pending migrations
	docker compose run --rm migrate

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	docker run --rm \
	  --network goshield_goshield-net \
	  -e GOOSE_DRIVER=postgres \
	  -e "GOOSE_DBSTRING=postgres://goshield:goshield_dev_pass@postgres:5432/goshield?sslmode=disable" \
	  -v $(PWD)/migrations:/migrations \
	  pressly/goose -dir /migrations down

.PHONY: migrate-status
migrate-status: ## Show migration status
	docker run --rm \
	  --network goshield_goshield-net \
	  -e GOOSE_DRIVER=postgres \
	  -e "GOOSE_DBSTRING=postgres://goshield:goshield_dev_pass@postgres:5432/goshield?sslmode=disable" \
	  -v $(PWD)/migrations:/migrations \
	  pressly/goose -dir /migrations status

.PHONY: psql
psql: ## Open psql shell
	docker exec -it goshield-postgres psql -U goshield -d goshield

# ── Protobuf ──────────────────────────────────────────────────────────────────

.PHONY: proto
proto: ## Generate gRPC code from .proto files
	@command -v protoc >/dev/null || (echo "❌ protoc not found. Run: make install-tools"; exit 1)
	@mkdir -p gen
	@for svc in claim notification auth; do \
	  protoc \
	    --go_out=gen --go_opt=paths=source_relative \
	    --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
	    proto/$$svc/$$svc.proto || exit 1; \
	done
	@echo "✅ gRPC code generated → gen/"

# ── Tests ─────────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run all tests with coverage
	$(GO) test -v -race -count=1 \
	  -coverprofile=coverage.out \
	  -covermode=atomic \
	  ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report → coverage.html"

.PHONY: test-unit
test-unit: ## Run unit tests only (no integration)
	$(GO) test -v -short -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires running infra)
	$(GO) test -v -run Integration -count=1 ./...

.PHONY: bench
bench: ## Run benchmarks
	$(GO) test -bench=. -benchmem ./...

# ── Code Quality ──────────────────────────────────────────────────────────────

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run --timeout=5m ./...

.PHONY: fmt
fmt: ## Format Go code
	$(GO) fmt ./...
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Tidy go modules
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: ci
ci: tidy vet lint test ## Run full CI pipeline locally

# ── AI Model ─────────────────────────────────────────────────────────────────

.PHONY: train
train: ## Train the fraud detection model
	@echo "▶  Training XGBoost model..."
	cd ai/training && python3 train.py
	@echo "✅ Model saved → ai/models/fraud_model_v1.pkl"

.PHONY: ai-shell
ai-shell: ## Open Python shell in AI service container
	docker exec -it goshield-ai-py bash

# ── Deployment ────────────────────────────────────────────────────────────────

.PHONY: deploy-staging
deploy-staging: ## Deploy to staging via Helm
	helm upgrade --install goshield-staging helm/goshield \
	  --namespace goshield-staging \
	  --create-namespace \
	  --values k8s/values-staging.yaml \
	  --set global.imageTag=$(VERSION) \
	  --atomic --timeout=10m

.PHONY: deploy-prod
deploy-prod: ## Deploy to production via Helm
	@echo "⚠️  Deploying to PRODUCTION — are you sure? [y/N]" && read ans && [ "$$ans" = y ]
	helm upgrade --install goshield helm/goshield \
	  --namespace goshield \
	  --create-namespace \
	  --values k8s/values-production.yaml \
	  --set global.imageTag=$(VERSION) \
	  --atomic --timeout=15m

.PHONY: k8s-apply
k8s-apply: ## Apply raw Kubernetes manifests
	kubectl apply -k k8s/

# ── Tools ─────────────────────────────────────────────────────────────────────

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "▶  Installing Go tools..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.59.1
	@echo "▶  Installing Python tools..."
	pip3 install -r services/ai-service-py/requirements.txt
	@echo "✅ Tools installed"

.PHONY: version
version: ## Print version info
	@echo "Version:  $(VERSION)"
	@echo "Git SHA:  $(GIT_SHA)"
	@echo "Built at: $(BUILD_TS)"

# ── URLs (dev) ────────────────────────────────────────────────────────────────

.PHONY: urls
urls: ## Print all local service URLs
	@echo ""
	@echo "  🌐  Frontend Dashboard:  http://localhost:3000"
	@echo "  🔀  API Gateway:         http://localhost:8080"
	@echo "  📬  MailHog (email):     http://localhost:8025"
	@echo "  📦  MinIO Console:       http://localhost:9001  (minioadmin/minioadmin123)"
	@echo "  📊  Grafana:             http://localhost:3001  (admin/admin123)"
	@echo "  🔥  Prometheus:          http://localhost:9090"
	@echo "  🔍  Jaeger (traces):     http://localhost:16686"
	@echo "  📨  Kafka UI:            http://localhost:8090"
	@echo "  🐘  PostgreSQL:          localhost:5432         (goshield/goshield_dev_pass)"
	@echo ""
