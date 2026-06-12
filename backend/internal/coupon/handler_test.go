package coupon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"stylemind/internal/auth"
	"stylemind/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newCouponTestRouter(repo *fakeCouponRepo, secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	tokenConfig := auth.TokenConfig{
		Secret:   secret,
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
	}
	jwtAuth := middleware.JWTAuth(tokenConfig)
	admin.Use(jwtAuth, middleware.RequireRole("admin"))
	RegisterRoutes(api, admin, jwtAuth, NewService(repo))
	return router
}

func TestApplyCouponRequiresAuth(t *testing.T) {
	router := newCouponTestRouter(&fakeCouponRepo{}, "secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart/apply-coupon", bytes.NewBufferString(`{"code":"SAVE20"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertCouponError(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestAdminCouponsUserForbidden(t *testing.T) {
	router := newCouponTestRouter(&fakeCouponRepo{}, "secret")
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/coupons", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertCouponError(t, w, http.StatusForbidden, "forbidden")
}

func TestAdminCreateCouponAllowed(t *testing.T) {
	repo := &fakeCouponRepo{}
	router := newCouponTestRouter(repo, "secret")
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/coupons", bytes.NewBufferString(`{"code":"save20","type":"percent","value":20,"min_order_amount":100000,"is_active":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
	if repo.created == nil || repo.created.Code != "SAVE20" {
		t.Fatalf("created = %+v, want normalized SAVE20", repo.created)
	}
}

func TestAdminUpdateInvalidCouponID(t *testing.T) {
	router := newCouponTestRouter(&fakeCouponRepo{}, "secret")
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/coupons/bad-id", bytes.NewBufferString(`{"code":"save20","type":"percent","value":20}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertCouponError(t, w, http.StatusBadRequest, "invalid coupon id")
}

func TestAdminDeleteCouponAllowed(t *testing.T) {
	repo := &fakeCouponRepo{}
	router := newCouponTestRouter(repo, "secret")
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	couponID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/coupons/"+couponID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if repo.deletedID != couponID {
		t.Fatalf("deletedID = %s, want %s", repo.deletedID, couponID)
	}
}

func assertCouponError(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, status, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false || body["message"] != message {
		t.Fatalf("body = %+v, want success=false message=%s", body, message)
	}
}
