FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /bin/ai-service-go \
    ./services/ai-service-go/cmd/main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /bin/ai-service-go /ai-service-go
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/services/ai-service-go/config/config.yaml /config/config.yaml
ENV CONFIG_PATH=/config/config.yaml
EXPOSE 8093
USER nonroot:nonroot
ENTRYPOINT ["/ai-service-go"]
