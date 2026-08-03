# GoShield — Production-Ready Distributed Insurance Fraud Detection Platform

> Go 1.22 · Python 3.11 · React 18 · PostgreSQL 16 · Redis 7 · Kafka · MinIO/Azure Blob · XGBoost · SHAP · Kubernetes · Terraform (Azure AKS)

---

## Architecture

```
Browser (React 18)
      │
      ▼
API Gateway :8080  ←── JWT auth, Redis rate-limit, OTel tracing, reverse proxy
      │
      ├──▶ auth-service      :8081  JWT/OAuth2 Google/Casbin RBAC/refresh rotation
      ├──▶ claim-service      :8082  CRUD, MinIO upload, Kafka producer
      ├──▶ notification-svc   :8083  WebSocket hub, SMTP, Slack Block Kit
      └──▶ ai-service-go      :8093  Kafka consumer → HTTP bridge → ai-service-py
                                             │
                                             ▼
                               ai-service-py  :8090  FastAPI + XGBoost + SHAP

Kafka topics: claims.new → claims.analyzed → claims.flagged → claims.failed (DLQ)
```

## Service Port Map

| Service | Port |
|---|---|
| api-gateway | 8080 |
| auth-service | 8081 |
| claim-service | 8082 |
| notification-service | 8083 |
| ai-service-py | 8090 |
| ai-service-go | 8093 |

## Quick Start (Local)

### Prerequisites
- Docker + Docker Compose
- Go 1.22
- Node 20
- Python 3.11

### 1. Start infrastructure
```bash
cd goshield
docker compose up -d   # PostgreSQL, Redis, Kafka, MinIO, Jaeger, Prometheus, Grafana
```

### 2. Run migrations
```bash
for f in migrations/*.sql; do psql "$DATABASE_URL" -f "$f"; done
```

### 3. Start Go services
```bash
# Each in its own terminal:
go run ./services/auth-service/cmd/...
go run ./services/claim-service/cmd/...
go run ./services/notification-service/cmd/...
go run ./services/ai-service-go/cmd/...
go run ./services/api-gateway/cmd/...
```

### 4. Start Python AI service
```bash
cd services/ai-service-py
pip install -r requirements.txt
uvicorn app.main:app --host 0.0.0.0 --port 8090 --reload
```

### 5. Start frontend
```bash
cd frontend
npm install
npm run dev   # http://localhost:5173
```

### Demo credentials (from migration 002)
| Email | Password | Role |
|---|---|---|
| admin@goshield.io | admin123 | ADMIN |
| analyst@goshield.io | analyst123 | ANALYST |

---

## Fraud Detection Pipeline

```
1. Analyst uploads claim (PDF) via React UI
2. claim-service stores claim in PostgreSQL + uploads PDF to MinIO
3. claim-service publishes claim.created → Kafka claims.new topic
4. ai-service-go consumes claims.new
5. ai-service-go calls ai-service-py POST /api/v1/analyze
6. ai-service-py runs XGBoost inference on 10 engineered features
7. SHAP TreeExplainer computes per-feature fraud explanations
8. ai-service-go publishes claim.analyzed → Kafka claims.analyzed topic
9. If fraud_score ≥ 0.80 → also publishes claim.flagged
10. claim-service consumes claim.analyzed → updates PostgreSQL status
11. notification-service:
    - Broadcasts real-time WebSocket message to browser
    - Sends SMTP fraud alert email
    - Posts Slack Block Kit alert
12. React Dashboard updates live via WebSocket
```

## XGBoost Features (10)

| Feature | Description |
|---|---|
| amount_log | log1p(claim amount) |
| account_age_days | days since account creation |
| prior_claims_count | number of prior claims |
| claim_type_enc | label-encoded claim type |
| amount_per_day | amount / account_age_days |
| is_new_account | account < 90 days |
| high_amount_flag | amount > $50,000 |
| repeat_claimant_flag | prior_claims > 3 |
| days_since_incident | days between incident and filing |
| policy_hash_mod | policy number hash mod 100 |

## Risk Levels

| Score | Level |
|---|---|
| < 0.40 | LOW |
| 0.40–0.69 | MEDIUM |
| 0.70–0.89 | HIGH |
| ≥ 0.90 | CRITICAL |

## Security

- JWT (15min access + 7-day refresh) with SHA-256 hash storage and token rotation
- HttpOnly Secure SameSite=Strict refresh token cookies
- Casbin v2 RBAC: ADMIN > ANALYST > VIEWER hierarchy
- In-memory access token storage in frontend (never localStorage)
- Silent 401→refresh in Axios interceptor
- Distroless (`gcr.io/distroless/static-debian12:nonroot`) Docker images for all Go services
- Redis sliding-window rate limiter (Lua, atomic, fail-open)
- Multi-tenant data isolation: every query scoped by company_id

