# ── Stage 1: Builder ─────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go module files from repo root (monorepo: single go.mod at root)
COPY go.mod go.sum ./
RUN go mod download

# Copy full source
COPY . .

# Build the claim-service binary with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev)" \
    -o /bin/claim-service \
    ./services/claim-service/cmd/main.go

# ── Stage 2: Distroless runtime ──────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binary and CA certs
COPY --from=builder /bin/claim-service /claim-service
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy default config (overridden by ConfigMap in Kubernetes)
COPY --from=builder /build/services/claim-service/config/config.yaml /config/config.yaml

ENV CONFIG_PATH=/config/config.yaml
ENV TZ=UTC

EXPOSE 8082

USER nonroot:nonroot

ENTRYPOINT ["/claim-service"]
