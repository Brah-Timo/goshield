// Package logger provides a structured zap logger with trace_id injection.
package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey contextKey = "logger"

// Config holds logger configuration.
type Config struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`      // json | console
	ServiceName string `mapstructure:"service_name"`
	Development bool   `mapstructure:"development"` // use development preset
}

// New creates a production-ready zap logger.
func New(cfg Config) (*zap.Logger, error) {
	var zapCfg zap.Config
	if cfg.Development || cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	logger, err := zapCfg.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(0),
		zap.Fields(
			zap.String("service", cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}
	return logger, nil
}

// WithContext injects logger into context.
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves logger from context, falls back to global.
func FromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.L()
}

// WithFields returns a logger with additional fields.
func WithFields(logger *zap.Logger, fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// WithTraceID returns a logger with trace_id field.
func WithTraceID(logger *zap.Logger, traceID string) *zap.Logger {
	return logger.With(zap.String("trace_id", traceID))
}

// WithTenant returns a logger scoped to a company/tenant.
func WithTenant(logger *zap.Logger, companyID, userID string) *zap.Logger {
	return logger.With(
		zap.String("company_id", companyID),
		zap.String("user_id", userID),
	)
}
