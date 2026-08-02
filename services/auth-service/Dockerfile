# ── Stage 1: Builder ─────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s" \
    -o /bin/auth-service \
    ./services/auth-service/cmd/main.go

# ── Stage 2: Distroless runtime ──────────────────────────────────────────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/auth-service /auth-service
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/services/auth-service/config/config.yaml /config/config.yaml
COPY --from=builder /build/services/auth-service/policies/ /policies/

ENV CONFIG_PATH=/config/config.yaml
ENV POLICY_DIR=/policies
ENV TZ=UTC

EXPOSE 8081

USER nonroot:nonroot

ENTRYPOINT ["/auth-service"]
