package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"stylemind/pkg/logger"
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

func TestHandlerLoginSuccess_WritesAuditEvent(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	repo := newFakeUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword error = %v", err)
	}
	repo.usersByEmail["user@example.com"] = &User{
		ID:           "user-1",
		Email:        "user@example.com",
		PasswordHash: string(hash),
		Role:         "user",
	}
	router := newAuthAuditTestRouter(NewService(repo, testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"email":"USER@example.com",
		"password":"password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "audit-test")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	assertAuditEvent(t, audit.String(), map[string]any{
		"type":       "audit",
		"event":      "auth.login",
		"result":     "success",
		"user_id":    "user-1",
		"role":       "user",
		"email":      "user@example.com",
		"request_id": "req-audit-1",
		"user_agent": "audit-test",
	})
	assertAuditDoesNotContain(t, audit.String(), "password123", "Authorization", "Bearer", "token")
}

func TestHandlerLoginFailed_WritesAuditEventWithoutPassword(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	router := newAuthAuditTestRouter(NewService(newFakeUserRepository(), testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"email":"missing@example.com",
		"password":"password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer should-not-log")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusUnauthorized, "invalid email or password")
	assertAuditEvent(t, audit.String(), map[string]any{
		"event":  "auth.login",
		"result": "failed",
		"email":  "missing@example.com",
		"reason": "invalid_credentials",
	})
	assertAuditDoesNotContain(t, audit.String(), "password123", "should-not-log", "Authorization", "Bearer")
}

func TestHandlerLoginDisabled_ReturnsForbiddenAndAuditsSafely(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	repo := newFakeUserRepository()
	hash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword error = %v", err)
	}
	repo.usersByEmail["disabled@example.com"] = &User{
		ID:           "disabled-1",
		Email:        "disabled@example.com",
		PasswordHash: string(hash),
		Role:         "user",
		Status:       "disabled",
	}
	router := newAuthAuditTestRouter(NewService(repo, testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{
		"email":"disabled@example.com",
		"password":"password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusForbidden, "account disabled")
	assertAuditEvent(t, audit.String(), map[string]any{
		"event":  "auth.login",
		"result": "failed",
		"email":  "disabled@example.com",
		"reason": "account_disabled",
	})
	assertAuditDoesNotContain(t, audit.String(), "password123")
}

func TestHandlerRegisterSuccess_WritesAuditEvent(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	router := newAuthAuditTestRouter(NewService(newFakeUserRepository(), testTokenConfig()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{
		"email":"NEW@example.com",
		"full_name":"New User",
		"password":"password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	assertAuditEvent(t, audit.String(), map[string]any{
		"event":  "auth.register",
		"result": "success",
		"email":  "new@example.com",
		"role":   "user",
	})
	assertAuditDoesNotContain(t, audit.String(), "password123")
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
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

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
	assertAuditEvent(t, audit.String(), map[string]any{
		"event":   "auth.logout",
		"result":  "success",
		"user_id": "user-1",
		"role":    "user",
	})

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
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

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
	assertAuditEvent(t, audit.String(), map[string]any{
		"event":  "auth.logout",
		"result": "failed",
		"reason": "revocation_store_error",
	})
}

func TestHandlerLogout_UsesRequestContextDeadlineForRevocation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &recordingAuthRevocationStore{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	api := router.Group("/api/v1")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Set("token_jti", "jti-1")
		c.Set("token_expires_at", time.Now().Add(time.Hour))
		c.Next()
	}
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(
		api,
		NewService(newFakeUserRepository(), testTokenConfig(), WithTokenRevocationStore(store)),
		authMiddleware,
		passThrough,
		passThrough,
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if !store.contextHadDeadline {
		t.Fatal("revocation store did not receive request context deadline")
	}
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

func newAuthAuditTestRouter(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-audit-1")
		c.Next()
	})
	api := router.Group("/api/v1")
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, service, passThrough, passThrough, passThrough)
	return router
}

func assertAuditEvent(t *testing.T, raw string, expected map[string]any) {
	t.Helper()

	entries := strings.Split(strings.TrimSpace(raw), "\n")
	if len(entries) == 0 || entries[0] == "" {
		t.Fatal("expected audit log entry, got empty output")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(entries[len(entries)-1]), &entry); err != nil {
		t.Fatalf("audit json unmarshal error = %v, raw=%s", err, raw)
	}
	for key, want := range expected {
		if got := entry[key]; got != want {
			t.Fatalf("audit[%s] = %v, want %v; entry=%+v", key, got, want, entry)
		}
	}
}

func assertAuditDoesNotContain(t *testing.T, raw string, forbidden ...string) {
	t.Helper()

	for _, value := range forbidden {
		if strings.Contains(raw, value) {
			t.Fatalf("audit output contained forbidden value %q: %s", value, raw)
		}
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

type recordingAuthRevocationStore struct {
	contextHadDeadline bool
}

func (s *recordingAuthRevocationStore) RevokeToken(ctx context.Context, _ string, _ time.Duration) error {
	_, s.contextHadDeadline = ctx.Deadline()
	return nil
}

func (s *recordingAuthRevocationStore) IsTokenRevoked(context.Context, string) (bool, error) {
	return false, nil
}

func (s *recordingAuthRevocationStore) Close() error {
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
