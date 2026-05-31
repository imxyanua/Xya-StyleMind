package auth

import "testing"

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
