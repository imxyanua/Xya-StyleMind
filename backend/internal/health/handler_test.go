package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct {
	err error
}

func (p fakePinger) Ping(context.Context) error {
	return p.err
}

func newHealthTestRouter(postgres Pinger, redis Pinger, redisConfigured bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterRoutes(router, api, postgres, redis, redisConfigured)
	return router
}

func TestLivezReturnsOK(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, nil, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	router.ServeHTTP(w, req)

	assertHealthStatus(t, w, http.StatusOK, true, "ok")
}

func TestHealthzReturnsOK(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, nil, false)

	for _, path := range []string{"/healthz", "/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)

			assertHealthStatus(t, w, http.StatusOK, true, "ok")
		})
	}
}

func TestReadyzReturnsOKWhenPostgresAndRedisOK(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, fakePinger{}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	router.ServeHTTP(w, req)

	assertHealthStatus(t, w, http.StatusOK, true, "ok")
	data := readinessData(t, w)
	if data.Status != "ready" {
		t.Fatalf("status = %q, want ready", data.Status)
	}
	if data.Dependencies["postgres"].Status != StatusUp {
		t.Fatalf("postgres status = %q, want up", data.Dependencies["postgres"].Status)
	}
	if data.Dependencies["redis"].Status != StatusUp {
		t.Fatalf("redis status = %q, want up", data.Dependencies["redis"].Status)
	}
}

func TestReadyzReturnsServiceUnavailableWhenPostgresFails(t *testing.T) {
	router := newHealthTestRouter(fakePinger{err: errors.New("postgres://user:password@internal/stylemind")}, fakePinger{}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	router.ServeHTTP(w, req)

	assertHealthStatus(t, w, http.StatusServiceUnavailable, false, "not ready")
	data := readinessData(t, w)
	if data.Dependencies["postgres"].Status != StatusDown {
		t.Fatalf("postgres status = %q, want down", data.Dependencies["postgres"].Status)
	}
	assertNoSensitiveHealthOutput(t, w.Body.String())
}

func TestReadyzReturnsServiceUnavailableWhenRedisConfiguredAndFails(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, fakePinger{err: errors.New("redis://:secret@redis:6379/0")}, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	router.ServeHTTP(w, req)

	assertHealthStatus(t, w, http.StatusServiceUnavailable, false, "not ready")
	data := readinessData(t, w)
	if data.Dependencies["redis"].Status != StatusDown {
		t.Fatalf("redis status = %q, want down", data.Dependencies["redis"].Status)
	}
	assertNoSensitiveHealthOutput(t, w.Body.String())
}

func TestReadyzSkipsRedisWhenNotConfigured(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, fakePinger{err: errors.New("redis unavailable")}, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	router.ServeHTTP(w, req)

	assertHealthStatus(t, w, http.StatusOK, true, "ok")
	data := readinessData(t, w)
	if data.Dependencies["redis"].Status != StatusSkipped {
		t.Fatalf("redis status = %q, want skipped", data.Dependencies["redis"].Status)
	}
}

func TestHealthEndpointsDoNotRequireAuth(t *testing.T) {
	router := newHealthTestRouter(fakePinger{}, fakePinger{}, true)

	for _, path := range []string{"/livez", "/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			router.ServeHTTP(w, req)

			if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
				t.Fatalf("%s unexpectedly required auth: status=%d", path, w.Code)
			}
		})
	}
}

func assertHealthStatus(t *testing.T, w *httptest.ResponseRecorder, status int, success bool, message string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, status, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != success {
		t.Fatalf("success = %v, want %v", body["success"], success)
	}
	if body["message"] != message {
		t.Fatalf("message = %v, want %s", body["message"], message)
	}
}

func readinessData(t *testing.T, w *httptest.ResponseRecorder) ReadinessResponse {
	t.Helper()

	var body struct {
		Data ReadinessResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	return body.Data
}

func assertNoSensitiveHealthOutput(t *testing.T, output string) {
	t.Helper()
	for _, forbidden := range []string{"password", "secret", "postgres://", "redis://", "@internal", "redis:6379"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("health output leaked sensitive value %q: %s", forbidden, output)
		}
	}
}
