package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                string
	Port                  string
	JWTSecret             string
	JWTIssuer             string
	JWTAudience           string
	RequestTimeoutSeconds int64
	MaxRequestBodyBytes   int64
	AuthRateLimitRequests int
	CORSAllowedOrigins    []string
	Database              DatabaseConfig
	Redis                 RedisConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env not found, using environment variables")
	}

	appEnv := getEnv("APP_ENV", "development")
	requestTimeoutSeconds, err := parsePositiveInt64(getEnv("REQUEST_TIMEOUT_SECONDS", "10"))
	if err != nil {
		log.Fatalf("REQUEST_TIMEOUT_SECONDS must be a positive integer: %v", err)
	}
	maxRequestBodyBytes, err := parsePositiveInt64(getEnv("MAX_REQUEST_BODY_BYTES", "1048576"))
	if err != nil {
		log.Fatalf("MAX_REQUEST_BODY_BYTES must be a positive integer: %v", err)
	}
	authRateLimitRequests, err := parsePositiveInt(getEnv("AUTH_RATE_LIMIT_REQUESTS", "10"))
	if err != nil {
		log.Fatalf("AUTH_RATE_LIMIT_REQUESTS must be a positive integer: %v", err)
	}

	cfg := Config{
		AppEnv:                appEnv,
		Port:                  getEnv("PORT", "8080"),
		JWTSecret:             getEnv("JWT_SECRET", "change-me-in-production"),
		JWTIssuer:             getEnv("JWT_ISSUER", "stylemind-api"),
		JWTAudience:           getEnv("JWT_AUDIENCE", "stylemind-web"),
		RequestTimeoutSeconds: requestTimeoutSeconds,
		MaxRequestBodyBytes:   maxRequestBodyBytes,
		AuthRateLimitRequests: authRateLimitRequests,
		CORSAllowedOrigins:    getEnvList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "stylemind"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", ""),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnv("REDIS_DB", "0"),
		},
	}

	cfg.validate()
	return cfg
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvList(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return fallback
	}
	return items
}

func parsePositiveInt64(value string) (int64, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, strconv.ErrSyntax
	}
	return parsed, nil
}

func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if parsed <= 0 {
		return 0, strconv.ErrSyntax
	}
	return parsed, nil
}

func (c Config) validate() {
	if !strings.EqualFold(c.AppEnv, "production") {
		return
	}

	if c.JWTSecret == "" || c.JWTSecret == "change-me-in-production" || c.JWTSecret == "change_me_in_production" {
		log.Fatal("JWT_SECRET must be configured in production")
	}
	if c.JWTIssuer == "" {
		log.Fatal("JWT_ISSUER must be configured in production")
	}
	if c.JWTAudience == "" {
		log.Fatal("JWT_AUDIENCE must be configured in production")
	}
	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			log.Fatal("CORS_ALLOWED_ORIGINS cannot contain * in production")
		}
	}
}
