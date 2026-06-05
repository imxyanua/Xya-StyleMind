package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
)

type fakeAuditStore struct {
	created []CreateLogParams
	items   []Log
	total   int64
	filter  ListFilter
}

func (s *fakeAuditStore) Create(_ context.Context, params CreateLogParams) (*Log, error) {
	s.created = append(s.created, params)
	return &Log{ID: "audit-1", Action: params.Action, ResourceType: params.ResourceType, ResourceID: params.ResourceID, Result: params.Result, Metadata: params.Metadata}, nil
}

func (s *fakeAuditStore) List(_ context.Context, filter ListFilter, limit, offset int) ([]Log, int64, error) {
	s.filter = filter
	if s.total == 0 {
		s.total = int64(len(s.items))
	}
	return s.items, s.total, nil
}

func TestServiceRecordAdmin_SanitizesSensitiveMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeAuditStore{}
	service := NewService(store)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/products", nil)
	c.Request.Header.Set("User-Agent", "audit-test")
	c.Set("user_id", "11111111-1111-1111-1111-111111111111")
	c.Set("user_role", "admin")
	c.Set("request_id", "req-1")

	service.RecordAdmin(c, "admin.product.create", "product", "product-1", ResultSuccess, map[string]any{
		"product_name":  "Jacket",
		"password":      "never-log",
		"access_token":  "never-log",
		"authorization": "Bearer secret",
	})

	if len(store.created) != 1 {
		t.Fatalf("created audit count = %d, want 1", len(store.created))
	}
	entry := store.created[0]
	if entry.Action != "admin.product.create" || entry.ResourceType != "product" || entry.ResourceID != "product-1" || entry.Result != ResultSuccess {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.Metadata["product_name"] != "Jacket" {
		t.Fatalf("metadata = %+v, want product_name", entry.Metadata)
	}
	for _, key := range []string{"password", "access_token", "authorization"} {
		if _, ok := entry.Metadata[key]; ok {
			t.Fatalf("metadata contained sensitive key %q: %+v", key, entry.Metadata)
		}
	}
}

func TestAdminAuditLogsRoute_RequiresAdminAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeAuditStore{items: []Log{{ID: "audit-1", ActorRole: "admin", Action: "admin.order_status.update", ResourceType: "order", ResourceID: "order-1", Result: ResultSuccess, Metadata: map[string]any{"new_status": "paid"}, CreatedAt: time.Now()}}}
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	cfg := auth.TokenConfig{Secret: "test-secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
	admin.Use(middleware.JWTAuth(cfg), middleware.RequireRole("admin"))
	RegisterRoutes(admin, NewService(store))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", w.Code)
	}

	userToken, err := auth.GenerateToken(cfg, "22222222-2222-2222-2222-222222222222", "user")
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("user status = %d, want 403", w.Code)
	}

	adminToken, err := auth.GenerateToken(cfg, "33333333-3333-3333-3333-333333333333", "admin")
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?action=admin.order_status.update&resource_type=order&result=success&sort=oldest&page=2&limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if store.filter.Action != "admin.order_status.update" || store.filter.ResourceType != "order" || store.filter.Result != ResultSuccess || store.filter.Sort != SortOldest {
		t.Fatalf("filter = %+v", store.filter)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != true || body["meta"] == nil {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestAdminAuditLogsRoute_InvalidFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-1")
		c.Set("user_role", "admin")
		c.Next()
	})
	RegisterRoutes(admin, NewService(&fakeAuditStore{}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?sort=bad", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}
