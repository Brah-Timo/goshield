// Package main is the entry point for the GoShield api-gateway.
//
// Route map
//   /auth/**            → auth-service      (public — no JWT required)
//   /claims/**          → claim-service     (JWT required)
//   /notifications/**   → notification-service (JWT required)
//   /ws                 → notification-service (JWT via ?token=)
//   /ai/**              → ai-service-go     (JWT required)
//   /health             → gateway liveness
//   /readyz             → gateway readiness (checks Redis)
//   /metrics            → Prometheus scrape endpoint
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	fibercors "github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	pkgcfg "github.com/goshield/pkg/config"
	pkglog "github.com/goshield/pkg/logger"
	pkgmw "github.com/goshield/pkg/middleware"
	pkgtel "github.com/goshield/pkg/telemetry"
	gwhandler "github.com/goshield/services/api-gateway/internal/handler"
	gwmw "github.com/goshield/services/api-gateway/internal/middleware"
	"github.com/goshield/services/api-gateway/internal/proxy"
	"github.com/goshield/services/api-gateway/internal/ratelimit"
)

func main() {
	// ── Config ──────────────────────────────────────────────────────────────
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/config")
	viper.AddConfigPath("./config")
	viper.SetEnvPrefix("GOSHIELD")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	var cfg pkgcfg.AppConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "config unmarshal: %v\n", err)
		os.Exit(1)
	}

	// ── Logger ───────────────────────────────────────────────────────────────
	log, err := pkglog.New(pkglog.Config{
		Level:       cfg.Logger.Level,
		Format:      cfg.Logger.Format,
		ServiceName: cfg.Logger.ServiceName,
		Development: cfg.Logger.Development,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// ── Telemetry ────────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	tp, err := pkgtel.Setup(ctx, pkgtel.TelemetryConfig{
		JaegerEndpoint: cfg.Telemetry.JaegerEndpoint,
		ServiceName:    cfg.Telemetry.ServiceName,
		SampleRate:     cfg.Telemetry.SampleRate,
		Enabled:        cfg.Telemetry.Enabled,
	}, log)
	if err != nil {
		log.Fatal("telemetry init", zap.Error(err))
	}
	defer tp.Shutdown(ctx) //nolint:errcheck

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Warn("Redis unavailable at startup — rate limiter will fail-open", zap.Error(err))
	} else {
		log.Info("Redis connected", zap.String("addr", cfg.Redis.Addr))
	}
	defer rdb.Close()

	// ── JWT Manager ───────────────────────────────────────────────────────────
	jwtMgr := pkgmw.NewJWTManager(pkgmw.JWTConfig{
		Secret:        cfg.JWT.Secret,
		AccessExpiry:  cfg.JWT.AccessExpiry,
		RefreshExpiry: cfg.JWT.RefreshExpiry,
	})

	// ── Rate Limiter ──────────────────────────────────────────────────────────
	limiterCfg := ratelimit.Config{
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		Burst:             cfg.RateLimit.Burst,
		Window:            cfg.RateLimit.Window,
		SkipPaths:         cfg.RateLimit.SkipPaths,
	}
	limiter := ratelimit.New(rdb, limiterCfg)

	// ── Reverse Proxy Router ──────────────────────────────────────────────────
	upstreamCfg := proxy.UpstreamConfig{
		AuthService:         viper.GetString("upstream.auth_service"),
		ClaimService:        viper.GetString("upstream.claim_service"),
		NotificationService: viper.GetString("upstream.notification_service"),
		AIServiceGo:         viper.GetString("upstream.ai_service_go"),
	}
	proxyRouter := proxy.New(upstreamCfg, log)

	// ── Fiber App ─────────────────────────────────────────────────────────────
	app := fiber.New(fiber.Config{
		AppName:               "GoShield API Gateway",
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Error("unhandled error", zap.Error(err), zap.String("path", c.Path()))
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// ── Global Middleware Stack ───────────────────────────────────────────────
	app.Use(fiberrecover.New(fiberrecover.Config{EnableStackTrace: cfg.App.Debug}))
	app.Use(fiberlogger.New(fiberlogger.Config{
		Format: "${time} | ${status} | ${latency} | ${ip} | ${method} ${path}\n",
	}))

	// CORS
	allowedOrigins := viper.GetStringSlice("cors.allowed_origins")
	originsStr := ""
	for i, o := range allowedOrigins {
		if i > 0 {
			originsStr += ","
		}
		originsStr += o
	}
	app.Use(fibercors.New(fibercors.Config{
		AllowOrigins:     originsStr,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Authorization,Content-Type,X-Request-ID,X-Tenant-ID",
		ExposeHeaders:    "X-Request-ID",
		AllowCredentials: viper.GetBool("cors.allow_credentials"),
		MaxAge:           3600,
	}))

	// OpenTelemetry tracing
	app.Use(gwmw.TracingMiddleware())

	// Rate limiting (applied globally, skip list honoured inside)
	app.Use(gwmw.RateLimitMiddleware(limiter, limiterCfg.SkipPaths, log))

	// JWT auth — public paths skip validation
	publicPaths := []string{
		"/health", "/readyz", "/metrics",
		"/auth/v1/register", "/auth/v1/login",
		"/auth/v1/refresh", "/auth/v1/oauth",
	}
	app.Use(gwmw.JWTAuthMiddleware(jwtMgr, publicPaths))

	// ── Route Registration ─────────────────────────────────────────────────────
	gwHandler := gwhandler.New(rdb, log)
	gwHandler.RegisterRoutes(app)
	proxyRouter.RegisterRoutes(app)

	// ── Graceful Start / Stop ─────────────────────────────────────────────────
	addr := fmt.Sprintf(":%d", cfg.App.Port)
	go func() {
		log.Info("api-gateway listening", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			log.Error("server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}
	log.Info("api-gateway stopped")
}
