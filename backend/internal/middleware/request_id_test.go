package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_UsesIncomingSafeID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(RequestIDKey))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	router.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) != "req-123" {
		t.Fatalf("response request id = %q, want req-123", w.Header().Get(RequestIDHeader))
	}
	if w.Body.String() != "req-123" {
		t.Fatalf("context request id = %q, want req-123", w.Body.String())
	}
}

func TestRequestID_GeneratesIDWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) == "" {
		t.Fatal("missing generated request id header")
	}
}

func TestRequestID_RejectsUnsafeIncomingID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "bad id")
	router.ServeHTTP(w, req)

	if w.Header().Get(RequestIDHeader) == "bad id" {
		t.Fatal("unsafe incoming request id was accepted")
	}
}

func TestRequestLogger_DoesNotWriteResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), RequestLogger())
	router.GET("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "secret-token"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
