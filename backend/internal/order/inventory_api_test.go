package order

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/inventory"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
)

type fakeInventoryStore struct {
	items      []inventory.Reservation
	lastUserID string
}

func (s *fakeInventoryStore) ListActiveByUser(_ context.Context, userID string, _, _ int) ([]inventory.Reservation, int64, error) {
	s.lastUserID = userID
	out := make([]inventory.Reservation, 0)
	for _, item := range s.items {
		if item.UserID == userID && item.ExpiresAt.After(time.Now()) {
			out = append(out, item)
		}
	}
	return out, int64(len(out)), nil
}

func newInventoryTestRouter(store *fakeInventoryStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	tokenConfig := auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
	inventory.RegisterRoutes(api, middleware.JWTAuth(tokenConfig), inventory.NewService(store))
	return router
}

func TestInventoryReservationsRouteRequiresAuth(t *testing.T) {
	router := newInventoryTestRouter(&fakeInventoryStore{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reservations", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 body=%s", w.Code, w.Body.String())
	}
}

func TestInventoryReservationsRouteScopesCurrentUserAndActiveReservations(t *testing.T) {
	store := &fakeInventoryStore{items: []inventory.Reservation{
		{ID: "reservation-1", UserID: "user-1", ProductID: "product-1", Quantity: 2, ExpiresAt: time.Now().Add(time.Minute)},
		{ID: "reservation-2", UserID: "user-2", ProductID: "product-2", Quantity: 1, ExpiresAt: time.Now().Add(time.Minute)},
		{ID: "reservation-3", UserID: "user-1", ProductID: "product-3", Quantity: 1, ExpiresAt: time.Now().Add(-time.Minute)},
	}}
	router := newInventoryTestRouter(store)
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/reservations", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if store.lastUserID != "user-1" || !strings.Contains(body, "reservation-1") || strings.Contains(body, "reservation-2") || strings.Contains(body, "reservation-3") {
		t.Fatalf("body/user = %s/%s, want only active current user reservation", body, store.lastUserID)
	}
}
