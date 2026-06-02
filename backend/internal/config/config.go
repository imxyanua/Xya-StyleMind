package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	Port               string
	JWTSecret          string
	JWTIssuer          string
	JWTAudience        string
	CORSAllowedOrigins []string
	Database           DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env not found, using environment variables")
	}

	appEnv := getEnv("APP_ENV", "development")
	cfg := Config{
		AppEnv:             appEnv,
		Port:               getEnv("PORT", "8080"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		JWTIssuer:          getEnv("JWT_ISSUER", "stylemind-api"),
		JWTAudience:        getEnv("JWT_AUDIENCE", "stylemind-web"),
		CORSAllowedOrigins: getEnvList("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "stylemind"),
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
