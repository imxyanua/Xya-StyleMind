package config

import "testing"

func TestLoad_ParsesCORSAllowedOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, https://example.com")
	t.Setenv("JWT_SECRET", "test-secret")

	cfg := Load()

	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("CORSAllowedOrigins length = %d, want 2", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[1] != "https://example.com" {
		t.Fatalf("CORSAllowedOrigins[1] = %q, want https://example.com", cfg.CORSAllowedOrigins[1])
	}
}
