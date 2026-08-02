// Package main is the entry point for the ai-service-go.
// It consumes claims.new from Kafka, calls the Python AI service, and publishes results.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/events"
	"github.com/goshield/pkg/logger"
	"github.com/goshield/pkg/telemetry"
	"github.com/goshield/services/ai-service-go/internal/bridge"
	"github.com/goshield/services/ai-service-go/internal/consumer"
	"github.com/goshield/services/ai-service-go/internal/handler"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ai-service-go fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		// When run via `go run ./services/ai-service-go/cmd/...` from the repo root,
		// the working directory is the repo root, not the service directory.
		// Try both locations so the service works from either CWD.
		for _, p := range []string{
			"config/config.yaml",
			"services/ai-service-go/config/config.yaml",
		} {
			if _, e := os.Stat(p); e == nil {
				cfgPath = p
				break
			}
		}
		if cfgPath == "" {
			cfgPath = "config/config.yaml" // let config.Load produce a clear error
		}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(logger.Config{
		Level:       "info",
		Development: cfg.Service.Environment == "development",
		ServiceName: cfg.Service.Name,
	})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting ai-service-go", zap.String("env", cfg.Service.Environment))

	ctx := context.Background()
	tel, err := telemetry.Setup(ctx, cfg.Telemetry, log)
	if err != nil {
		log.Warn("telemetry setup failed (non-fatal)", zap.Error(err))
	} else {
		defer tel.Shutdown(ctx) //nolint:errcheck
	}

	// Python AI service URL (stored in grpc.ai_service_addr for flexibility)
	pythonAIURL := cfg.GRPC.AIServiceAddr
	if pythonAIURL == "" {
		pythonAIURL = "http://ai-service-py:8090"
	}

	aiClient := bridge.NewPythonAIClient(pythonAIURL, log)

	// Wait for Python AI service to be ready (retry up to 60s)
	log.Info("waiting for Python AI service", zap.String("url", pythonAIURL))
	waitCtx, waitCancel := context.WithTimeout(ctx, 60*time.Second)
	defer waitCancel()
	for {
		if err := aiClient.HealthCheck(waitCtx); err == nil {
			log.Info("Python AI service is ready")
			break
		}
		select {
		case <-waitCtx.Done():
			log.Warn("Python AI service not ready after 60s — continuing anyway")
			goto startConsumer
		case <-time.After(3 * time.Second):
		}
	}
startConsumer:

	// Kafka producer
	producer := events.NewProducer(cfg.Kafka.Brokers, log)
	defer producer.Close() //nolint:errcheck

	// Claim consumer
	claimConsumer := consumer.New(
		aiClient,
		producer,
		cfg.Kafka.TopicClaimsAnalyzed,
		cfg.Kafka.TopicClaimsFlagged,
		cfg.Kafka.TopicClaimsFailed,
		log,
	)

	kafkaReader := events.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicClaimsNew,
		cfg.Kafka.GroupID,
		log,
	)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- kafkaReader.Consume(consumerCtx, claimConsumer.Handle)
	}()

	// HTTP health server
	healthHandler := handler.NewHealthHandler(aiClient)
	app := fiber.New(fiber.Config{AppName: "ai-service-go"})
	app.Get("/health", healthHandler.Live)
	app.Get("/readyz", healthHandler.Ready)
	app.Get("/metrics", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("# ai-service-go metrics placeholder")
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverDone := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Service.Port)
		log.Info("HTTP server listening", zap.String("addr", addr))
		serverDone <- app.Listen(addr)
	}()

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", zap.String("signal", sig.String()))
	case err := <-serverDone:
		if err != nil {
			log.Error("HTTP server error", zap.Error(err))
		}
	}

	cancelConsumer()
	if err := kafkaReader.Close(); err != nil {
		log.Warn("kafka reader close error", zap.Error(err))
	}
	<-consumerDone

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("ai-service-go stopped gracefully")
	return nil
}
