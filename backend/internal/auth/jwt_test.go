package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseToken(t *testing.T) {
	cfg := testTokenConfig()
	userID := "user-123"
	role := "admin"

	token, err := GenerateToken(cfg, userID, role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("claims.UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Fatalf("claims.Role = %q, want %q", claims.Role, role)
	}
	if claims.Issuer != cfg.Issuer {
		t.Fatalf("claims.Issuer = %q, want %q", claims.Issuer, cfg.Issuer)
	}
	hasAudience := false
	for _, audience := range claims.Audience {
		if audience == cfg.Audience {
			hasAudience = true
			break
		}
	}
	if !hasAudience {
		t.Fatalf("claims.Audience = %v, want %q", claims.Audience, cfg.Audience)
	}
	if claims.ID == "" {
		t.Fatal("claims.ID is empty, want jti")
	}
	if claims.NotBefore == nil {
		t.Fatal("claims.NotBefore is nil, want nbf")
	}
}

func TestParseTokenInvalidSecret(t *testing.T) {
	cfg := testTokenConfig()
	token, err := GenerateToken(cfg, "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	cfg.Secret = "wrong-secret"
	_, err = ParseToken(cfg, token)
	if err == nil {
		t.Fatal("ParseToken() expected error for invalid secret, got nil")
	}
}

func TestParseTokenRejectsWrongIssuer(t *testing.T) {
	cfg := testTokenConfig()
	tokenString := signClaimsForTest(t, cfg, validClaimsForTest(cfg))

	cfg.Issuer = "other-issuer"
	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for wrong issuer, got nil")
	}
}

func TestParseTokenRejectsWrongAudience(t *testing.T) {
	cfg := testTokenConfig()
	tokenString := signClaimsForTest(t, cfg, validClaimsForTest(cfg))

	cfg.Audience = "other-audience"
	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for wrong audience, got nil")
	}
}

func TestParseTokenRejectsMissingJTI(t *testing.T) {
	cfg := testTokenConfig()
	claims := validClaimsForTest(cfg)
	claims.ID = ""
	tokenString := signClaimsForTest(t, cfg, claims)

	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for missing jti, got nil")
	}
}

func TestParseTokenRejectsMissingIssuedAt(t *testing.T) {
	cfg := testTokenConfig()
	claims := validClaimsForTest(cfg)
	claims.IssuedAt = nil
	tokenString := signClaimsForTest(t, cfg, claims)

	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for missing iat, got nil")
	}
}

func TestParseTokenRejectsMissingNotBefore(t *testing.T) {
	cfg := testTokenConfig()
	claims := validClaimsForTest(cfg)
	claims.NotBefore = nil
	tokenString := signClaimsForTest(t, cfg, claims)

	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for missing nbf, got nil")
	}
}

func TestParseTokenExpired(t *testing.T) {
	cfg := testTokenConfig()
	claims := validClaimsForTest(cfg)
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	tokenString := signClaimsForTest(t, cfg, claims)

	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for expired token, got nil")
	}
}

func TestParseTokenRejectsNotBeforeInFuture(t *testing.T) {
	cfg := testTokenConfig()
	claims := validClaimsForTest(cfg)
	claims.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Minute))
	tokenString := signClaimsForTest(t, cfg, claims)

	_, err := ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for future nbf, got nil")
	}
}

func TestParseTokenRejectsMalformedToken(t *testing.T) {
	_, err := ParseToken(testTokenConfig(), "not-a-token")
	if err == nil {
		t.Fatal("ParseToken() expected error for malformed token, got nil")
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	cfg := testTokenConfig()
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ID:        "token-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}

	_, err = ParseToken(cfg, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for unexpected signing method, got nil")
	}
}

func testTokenConfig() TokenConfig {
	return TokenConfig{
		Secret:   "test-secret",
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
		TTL:      DefaultTokenTTL,
	}
}

func validClaimsForTest(cfg TokenConfig) Claims {
	now := time.Now()
	return Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ID:        "token-id",
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
}

func signClaimsForTest(t *testing.T, cfg TokenConfig, claims Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}
	return tokenString
}
