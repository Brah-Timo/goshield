# ─── Stage 1: Builder ─────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy dependency files first for layer caching
COPY go.mod go.sum ./

# Download modules
RUN go mod download

# Copy source code
COPY . .

# Build the binary — static, stripped, CGO disabled
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o goerp-server \
    ./cmd/server/

# ─── Stage 2: Final Image ─────────────────────────────────────────────────────
FROM alpine:3.19

# Security: run as non-root user
RUN addgroup -S goerp && adduser -S goerp -G goerp

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/goerp-server ./goerp-server

# Copy the web frontend
COPY --from=builder /build/web ./web

# Set ownership
RUN chown -R goerp:goerp /app

# Switch to non-root user
USER goerp

# Expose the application port
EXPOSE 8080

# Health check — polls /health every 30s
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the server
ENTRYPOINT ["/app/goerp-server"]
