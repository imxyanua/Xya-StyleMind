package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndParseToken(t *testing.T) {
	secret := "test-secret"
	userID := "user-123"
	role := "admin"

	token, err := GenerateToken(secret, userID, role)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("claims.UserID = %q, want %q", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Fatalf("claims.Role = %q, want %q", claims.Role, role)
	}
}

func TestParseTokenInvalidSecret(t *testing.T) {
	token, err := GenerateToken("secret-a", "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, err = ParseToken("secret-b", token)
	if err == nil {
		t.Fatal("ParseToken() expected error for invalid secret, got nil")
	}
}

func TestParseTokenExpired(t *testing.T) {
	secret := "test-secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}

	_, err = ParseToken(secret, tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for expired token, got nil")
	}
}

func TestParseTokenRejectsUnexpectedSigningMethod(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}

	_, err = ParseToken("test-secret", tokenString)
	if err == nil {
		t.Fatal("ParseToken() expected error for unexpected signing method, got nil")
	}
}
