package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestTimeout_AllowsRequestWithinTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTimeout(time.Second))
	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/fast", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func TestRequestTimeout_ReturnsGatewayTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTimeout(10 * time.Millisecond))
	router.GET("/slow", func(c *gin.Context) {
		<-c.Request.Context().Done()
		time.Sleep(10 * time.Millisecond)
		_, _ = c.Writer.WriteString("late body")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusGatewayTimeout)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v, body=%s", err, w.Body.String())
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != "request timeout" {
		t.Fatalf("message = %v, want request timeout", body["message"])
	}
	if w.Body.String() != `{"success":false,"message":"request timeout"}` {
		t.Fatalf("body leaked late output or wrong format: %s", w.Body.String())
	}
}

func TestRequestTimeout_AddsDeadlineToRequestContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestTimeout(time.Second))
	deadlineSeen := false
	backgroundContextSeen := false
	router.GET("/deadline", func(c *gin.Context) {
		_, deadlineSeen = c.Request.Context().Deadline()
		backgroundContextSeen = c.Request.Context() == context.Background()
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/deadline", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
	if !deadlineSeen {
		t.Fatal("request context did not have deadline")
	}
	if backgroundContextSeen {
		t.Fatal("request context unexpectedly used context.Background")
	}
}
