package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHTTPMetrics_RecordsRequestCountLatencyAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewHTTPMetrics()
	router := gin.New()
	router.Use(metrics.Middleware())
	metrics.RegisterRoutes(router)
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/fail", func(c *gin.Context) {
		c.Status(http.StatusInternalServerError)
	})

	serve(router, http.MethodGet, "/ok", nil)
	serve(router, http.MethodGet, "/fail", nil)

	body := serve(router, http.MethodGet, "/metrics", nil).Body.String()
	assertContains(t, body, `http_requests_total{method="GET",route="/ok",status="200"} 1`)
	assertContains(t, body, `http_requests_total{method="GET",route="/fail",status="500"} 1`)
	assertContains(t, body, `http_errors_total{method="GET",route="/fail",status="500"} 1`)
	assertContains(t, body, `http_request_duration_seconds_bucket{method="GET",route="/ok",status="200"`)
}

func TestHTTPMetrics_UsesRoutePatternInsteadOfRawPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewHTTPMetrics()
	router := gin.New()
	router.Use(metrics.Middleware())
	metrics.RegisterRoutes(router)
	router.GET("/api/v1/orders/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	rawID := "123e4567-e89b-12d3-a456-426614174000"
	serve(router, http.MethodGet, "/api/v1/orders/"+rawID, nil)

	body := serve(router, http.MethodGet, "/metrics", nil).Body.String()
	assertContains(t, body, `route="/api/v1/orders/:id"`)
	if strings.Contains(body, rawID) {
		t.Fatalf("metrics output contained raw id %q: %s", rawID, body)
	}
}

func TestHTTPMetrics_MetricsEndpointReturnsPrometheusText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewHTTPMetrics()
	router := gin.New()
	router.Use(metrics.Middleware())
	metrics.RegisterRoutes(router)
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	serve(router, http.MethodGet, "/ping", nil)
	w := serve(router, http.MethodGet, "/metrics", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", w.Header().Get("Content-Type"))
	}
	assertContains(t, w.Body.String(), "# HELP http_requests_total")
}

func TestHTTPMetrics_DoesNotExposeSensitiveRequestData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := NewHTTPMetrics()
	router := gin.New()
	router.Use(metrics.Middleware())
	metrics.RegisterRoutes(router)
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	serve(router, http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.com","password":"password123"}`), func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer raw-token-value")
		req.Header.Set("Content-Type", "application/json")
	})

	body := serve(router, http.MethodGet, "/metrics", nil).Body.String()
	for _, forbidden := range []string{"user@example.com", "password123", "raw-token-value", "Authorization", "Bearer"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metrics output contained forbidden value %q: %s", forbidden, body)
		}
	}
}

func serve(router *gin.Engine, method, path string, body io.Reader, mutators ...func(*http.Request)) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	for _, mutate := range mutators {
		mutate(req)
	}
	router.ServeHTTP(w, req)
	return w
}

func assertContains(t *testing.T, value, pattern string) {
	t.Helper()
	if !strings.Contains(value, pattern) {
		t.Fatalf("expected output to contain %q, got: %s", pattern, value)
	}
}
