package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func newAuthTestRouter(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, service, passThrough, passThrough, passThrough)
	return router
}

func TestHandlerRegister_InvalidPayload(t *testing.T) {
	router := newAuthTestRouter(NewService(newFakeUserRepository(), testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
}

func TestHandlerRegister_ValidationFailed(t *testing.T) {
	router := newAuthTestRouter(NewService(newFakeUserRepository(), testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{
		"email":"not-an-email",
		"full_name":"A",
		"password":"short"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusBadRequest, "validation failed")
}

func TestHandlerLogin_InvalidCredentialsDoesNotLeakDetails(t *testing.T) {
	repo := newFakeUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword error = %v", err)
	}
	repo.usersByEmail["user@example.com"] = &User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "user",
	}
	router := newAuthTestRouter(NewService(repo, testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"email":"user@example.com",
		"password":"wrong-password"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusUnauthorized, "invalid email or password")
	if bytes.Contains(w.Body.Bytes(), []byte("bcrypt")) {
		t.Fatal("response leaked internal bcrypt details")
	}
}

func TestHandlerLogin_UsesRateLimiterBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	limiterCalled := false
	blockingLimiter := func(c *gin.Context) {
		limiterCalled = true
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, NewService(newFakeUserRepository(), testTokenConfig()), passThrough, blockingLimiter, passThrough)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if !limiterCalled {
		t.Fatal("login rate limiter was not called")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestHandlerRegister_UsesRateLimiterBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	limiterCalled := false
	blockingLimiter := func(c *gin.Context) {
		limiterCalled = true
		c.AbortWithStatus(http.StatusTooManyRequests)
	}
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, NewService(newFakeUserRepository(), testTokenConfig()), passThrough, passThrough, blockingLimiter)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(`{"email":"user@example.com","full_name":"Test User","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if !limiterCalled {
		t.Fatal("register rate limiter was not called")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestHandlerMe_ReturnsAuthenticatedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "admin")
		c.Next()
	}
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, NewService(newFakeUserRepository(), testTokenConfig()), authMiddleware, passThrough, passThrough)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	data := body["data"].(map[string]any)
	if data["user_id"] != "user-1" || data["role"] != "admin" {
		t.Fatalf("data = %+v, want user-1/admin", data)
	}
}

func TestHandlerLogout_RevokesCurrentToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testTokenConfig()
	store := NewMemoryTokenRevocationStore()
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := authHandlerTestMiddleware(cfg, store)
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(
		api,
		NewService(newFakeUserRepository(), cfg, WithTokenRevocationStore(store)),
		authMiddleware,
		passThrough,
		passThrough,
	)

	token, err := GenerateToken(cfg, "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("me status after logout = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandlerLogout_TokenMissingJTIUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testTokenConfig()
	store := NewMemoryTokenRevocationStore()
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := authHandlerTestMiddleware(cfg, store)
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, NewService(newFakeUserRepository(), cfg, WithTokenRevocationStore(store)), authMiddleware, passThrough, passThrough)

	tokenString := signClaimsForHandlerTest(t, cfg, Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{cfg.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestHandlerLogout_RevocationErrorReturnsServiceUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := func(c *gin.Context) {
		c.Set("token_jti", "jti-1")
		c.Set("token_expires_at", time.Now().Add(time.Hour))
		c.Next()
	}
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(
		api,
		NewService(newFakeUserRepository(), testTokenConfig(), WithTokenRevocationStore(&failingAuthRevocationStore{})),
		authMiddleware,
		passThrough,
		passThrough,
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusServiceUnavailable, "logout failed")
}

func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if w.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, status, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != message {
		t.Fatalf("message = %v, want %s", body["message"], message)
	}
	if _, ok := body["data"]; ok {
		t.Fatal("error response should not include data")
	}
}

func signClaimsForHandlerTest(t *testing.T, cfg TokenConfig, claims Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}
	return tokenString
}

type failingAuthRevocationStore struct{}

func (s *failingAuthRevocationStore) RevokeToken(context.Context, string, time.Duration) error {
	return errors.New("redis unavailable")
}

func (s *failingAuthRevocationStore) IsTokenRevoked(context.Context, string) (bool, error) {
	return false, errors.New("redis unavailable")
}

func (s *failingAuthRevocationStore) Close() error {
	return nil
}

func authHandlerTestMiddleware(cfg TokenConfig, store TokenRevocationStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(header, "Bearer ")
		if tokenString == header {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			c.Abort()
			return
		}

		claims, err := ParseToken(cfg, tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			c.Abort()
			return
		}
		revoked, err := store.IsTokenRevoked(c.Request.Context(), claims.ID)
		if err != nil || revoked {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Set("token_jti", claims.ID)
		c.Set("token_expires_at", claims.ExpiresAt.Time)
		c.Next()
	}
}
