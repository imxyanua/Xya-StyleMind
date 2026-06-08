package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stylemind/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWTAuth(middlewareTestTokenConfig()), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()

	r.GET("/protected", JWTAuth(cfg), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString("user_id"),
			"role":    c.GetString("user_role"),
		})
	})

	token, err := auth.GenerateToken(cfg, "u1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["user_id"] != "u1" {
		t.Fatalf("user_id = %v, want u1", body["user_id"])
	}
	if body["role"] != "admin" {
		t.Fatalf("role = %v, want admin", body["role"])
	}
}

func TestJWTAuth_ValidTokenWithRevocationStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()
	store := auth.NewMemoryTokenRevocationStore()

	r.GET("/protected", JWTAuth(cfg, WithTokenRevocationStore(store)), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString("user_id"),
			"jti":     c.GetString("token_jti"),
		})
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestJWTAuth_RevokedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()
	store := auth.NewMemoryTokenRevocationStore()

	r.GET("/protected", JWTAuth(cfg, WithTokenRevocationStore(store)), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}
	claims, err := auth.ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}
	if err := store.RevokeToken(context.Background(), claims.ID, time.Hour); err != nil {
		t.Fatalf("RevokeToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_RevokedJTIExpires(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()
	store := auth.NewMemoryTokenRevocationStore()

	r.GET("/protected", JWTAuth(cfg, WithTokenRevocationStore(store)), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}
	claims, err := auth.ParseToken(cfg, token)
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}
	if err := store.RevokeToken(context.Background(), claims.ID, 10*time.Millisecond); err != nil {
		t.Fatalf("RevokeToken error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestJWTAuth_RevocationStoreErrorFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()

	r.GET("/protected", JWTAuth(cfg, WithTokenRevocationStore(&failingTokenRevocationStore{})), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if w.Body.String() != `{"success":false,"message":"unauthorized"}` {
		t.Fatalf("body leaked details or had wrong format: %s", w.Body.String())
	}
}

func TestJWTAuth_DisabledUserFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()

	r.GET("/protected", JWTAuth(cfg, WithUserStatusChecker(&fakeUserStatusChecker{
		user: &auth.User{ID: "u1", Role: "user", Status: "disabled"},
	})), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if w.Body.String() != `{"success":false,"message":"unauthorized"}` {
		t.Fatalf("body leaked details or had wrong format: %s", w.Body.String())
	}
}

func TestJWTAuth_ActiveUserWithStatusCheckerPasses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()

	r.GET("/protected", JWTAuth(cfg, WithUserStatusChecker(&fakeUserStatusChecker{
		user: &auth.User{ID: "u1", Role: "user", Status: "active"},
	})), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestJWTAuth_StatusCheckerErrorFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()

	r.GET("/protected", JWTAuth(cfg, WithUserStatusChecker(&fakeUserStatusChecker{
		err: errors.New("database unavailable"),
	})), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token, err := auth.GenerateToken(cfg, "u1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_MalformedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWTAuth(middlewareTestTokenConfig()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middlewareTestTokenConfig()
	r.GET("/protected", JWTAuth(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ID:        "token-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestJWTAuth_WrongIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := middlewareTestTokenConfig()
	r := gin.New()
	r.GET("/protected", JWTAuth(cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wrong-issuer",
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

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func middlewareTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		Secret:   "test-secret",
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
	}
}

type failingTokenRevocationStore struct{}

func (s *failingTokenRevocationStore) RevokeToken(context.Context, string, time.Duration) error {
	return errors.New("redis unavailable")
}

func (s *failingTokenRevocationStore) IsTokenRevoked(context.Context, string) (bool, error) {
	return false, errors.New("redis unavailable")
}

func (s *failingTokenRevocationStore) Close() error {
	return nil
}

type fakeUserStatusChecker struct {
	user *auth.User
	err  error
}

func (c *fakeUserStatusChecker) GetUserByID(context.Context, string) (*auth.User, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.user, nil
}
