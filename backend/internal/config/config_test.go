package config

import "testing"

func TestLoad_ParsesCORSAllowedOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, https://example.com")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_ISSUER", "test-issuer")
	t.Setenv("JWT_AUDIENCE", "test-audience")
	t.Setenv("MAX_REQUEST_BODY_BYTES", "2097152")

	cfg := Load()

	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("CORSAllowedOrigins length = %d, want 2", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[1] != "https://example.com" {
		t.Fatalf("CORSAllowedOrigins[1] = %q, want https://example.com", cfg.CORSAllowedOrigins[1])
	}
	if cfg.JWTIssuer != "test-issuer" {
		t.Fatalf("JWTIssuer = %q, want test-issuer", cfg.JWTIssuer)
	}
	if cfg.JWTAudience != "test-audience" {
		t.Fatalf("JWTAudience = %q, want test-audience", cfg.JWTAudience)
	}
	if cfg.MaxRequestBodyBytes != 2097152 {
		t.Fatalf("MaxRequestBodyBytes = %d, want 2097152", cfg.MaxRequestBodyBytes)
	}
}

func TestParsePositiveInt64(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "valid", value: "1048576", want: 1048576},
		{name: "trims whitespace", value: " 2048 ", want: 2048},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "not number", value: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePositiveInt64(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePositiveInt64 error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("got = %d, want %d", got, tt.want)
			}
		})
	}
}
