package configs

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
}

type AppConfig struct {
	Name        string
	Version     string
	Environment string
	Debug       bool
}

type DatabaseConfig struct {
	Driver string // "sqlite"
	Path   string // file path for SQLite
	DSN    string // full DSN
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int // minutes
	RefreshTokenTTL int // hours
}

type ServerConfig struct {
	Host string
	Port int
}

// Load reads config from .env file or environment variables
func Load() (*Config, error) {
	// Try to load .env file (ignore error in production)
	_ = godotenv.Load()

	srvPort, _ := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	jwtAccess, _ := strconv.Atoi(getEnv("JWT_ACCESS_TTL", "60"))
	jwtRefresh, _ := strconv.Atoi(getEnv("JWT_REFRESH_TTL", "168"))
	debug, _ := strconv.ParseBool(getEnv("DEBUG", "false"))

	// SQLite file path - defaults to ./goerp.db next to the binary
	dbPath := getEnv("SQLITE_PATH", "./goerp.db")
	// mattn/go-sqlite3 DSN: WAL mode + busy timeout + foreign keys
	dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&cache=shared"

	return &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "GoERP"),
			Version:     getEnv("APP_VERSION", "2.0.0"),
			Environment: getEnv("APP_ENV", "production"),
			Debug:       debug,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			Path:   dbPath,
			DSN:    dsn,
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "goerp-super-secret-key-change-in-production"),
			AccessTokenTTL:  jwtAccess,
			RefreshTokenTTL: jwtRefresh,
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: srvPort,
		},
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