## API Reference (Gateway entry points)

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | /auth/v1/register | Public | Register new user |
| POST | /auth/v1/login | Public | Login, get tokens |
| POST | /auth/v1/refresh | Cookie | Refresh access token |
| POST | /auth/v1/logout | JWT | Revoke refresh token |
| GET | /auth/v1/oauth/google/login | Public | Google OAuth redirect |
| GET | /claims/v1/claims | JWT | List claims (paginated) |
| POST | /claims/v1/claims | JWT | Create claim |
| GET | /claims/v1/claims/:id | JWT | Get claim detail |
| POST | /claims/v1/claims/:id/document | JWT | Upload PDF |
| POST | /claims/v1/claims/:id/review | ANALYST+ | Review claim |
| GET | /claims/v1/stats | JWT | Dashboard statistics |
| POST | /ai/v1/analyze | JWT | Manual AI analysis |
| GET | /ws?token=... | JWT(query) | WebSocket stream |
| GET | /health | Public | Gateway liveness |
| GET | /metrics | Public | Prometheus metrics |

## Kubernetes Deployment

```bash
# Apply base manifests
kubectl apply -k k8s/base/

# Check rollout
kubectl rollout status deployment -n goshield

# Get gateway IP
kubectl get svc api-gateway-svc -n goshield
```

## Terraform (Azure AKS)

```bash
cd infra/terraform
terraform init
terraform workspace new prod
terraform apply -var-file=prod.tfvars
```

Resources provisioned:
- AKS cluster (3-node system pool + 1–4 AI workload pool, multi-zone)
- Azure Container Registry (AcrPull role assigned to AKS kubelet)
- Azure Database for PostgreSQL Flexible Server 16
- Azure Cache for Redis
- Azure Blob Storage (via MinIO-compatible SDK)

## CI/CD (GitHub Actions)

| Workflow | Trigger | Jobs |
|---|---|---|
| `ci-goshield.yml` | push/PR to main | Go lint+test, Python pytest, frontend tsc+build, Docker build+push to GHCR |
| `cd-goshield.yml` | CI success on main | kubectl set image, rollout wait, smoke test |

Required GitHub secrets: `AZURE_CREDENTIALS`, `AKS_RESOURCE_GROUP`, `AKS_CLUSTER_NAME`

## Monitoring

| Stack | URL |
|---|---|
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 (admin/admin) |
| Jaeger | http://localhost:16686 |

Grafana dashboards (import from `monitoring/grafana/dashboards/`):
- `overview.json` — Request rate, error rate, P99 latency, pod restarts
- `fraud.json` — Fraud detection rate, average score, risk level distribution, AI latency

## Project Structure

```
goshield/
├── go.mod                          # Single monorepo module: github.com/goshield
├── migrations/                     # SQL migrations (goose)
│   ├── 001_initial_schema.sql
│   ├── 002_seed_demo_data.sql
│   └── 003_auth_service_schema.sql
├── pkg/                            # Shared packages
│   ├── config/         AppConfig, JWT, RateLimit, Logger structs
│   ├── database/       pgxpool wrapper
│   ├── events/         Kafka producer + consumer
│   ├── logger/         Zap structured logger
│   ├── middleware/     JWT manager, Fiber JWT middleware, Casbin
│   ├── storage/        MinIO/Azure Blob client
│   └── telemetry/      OpenTelemetry setup (Jaeger OTLP)
├── proto/                          # Protobuf definitions
├── services/
│   ├── auth-service/               :8081  Go/Fiber
│   ├── claim-service/              :8082  Go/Fiber
│   ├── notification-service/       :8083  Go/Fiber + gofiber/websocket
│   ├── ai-service-py/              :8090  Python FastAPI
│   ├── ai-service-go/              :8093  Go/Fiber (Kafka→Python bridge)
│   └── api-gateway/                :8080  Go/Fiber (reverse proxy)
├── frontend/                       React 18 + TypeScript + Vite + Tailwind
├── k8s/base/                       Kubernetes manifests (Deployment/Service/HPA)
├── infra/terraform/                Azure AKS + PostgreSQL + Redis + ACR
├── monitoring/grafana/dashboards/  Grafana JSON dashboards
└── .github/workflows/              CI (test+build+push) + CD (AKS deploy)
```
