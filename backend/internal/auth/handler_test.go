package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-gonic/gin"
)

func newAuthTestRouter(service *Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	passThrough := func(c *gin.Context) { c.Next() }
	RegisterRoutes(api, service, passThrough, passThrough)
	return router
}

func TestHandlerRegister_InvalidPayload(t *testing.T) {
	router := newAuthTestRouter(NewService(newFakeUserRepository(), "test-secret"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
}

func TestHandlerRegister_ValidationFailed(t *testing.T) {
	router := newAuthTestRouter(NewService(newFakeUserRepository(), "test-secret"))

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
	router := newAuthTestRouter(NewService(repo, "test-secret"))

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
	RegisterRoutes(api, NewService(newFakeUserRepository(), "test-secret"), authMiddleware, passThrough)

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
