# ── Stage 1: builder ────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY pkg/ ./pkg/
COPY services/api-gateway/ ./services/api-gateway/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -extldflags=-static" \
    -o /api-gateway \
    ./services/api-gateway/cmd/...

# ── Stage 2: distroless runtime ─────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /api-gateway /api-gateway
COPY --from=builder /app/services/api-gateway/config/config.yaml /config/config.yaml

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/api-gateway"]
