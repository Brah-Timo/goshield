# ── Stage 1: builder ────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

# Copy go.mod / go.sum from monorepo root
COPY go.mod go.sum ./
RUN go mod download

# Copy shared packages
COPY pkg/ ./pkg/

# Copy service source
COPY services/notification-service/ ./services/notification-service/

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags=-static" \
    -o /notification-service \
    ./services/notification-service/cmd/...

# ── Stage 2: distroless runtime ─────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /notification-service /notification-service
COPY --from=builder /app/services/notification-service/config/config.yaml /config/config.yaml

USER nonroot:nonroot
EXPOSE 8083

ENTRYPOINT ["/notification-service"]
