package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_AddsBaselineHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	router.ServeHTTP(w, req)

	expected := map[string]string{
		headerContentTypeOptions: "nosniff",
		headerFrameOptions:       "DENY",
		headerReferrerPolicy:     "no-referrer",
		headerCSP:                "default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'",
		headerPermissionsPolicy:  "camera=(), microphone=(), geolocation=(), payment=()",
	}
	for header, want := range expected {
		if got := w.Header().Get(header); got != want {
			t.Fatalf("%s = %q, want %q", header, got, want)
		}
	}
	if got := w.Header().Get(headerCacheControl); got != "" {
		t.Fatalf("Cache-Control = %q, want empty for health endpoint", got)
	}
}

func TestSecurityHeaders_DisablesCacheForAuthAndAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/api/v1/auth/me", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/api/v1/admin/orders", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for _, path := range []string{"/api/v1/auth/me", "/api/v1/admin/orders"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)

			if got := w.Header().Get(headerCacheControl); got != "no-store" {
				t.Fatalf("Cache-Control = %q, want no-store", got)
			}
		})
	}
}

func TestRequestBodyLimit_AllowsBodyWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit(16))
	router.POST("/echo", func(c *gin.Context) {
		body := make(map[string]string)
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, body)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"a":"b"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestRequestBodyLimit_BlocksBodyOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestBodyLimit(8))
	router.POST("/echo", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"too":"large"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusRequestEntityTooLarge)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != "request body too large" {
		t.Fatalf("message = %v, want request body too large", body["message"])
	}
}
