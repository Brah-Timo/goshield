// Package main is the entry point for the claim-service.
// It wires all dependencies, starts the HTTP server and Kafka consumer.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberreqid "github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/database"
	"github.com/goshield/pkg/events"
	"github.com/goshield/pkg/logger"
	"github.com/goshield/pkg/middleware"
	"github.com/goshield/pkg/storage"
	"github.com/goshield/pkg/telemetry"

	handler "github.com/goshield/services/claim-service/internal/handler"
	repository "github.com/goshield/services/claim-service/internal/repository"
	svc "github.com/goshield/services/claim-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "claim-service fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ── Config ────────────────────────────────────────────────────────────────
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		// Support running from repo root (`go run ./services/claim-service/cmd/...`)
		// as well as from the service directory.
		for _, p := range []string{
			"config/config.yaml",
			"services/claim-service/config/config.yaml",
		} {
			if _, e := os.Stat(p); e == nil {
				cfgPath = p
				break
			}
		}
		if cfgPath == "" {
			cfgPath = "config/config.yaml"
		}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	log, err := logger.New(logger.Config{
		Level:       "info",
		Development: cfg.Service.Environment == "development",
		ServiceName: cfg.Service.Name,
	})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting claim-service",
		zap.String("version", cfg.Service.Version),
		zap.String("env", cfg.Service.Environment),
	)

	// ── Telemetry ─────────────────────────────────────────────────────────────
	ctx := context.Background()
	tel, err := telemetry.Setup(ctx, cfg.Telemetry, log)
	if err != nil {
		log.Warn("telemetry setup failed (non-fatal)", zap.Error(err))
	} else {
		defer tel.Shutdown(ctx) //nolint:errcheck
	}

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := database.Connect(ctx, cfg.Database, log)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer db.Close()

	// ── Storage ───────────────────────────────────────────────────────────────
	store, err := storage.NewClient(cfg.Storage, log)
	if err != nil {
		return fmt.Errorf("connect to storage: %w", err)
	}

	// ── Kafka Producer ────────────────────────────────────────────────────────
	producer := events.NewProducer(cfg.Kafka.Brokers, log)
	defer producer.Close() //nolint:errcheck

	// ── Repository + Service ──────────────────────────────────────────────────
	repo := repository.New(db, log)
	claimSvc := svc.New(repo, store, producer, cfg, log)

	// ── HTTP Handler ──────────────────────────────────────────────────────────
	httpHandler := handler.New(claimSvc, log)
	kafkaHandler := handler.NewKafkaHandler(claimSvc, log)

	// ── JWT Manager ───────────────────────────────────────────────────────────
	jwtMgr := middleware.NewJWTManager(middleware.JWTConfig{
		Secret:        cfg.Auth.JWTSecret,
		AccessExpiry:  cfg.Auth.JWTAccessExpiry,
		RefreshExpiry: cfg.Auth.JWTRefreshExpiry,
	})

	// ── Fiber App ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:           "GoShield claim-service",
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		BodyLimit:         25 * 1024 * 1024, // 25 MB
		EnablePrintRoutes: cfg.Service.Environment == "development",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error(), "code": code})
		},
	})

	app.Use(recover.New())
	app.Use(fiberreqid.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET,POST,PATCH,DELETE,OPTIONS",
	}))

	// ── Health endpoints (public) ─────────────────────────────────────────────
	app.Get("/health", handler.Health)

	// /livez — lightweight: just confirm the process is alive
	app.Get("/livez", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "alive"})
	})

	// /readyz — deep: check DB + Kafka connectivity
	app.Get("/readyz", func(c *fiber.Ctx) error {
		type checkResult struct {
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		dbCheck    := checkResult{Status: "ok"}
		kafkaCheck := checkResult{Status: "ok"}

		if err := db.HealthCheck(c.Context()); err != nil {
			dbCheck = checkResult{Status: "unhealthy", Error: err.Error()}
		}
		// Kafka: attempt a quick dial to the first broker
		if len(cfg.Kafka.Brokers) > 0 {
			if err := events.PingBroker(cfg.Kafka.Brokers[0]); err != nil {
				kafkaCheck = checkResult{Status: "unhealthy", Error: err.Error()}
			}
		}

		overall := "ready"
		httpCode := 200
		if dbCheck.Status != "ok" || kafkaCheck.Status != "ok" {
			overall  = "degraded"
			httpCode = 503
		}

		return c.Status(httpCode).JSON(fiber.Map{
			"status":    overall,
			"service":   "claim-service",
			"timestamp": time.Now().UTC(),
			"checks": fiber.Map{
				"database": dbCheck,
				"kafka":    kafkaCheck,
			},
		})
	})

	// /metrics — Prometheus scrape endpoint
	app.Get("/metrics", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; version=0.0.4")
		return c.SendString("# GoShield claim-service metrics\n")
	})

	// Protected API routes — prefix matches frontend proxy: /claims/v1/claims/...
	api := app.Group("/claims/v1", middleware.JWTMiddleware(jwtMgr))
	httpHandler.RegisterRoutes(api)

	// ── Kafka Consumer ────────────────────────────────────────────────────────
	consumer := events.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicClaimsAnalyzed,
		cfg.Kafka.GroupID,
		log,
	)

	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Consume(consumerCtx, kafkaHandler.HandleEvent)
	}()

	// ── Graceful Shutdown ─────────────────────────────────────────────────────
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

	// Stop consumer
	cancelConsumer()
	if err := consumer.Close(); err != nil {
		log.Warn("kafka consumer close error", zap.Error(err))
	}
	<-consumerDone

	// Shutdown HTTP
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("claim-service stopped gracefully")
	return nil
}
