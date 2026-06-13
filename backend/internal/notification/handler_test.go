package notification

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"stylemind/internal/auth"
	"stylemind/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newNotificationTestRouter(store *fakeNotificationStore, secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	tokenConfig := auth.TokenConfig{Secret: secret, Issuer: "stylemind-api", Audience: "stylemind-web"}
	RegisterRoutes(api, middleware.JWTAuth(tokenConfig), NewService(store))
	return router
}

func TestNotificationRoutesRequireAuth(t *testing.T) {
	router := newNotificationTestRouter(&fakeNotificationStore{}, "secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body=%s", w.Code, w.Body.String())
	}
}

func TestListNotificationsReturnsOnlyCurrentUser(t *testing.T) {
	store := &fakeNotificationStore{items: []Notification{
		{ID: uuid.NewString(), UserID: "user-1", Type: TypeOrderCreated, Title: "A", Message: "A"},
		{ID: uuid.NewString(), UserID: "user-2", Type: TypeOrderCreated, Title: "B", Message: "B"},
	}}
	router := newNotificationTestRouter(store, "secret")
	token := notificationTestToken(t, "user-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications?unread=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !containsAll(body, `"user_id":"user-1"`, `"meta"`) || containsAll(body, `"user_id":"user-2"`) {
		t.Fatalf("body did not scope notifications to current user: %s", body)
	}
}

func TestMarkReadAndReadAll(t *testing.T) {
	notificationID := uuid.NewString()
	store := &fakeNotificationStore{items: []Notification{{ID: notificationID, UserID: "user-1", Type: TypeOrderCreated, Title: "A", Message: "A"}}}
	router := newNotificationTestRouter(store, "secret")
	token := notificationTestToken(t, "user-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/notifications/"+notificationID+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("mark read status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me/notifications/read-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("read all status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if store.markAllUser != "user-1" {
		t.Fatalf("markAllUser = %s, want user-1", store.markAllUser)
	}
}

func notificationTestToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := auth.GenerateToken(auth.TokenConfig{Secret: "secret", Issuer: "stylemind-api", Audience: "stylemind-web"}, userID, "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}
	return token
}

func containsAll(raw string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(raw, value) {
			return false
		}
	}
	return true
}
