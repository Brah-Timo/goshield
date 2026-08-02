// Package main is the entry point for the auth-service.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberreqid "github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/database"
	"github.com/goshield/pkg/logger"
	"github.com/goshield/pkg/middleware"
	"github.com/goshield/pkg/telemetry"

	handler "github.com/goshield/services/auth-service/internal/handler"
	"github.com/goshield/services/auth-service/internal/repository"
	svc "github.com/goshield/services/auth-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "auth-service fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// ── Config ────────────────────────────────────────────────────────────────
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/config.yaml"
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

	log.Info("starting auth-service",
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

	// Ensure refresh_tokens table exists (migrations handle schema; this is a safety check).
	if err := ensureRefreshTokensTable(ctx, db); err != nil {
		log.Warn("refresh_tokens table check failed", zap.Error(err))
	}

	// ── Casbin RBAC enforcer ──────────────────────────────────────────────────
	policyDir := os.Getenv("POLICY_DIR")
	if policyDir == "" {
		policyDir = "policies"
	}
	enforcer, err := casbin.NewEnforcer(
		policyDir+"/rbac_model.conf",
		policyDir+"/rbac_policy.csv",
	)
	if err != nil {
		return fmt.Errorf("init casbin: %w", err)
	}

	// ── JWT Manager ───────────────────────────────────────────────────────────
	jwtMgr := middleware.NewJWTManager(middleware.JWTConfig{
		Secret:        cfg.Auth.JWTSecret,
		AccessExpiry:  cfg.Auth.JWTAccessExpiry,
		RefreshExpiry: cfg.Auth.JWTRefreshExpiry,
	})

	// ── Repository + Service ──────────────────────────────────────────────────
	repo := repository.New(db, log)
	authSvc := svc.New(repo, jwtMgr, enforcer, cfg, log)

	// ── HTTP Handler ──────────────────────────────────────────────────────────
	h := handler.New(authSvc, cfg, log)

	// ── Fiber App ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:      "GoShield auth-service",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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
	// AllowOrigins cannot be "*" when AllowCredentials is true (browsers reject it).
	// In development we explicitly list the Vite dev server. In production the
	// GOSHIELD_CORS_ORIGINS env var / config should list the real frontend origin.
	corsOrigins := "http://localhost:5173,http://localhost:3000"
	if cfg.Service.Environment != "development" {
		corsOrigins = "https://app.goshield.io" // override in prod config
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowCredentials: true,
	}))

	// Health endpoints (public)
	app.Get("/health", handler.Health)
	app.Get("/readyz", func(c *fiber.Ctx) error {
		if err := db.HealthCheck(c.Context()); err != nil {
			return c.Status(503).JSON(fiber.Map{"status": "unhealthy", "db": err.Error()})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	// API routes — primary prefix is /auth/v1 (matches Vite proxy + api-gateway).
	// A /api/v1 alias is kept for backwards-compat / direct service access.
	api := app.Group("/auth/v1")
	h.RegisterRoutes(api, jwtMgr)
	apiAlias := app.Group("/api/v1")
	h.RegisterRoutes(apiAlias, jwtMgr)

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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("HTTP server shutdown error", zap.Error(err))
	}

	log.Info("auth-service stopped gracefully")
	return nil
}

// ensureRefreshTokensTable creates the table if the migration hasn't run yet.
func ensureRefreshTokensTable(ctx context.Context, db *database.DB) error {
	const q = `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			revoked_at TIMESTAMPTZ
		)`
	_, err := db.Pool.Exec(ctx, q)
	return err
}
