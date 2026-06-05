package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
)

type fakeDashboardStore struct {
	stats  *Stats
	filter StatsFilter
	err    error
}

func (s *fakeDashboardStore) GetStats(_ context.Context, filter StatsFilter) (*Stats, error) {
	s.filter = filter
	if s.err != nil {
		return nil, s.err
	}
	if s.stats != nil {
		return s.stats, nil
	}
	return &Stats{
		TotalRevenue:  300000,
		TotalOrders:   3,
		TotalProducts: 2,
		TotalUsers:    4,
		OrdersByStatus: OrdersByStatus{
			Pending:   1,
			Paid:      1,
			Completed: 1,
		},
		RecentOrders:     []RecentOrder{{ID: "order-1", UserEmail: "buyer@example.com", Status: "paid", TotalAmount: 100000, CreatedAt: time.Now()}},
		LowStockProducts: []LowStockProduct{{ID: "product-1", Name: "Low Tee", Stock: 2, Price: 99000}},
		RevenueByDay:     []RevenueByDay{{Date: "2026-06-06", Revenue: 300000}},
		TopProducts:      []TopProduct{{ID: "product-1", Name: "Low Tee", QuantitySold: 3, Revenue: 300000}},
	}, nil
}

func TestAdminDashboardStatsRoute_RequiresAdminAndReturnsStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &fakeDashboardStore{}
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	cfg := auth.TokenConfig{Secret: "test-secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
	admin.Use(middleware.JWTAuth(cfg), middleware.RequireRole("admin"))
	RegisterRoutes(admin, NewService(store))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/stats", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", w.Code)
	}

	userToken, err := auth.GenerateToken(cfg, "22222222-2222-2222-2222-222222222222", "user")
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/stats", nil)
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
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/stats?from=2026-06-01&to=2026-06-06", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if store.filter.From == nil || store.filter.To == nil {
		t.Fatalf("filter = %+v, want from/to", store.filter)
	}
	for _, want := range []string{"\"total_revenue\":300000", "\"total_orders\":3", "\"low_stock_products\"", "\"recent_orders\""} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("body missing %s: %s", want, w.Body.String())
		}
	}
}

func TestAdminDashboardStatsRoute_InvalidDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		c.Set("user_id", "admin-1")
		c.Set("user_role", "admin")
		c.Next()
	})
	RegisterRoutes(admin, NewService(&fakeDashboardStore{}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/dashboard/stats?from=2026-06-07&to=2026-06-01", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
}

func TestServiceGetStats_InvalidRange(t *testing.T) {
	from := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := NewService(&fakeDashboardStore{}).GetStats(context.Background(), StatsFilter{From: &from, To: &to})
	if err != errs.ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed", err)
	}
}

func TestRepositoryHelpers_RevenueStatusAndDateWhere(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	where, args := buildOrderDateWhere(StatsFilter{From: &from}, "o")
	if where != " WHERE o.created_at >= $1" || len(args) != 1 {
		t.Fatalf("where/args = %q/%d", where, len(args))
	}
	revenueWhere := appendRevenueStatusWhere(where, "o")
	for _, want := range []string{"o.status IN", RevenueStatusPaid, RevenueStatusShipping, RevenueStatusCompleted} {
		if !strings.Contains(revenueWhere, want) {
			t.Fatalf("revenueWhere = %q, missing %q", revenueWhere, want)
		}
	}
}
