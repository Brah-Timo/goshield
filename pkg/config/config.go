// Package config provides centralized configuration loading via Viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig is the top-level configuration structure for any GoShield service.
type AppConfig struct {
	// Generic app block (used by api-gateway and other services that set app.port).
	App          AppBlockConfig     `mapstructure:"app"`
	Service      ServiceConfig      `mapstructure:"service"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Kafka        KafkaConfig        `mapstructure:"kafka"`
	Storage      StorageConfig      `mapstructure:"storage"`
	Auth         AuthConfig         `mapstructure:"auth"`
	JWT          JWTBlockConfig     `mapstructure:"jwt"`
	GRPC         GRPCConfig         `mapstructure:"grpc"`
	Telemetry    TelemetryConfig    `mapstructure:"telemetry"`
	Notification NotificationConfig `mapstructure:"notification"`
	RateLimit    RateLimitConfig    `mapstructure:"rate_limit"`
	Logger       LoggerConfig       `mapstructure:"logger"`
}

// AppBlockConfig is the "app:" block used by services that prefer this naming.
type AppBlockConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Port    int    `mapstructure:"port"`
	Debug   bool   `mapstructure:"debug"`
}

// JWTBlockConfig is the "jwt:" top-level block used by api-gateway and notification-service.
type JWTBlockConfig struct {
	Secret        string        `mapstructure:"secret"`
	AccessExpiry  time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// RateLimitConfig holds api-gateway rate-limiter settings.
type RateLimitConfig struct {
	RequestsPerMinute int           `mapstructure:"requests_per_minute"`
	Burst             int           `mapstructure:"burst"`
	Window            time.Duration `mapstructure:"window"`
	SkipPaths         []string      `mapstructure:"skip_paths"`
}

// LoggerConfig holds structured-logger settings.
type LoggerConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	ServiceName string `mapstructure:"service_name"`
	Development bool   `mapstructure:"development"`
}

type ServiceConfig struct {
	Name        string `mapstructure:"name"`
	Port        int    `mapstructure:"port"`
	Environment string `mapstructure:"environment"` // development | staging | production
	Version     string `mapstructure:"version"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode)
}

func (d DatabaseConfig) DSNWithPool() string {
	return fmt.Sprintf("%s&pool_max_conns=%d&pool_min_conns=%d",
		d.DSN(), d.MaxOpenConns, d.MaxIdleConns)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type KafkaConfig struct {
	Brokers         []string `mapstructure:"brokers"`
	GroupID         string   `mapstructure:"group_id"`
	TopicClaimsNew  string   `mapstructure:"topic_claims_new"`
	TopicClaimsAnalyzed string `mapstructure:"topic_claims_analyzed"`
	TopicClaimsFlagged  string `mapstructure:"topic_claims_flagged"`
	TopicClaimsFailed   string `mapstructure:"topic_claims_failed"`
}

type StorageConfig struct {
	Provider        string `mapstructure:"provider"`  // minio | azure
	Endpoint        string `mapstructure:"endpoint"`
	AccessKey       string `mapstructure:"access_key"`
	SecretKey       string `mapstructure:"secret_key"`
	Bucket          string `mapstructure:"bucket"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	AzureAccountName string `mapstructure:"azure_account_name"`
	AzureAccountKey  string `mapstructure:"azure_account_key"`
	AzureContainer   string `mapstructure:"azure_container"`
}

type AuthConfig struct {
	JWTSecret            string        `mapstructure:"jwt_secret"`
	JWTAccessExpiry      time.Duration `mapstructure:"jwt_access_expiry"`
	JWTRefreshExpiry     time.Duration `mapstructure:"jwt_refresh_expiry"`
	OAuthGoogleClientID  string        `mapstructure:"oauth_google_client_id"`
	OAuthGoogleSecret    string        `mapstructure:"oauth_google_secret"`
	OAuthGoogleCallback  string        `mapstructure:"oauth_google_callback"`
	RateLimitRequests    int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow      time.Duration `mapstructure:"rate_limit_window"`
}

type GRPCConfig struct {
	ClaimServiceAddr    string `mapstructure:"claim_service_addr"`
	AuthServiceAddr     string `mapstructure:"auth_service_addr"`
	AIServiceAddr       string `mapstructure:"ai_service_addr"`
	NotificationServiceAddr string `mapstructure:"notification_service_addr"`
	Port                int    `mapstructure:"port"`
}

type TelemetryConfig struct {
	OTLPEndpoint    string  `mapstructure:"otlp_endpoint"`
	// JaegerEndpoint is the HTTP collector endpoint used by the Jaeger exporter.
	JaegerEndpoint  string  `mapstructure:"jaeger_endpoint"`
	ServiceName     string  `mapstructure:"service_name"`
	SamplingRate    float64 `mapstructure:"sampling_rate"`
	// SampleRate is an alias for SamplingRate for config files that prefer this key.
	SampleRate      float64 `mapstructure:"sample_rate"`
	MetricsEnabled  bool    `mapstructure:"metrics_enabled"`
	TracingEnabled  bool    `mapstructure:"tracing_enabled"`
	// Enabled is an alias for TracingEnabled for config files that prefer this key.
	Enabled         bool    `mapstructure:"enabled"`
}

type NotificationConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	SMTPFrom     string `mapstructure:"smtp_from"`
	SlackWebhook string `mapstructure:"slack_webhook"`
	FraudThresholdAlert float64 `mapstructure:"fraud_threshold_alert"`
	FraudThresholdFlag  float64 `mapstructure:"fraud_threshold_flag"`
}

// Load reads configuration from file + environment variables.
// Environment variables override file values. Prefix: GOSHIELD_
func Load(cfgPath string) (*AppConfig, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("service.port", 8080)
	v.SetDefault("service.environment", "development")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 50)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.pool_size", 20)
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic_claims_new", "claims.new")
	v.SetDefault("kafka.topic_claims_analyzed", "claims.analyzed")
	v.SetDefault("kafka.topic_claims_flagged", "claims.flagged")
	v.SetDefault("kafka.topic_claims_failed", "claims.failed")
	v.SetDefault("storage.provider", "minio")
	v.SetDefault("storage.bucket", "goshield-claims")
	v.SetDefault("auth.jwt_access_expiry", "15m")
	v.SetDefault("auth.jwt_refresh_expiry", "168h")
	v.SetDefault("auth.rate_limit_requests", 100)
	v.SetDefault("auth.rate_limit_window", "1m")
	v.SetDefault("telemetry.sampling_rate", 1.0)
	v.SetDefault("telemetry.metrics_enabled", true)
	v.SetDefault("telemetry.tracing_enabled", true)
	v.SetDefault("notification.smtp_port", 587)
	v.SetDefault("notification.fraud_threshold_alert", 0.95)
	v.SetDefault("notification.fraud_threshold_flag", 0.80)

	// Read config file
	if cfgPath != "" {
		v.SetConfigFile(cfgPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	// Override with environment variables
	v.SetEnvPrefix("GOSHIELD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg AppConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
