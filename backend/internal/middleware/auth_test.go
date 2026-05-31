package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stylemind/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestJWTAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", JWTAuth("secret"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	secret := "secret"

	r.GET("/protected", JWTAuth(secret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString("user_id"),
			"role":    c.GetString("user_role"),
		})
	})

	token, err := auth.GenerateToken(secret, "u1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["user_id"] != "u1" {
		t.Fatalf("user_id = %v, want u1", body["user_id"])
	}
	if body["role"] != "admin" {
		t.Fatalf("role = %v, want admin", body["role"])
	}
}
