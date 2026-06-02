package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_AllowsRequestsWithinLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/limited", RateLimit(2, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/limited", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
	}
}

func TestRateLimit_BlocksRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/limited", RateLimit(1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	firstReq.RemoteAddr = "203.0.113.11:1234"
	r.ServeHTTP(first, firstReq)

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	secondReq.RemoteAddr = "203.0.113.11:1234"
	r.ServeHTTP(second, secondReq)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", second.Code, http.StatusTooManyRequests)
	}

	var body map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
}

func TestRateLimit_DifferentKeysDoNotAffectEachOther(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/limited", RateLimit(1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	firstReq.RemoteAddr = "203.0.113.20:1234"
	r.ServeHTTP(first, firstReq)

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/limited", nil)
	secondReq.RemoteAddr = "203.0.113.21:1234"
	r.ServeHTTP(second, secondReq)

	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d/%d, want both 200", first.Code, second.Code)
	}
}

func TestRateLimiterEmailKeyExtractorRestoresBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := NewMemoryRateLimitStore()
	limiter := NewRateLimiter(store, 10, time.Minute)
	r.POST("/login", limiter.Middleware("auth:login", EmailKeyExtractor), func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("ReadAll body error = %v", err)
		}
		c.String(http.StatusOK, string(body))
	})

	payload := `{"email":"USER@example.com","password":"secret"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != payload {
		t.Fatalf("body = %q, want original payload", w.Body.String())
	}
}

func TestRateLimiterUsesIPAndEmailKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &recordingRateLimitStore{}
	limiter := NewRateLimiter(store, 10, time.Minute)
	r := gin.New()
	r.POST("/login", limiter.Middleware("auth:login", IPKeyExtractor, EmailKeyExtractor), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"USER@example.com","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.30:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if len(store.keys) != 2 {
		t.Fatalf("keys = %+v, want 2 keys", store.keys)
	}
	if store.keys[0] != "rl:auth:login:ip:203.0.113.30" {
		t.Fatalf("ip key = %q", store.keys[0])
	}
	if store.keys[1] != "rl:auth:login:email:user@example.com" {
		t.Fatalf("email key = %q", store.keys[1])
	}
}

func TestRateLimiterUsesRequestContextDeadline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &recordingRateLimitStore{}
	limiter := NewRateLimiter(store, 10, time.Minute)
	r := gin.New()
	r.Use(RequestTimeout(time.Second))
	r.GET("/limited", limiter.Middleware("auth:login", IPKeyExtractor), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !store.contextHadDeadline {
		t.Fatal("rate limit store did not receive request context deadline")
	}
}

func TestRateLimiterFailClosedOnStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	limiter := NewRateLimiter(&failingRateLimitStore{}, 10, time.Minute, WithFailClosed(true))
	r.GET("/limited", limiter.Middleware("auth:login", IPKeyExtractor), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}
}

func TestRateLimiterFailOpenOnStoreError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	limiter := NewRateLimiter(&failingRateLimitStore{}, 10, time.Minute, WithFailClosed(false))
	r.GET("/limited", limiter.Middleware("auth:login", IPKeyExtractor), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestNewRateLimitStoreFromConfig_UsesMemoryWhenRedisMissing(t *testing.T) {
	store, redisConfigured, err := NewRateLimitStoreFromConfig("", "", "0")
	if err != nil {
		t.Fatalf("NewRateLimitStoreFromConfig error = %v", err)
	}
	defer store.Close()
	if redisConfigured {
		t.Fatal("redisConfigured = true, want false for empty REDIS_ADDR")
	}
}

func TestNewRateLimitStoreFromConfig_InvalidRedisDB(t *testing.T) {
	_, _, err := NewRateLimitStoreFromConfig("localhost:6379", "", "bad")
	if err == nil {
		t.Fatal("expected invalid redis db error, got nil")
	}
}

type failingRateLimitStore struct{}

func (s *failingRateLimitStore) Increment(context.Context, string, time.Duration) (int, error) {
	return 0, errors.New("redis unavailable")
}

func (s *failingRateLimitStore) Close() error {
	return nil
}

type recordingRateLimitStore struct {
	mu                 sync.Mutex
	keys               []string
	contextHadDeadline bool
}

func (s *recordingRateLimitStore) Increment(ctx context.Context, key string, _ time.Duration) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = append(s.keys, key)
	if _, ok := ctx.Deadline(); ok {
		s.contextHadDeadline = true
	}
	return 1, nil
}

func (s *recordingRateLimitStore) Close() error {
	return nil
}

func TestCORSConfig_UsesConfiguredOrigins(t *testing.T) {
	cfg := CORSConfig([]string{"http://localhost:3000", "https://example.com"})

	if len(cfg.AllowOrigins) != 2 {
		t.Fatalf("AllowOrigins length = %d, want 2", len(cfg.AllowOrigins))
	}
	if cfg.AllowOrigins[0] != "http://localhost:3000" {
		t.Fatalf("AllowOrigins[0] = %q, want localhost origin", cfg.AllowOrigins[0])
	}
}

func TestCORSConfig_DoesNotAllowCredentials(t *testing.T) {
	cfg := CORSConfig([]string{"*"})

	if cfg.AllowCredentials {
		t.Fatal("AllowCredentials = true, want false when wildcard origins are configured")
	}
}
