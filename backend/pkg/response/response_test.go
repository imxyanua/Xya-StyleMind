package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccessWithMetaFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/ok", func(c *gin.Context) {
		SuccessWithMeta(c, http.StatusOK, "ok", gin.H{"id": "p1"}, gin.H{"page": 1})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if body["success"] != true {
		t.Fatalf("success = %v, want true", body["success"])
	}
	if body["message"] != "ok" {
		t.Fatalf("message = %v, want ok", body["message"])
	}
	if _, ok := body["data"]; !ok {
		t.Fatal("data missing")
	}
	if _, ok := body["meta"]; !ok {
		t.Fatal("meta missing")
	}
}

func TestErrorFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/err", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, "bad request")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	r.ServeHTTP(w, req)

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}

	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != "bad request" {
		t.Fatalf("message = %v, want bad request", body["message"])
	}
	if _, ok := body["error"]; ok {
		t.Fatal("error field should not exist in standardized response")
	}
}
