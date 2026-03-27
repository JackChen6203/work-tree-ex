package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Environment string
	TripsStore  string
	RuntimeMode string
	CORS        CORSConfig
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
}

func (c Config) DistributedModeEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(c.RuntimeMode), "distributed")
}

type CORSConfig struct {
	AllowedOrigins []string
}

// HTTPConfig holds HTTP server settings.
type HTTPConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	AppRole         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	RequestTimeout  time.Duration
}

// DSN returns a PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password +
		"@" + d.Host + ":" + d.Port +
		"/" + d.Name + "?sslmode=" + d.SSLMode
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// JWTConfig holds JWT authentication settings.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		Environment: getEnv("APP_ENV", "dev"),
		TripsStore:  getEnv("TRIPS_STORE", "memory"),
		RuntimeMode: getEnv("RUNTIME_MODE", "single"),
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		},
		HTTP: HTTPConfig{
			Host:            getEnv("HTTP_HOST", "0.0.0.0"),
			Port:            getEnv("HTTP_PORT", "8080"),
			ReadTimeout:     getDurationSeconds("HTTP_READ_TIMEOUT_SEC", 10),
			WriteTimeout:    getDurationSeconds("HTTP_WRITE_TIMEOUT_SEC", 15),
			ShutdownTimeout: getDurationSeconds("HTTP_SHUTDOWN_TIMEOUT_SEC", 10),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "travel"),
			Password:        getEnv("DB_PASSWORD", "travel"),
			Name:            getEnv("DB_NAME", "travel_planner"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			AppRole:         getEnv("DB_APP_ROLE", "service_role"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDurationMinutes("DB_CONN_MAX_LIFETIME_MIN", 30),
			RequestTimeout:  getDurationSeconds("DB_REQUEST_TIMEOUT_SEC", 5),
		},
		Redis: RedisConfig{
			Addr:            getEnv("REDIS_ADDR", "localhost:6379"),
			Password:        getEnv("REDIS_PASSWORD", ""),
			DB:              getEnvInt("REDIS_DB", 0),
			PoolSize:        getEnvInt("REDIS_POOL_SIZE", 50),
			MinIdleConns:    getEnvInt("REDIS_MIN_IDLE_CONNS", 10),
			ConnMaxLifetime: getDurationMinutes("REDIS_CONN_MAX_LIFETIME_MIN", 30),
			ConnMaxIdleTime: getDurationMinutes("REDIS_CONN_MAX_IDLE_MIN", 5),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "change-me-in-production"),
			AccessTTL:  getDurationMinutes("JWT_ACCESS_TTL_MIN", 60),
			RefreshTTL: getDurationHours("JWT_REFRESH_TTL_HOURS", 168),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := getEnv(key, strconv.Itoa(fallback))
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getDurationSeconds(key string, fallback int) time.Duration {
	return time.Duration(getEnvInt(key, fallback)) * time.Second
}

func getDurationMinutes(key string, fallback int) time.Duration {
	return time.Duration(getEnvInt(key, fallback)) * time.Minute
}

func getDurationHours(key string, fallback int) time.Duration {
	return time.Duration(getEnvInt(key, fallback)) * time.Hour
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	var parts []string
	start := 0
	for i := 0; i <= len(value); i++ {
		if i == len(value) || value[i] == ',' {
			if i > start {
				parts = append(parts, value[start:i])
			}
			start = i + 1
		}
	}

	return parts
}
