// Package main is the entry point for the notification-service.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/goshield/pkg/config"
	"github.com/goshield/pkg/events"
	"github.com/goshield/pkg/logger"
	"github.com/goshield/pkg/middleware"
	"github.com/goshield/pkg/telemetry"
	"github.com/goshield/services/notification-service/internal/consumer"
	"github.com/goshield/services/notification-service/internal/email"
	"github.com/goshield/services/notification-service/internal/handler"
	"github.com/goshield/services/notification-service/internal/hub"
	"github.com/goshield/services/notification-service/internal/slack"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "notification-service fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config/config.yaml"
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

	log.Info("starting notification-service", zap.String("env", cfg.Service.Environment))

	ctx := context.Background()
	tel, err := telemetry.Setup(ctx, cfg.Telemetry, log)
	if err != nil {
		log.Warn("telemetry non-fatal", zap.Error(err))
	} else {
		defer tel.Shutdown(ctx) //nolint:errcheck
	}

	// WebSocket hub
	wsHub := hub.New(log)
	go wsHub.Run()

	// Email + Slack notifiers
	mailer := email.New(cfg.Notification, log)
	slackNotifier := slack.New(cfg.Notification.SlackWebhook, log)

	// Kafka producer (for potential outbound events)
	producer := events.NewProducer(cfg.Kafka.Brokers, log)
	defer producer.Close() //nolint:errcheck

	// Notification consumer
	notifConsumer := consumer.New(
		wsHub,
		mailer,
		slackNotifier,
		cfg.Notification.FraudThresholdAlert,
		log,
	)

	// Two Kafka consumers: claims.flagged + claims.analyzed
	flaggedConsumer := events.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicClaimsFlagged, "notification-service-flagged", log)
	analyzedConsumer := events.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicClaimsAnalyzed, "notification-service-analyzed", log)

	consumerCtx, cancelConsumers := context.WithCancel(ctx)
	go flaggedConsumer.Consume(consumerCtx, notifConsumer.Handle) //nolint:errcheck
	go analyzedConsumer.Consume(consumerCtx, notifConsumer.Handle) //nolint:errcheck

	// JWT manager for WebSocket auth
	jwtMgr := middleware.NewJWTManager(middleware.JWTConfig{
		Secret:        cfg.Auth.JWTSecret,
		AccessExpiry:  cfg.Auth.JWTAccessExpiry,
		RefreshExpiry: cfg.Auth.JWTRefreshExpiry,
	})

	// HTTP + WebSocket server
	wsHandler := handler.NewWSHandler(wsHub, log)
	app := fiber.New(fiber.Config{AppName: "notification-service"})
	wsHandler.RegisterRoutes(app, jwtMgr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverDone := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Service.Port)
		log.Info("notification-service HTTP+WS listening", zap.String("addr", addr))
		serverDone <- app.Listen(addr)
	}()

	select {
	case sig := <-quit:
		log.Info("shutdown", zap.String("signal", sig.String()))
	case err := <-serverDone:
		if err != nil {
			log.Error("server error", zap.Error(err))
		}
	}

	cancelConsumers()
	_ = flaggedConsumer.Close()
	_ = analyzedConsumer.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = app.ShutdownWithContext(shutdownCtx)

	log.Info("notification-service stopped")
	return nil
}
