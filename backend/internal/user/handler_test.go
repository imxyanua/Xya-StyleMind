package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stylemind/internal/audit"
	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
)

type fakeRepo struct {
	items      []User
	total      int64
	filter     ListFilter
	updated    *User
	oldRole    string
	updateErr  error
	updateRole string
}

func (r *fakeRepo) List(_ context.Context, filter ListFilter, _, _ int) ([]User, int64, error) {
	r.filter = filter
	if r.total == 0 {
		r.total = int64(len(r.items))
	}
	return r.items, r.total, nil
}

func (r *fakeRepo) GetByID(_ context.Context, id string) (*User, error) {
	for _, item := range r.items {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, errs.ErrUserNotFound
}

func (r *fakeRepo) UpdateRole(_ context.Context, _, targetUserID, newRole string) (*User, string, error) {
	r.updateRole = newRole
	if r.updateErr != nil {
		return nil, r.oldRole, r.updateErr
	}
	updated := r.updated
	if updated == nil {
		updated = &User{
			ID:        targetUserID,
			Email:     "target@example.com",
			FullName:  "Target User",
			Role:      newRole,
			Status:    StatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}
	return updated, r.oldRole, nil
}

type auditEvent struct {
	action       string
	resourceType string
	resourceID   string
	result       string
	metadata     map[string]any
}

type fakeAuditRecorder struct {
	events []auditEvent
}

func (r *fakeAuditRecorder) RecordAdmin(_ *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	r.events = append(r.events, auditEvent{action: action, resourceType: resourceType, resourceID: resourceID, result: result, metadata: metadata})
}

func TestAdminUsersRoutes_RequireAdmin(t *testing.T) {
	router, cfg, _ := setupRouter(&fakeRepo{}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", w.Code)
	}

	token, err := auth.GenerateToken(cfg, "22222222-2222-2222-2222-222222222222", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("user status = %d, want 403", w.Code)
	}
}

func TestAdminUsersList_SuccessFiltersAndDoesNotExposePasswordHash(t *testing.T) {
	repo := &fakeRepo{items: []User{{
		ID:        "11111111-1111-1111-1111-111111111111",
		Email:     "shopper@example.com",
		FullName:  "Shopper",
		Role:      RoleUser,
		Status:    StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}}}
	router, cfg, _ := setupRouter(repo, nil)
	token, err := auth.GenerateToken(cfg, "33333333-3333-3333-3333-333333333333", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?q=shopper&role=user&status=active&page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "hash") {
		t.Fatalf("response exposed password material: %s", w.Body.String())
	}
	if repo.filter.Query != "shopper" || repo.filter.Role != RoleUser || repo.filter.Status != StatusActive {
		t.Fatalf("filter = %+v", repo.filter)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["success"] != true || body["meta"] == nil {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestAdminUsersUpdateRole_InvalidRole(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	router, cfg, _ := setupRouter(&fakeRepo{}, recorder)
	token, err := auth.GenerateToken(cfg, "33333333-3333-3333-3333-333333333333", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/11111111-1111-1111-1111-111111111111/role", strings.NewReader(`{"role":"owner"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 || recorder.events[0].result != audit.ResultFailed {
		t.Fatalf("audit events = %+v, want one failed event", recorder.events)
	}
}

func TestAdminUsersUpdateRole_SuccessAudited(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	repo := &fakeRepo{oldRole: RoleUser}
	router, cfg, _ := setupRouter(repo, recorder)
	token, err := auth.GenerateToken(cfg, "33333333-3333-3333-3333-333333333333", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/11111111-1111-1111-1111-111111111111/role", strings.NewReader(`{"role":"admin"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if repo.updateRole != RoleAdmin {
		t.Fatalf("update role = %q, want admin", repo.updateRole)
	}
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.user_role.update" || event.resourceType != "user" || event.result != audit.ResultSuccess {
		t.Fatalf("audit event = %+v", event)
	}
	if event.metadata["old_role"] != RoleUser || event.metadata["new_role"] != RoleAdmin {
		t.Fatalf("audit metadata = %+v", event.metadata)
	}
}

func TestAdminUsersUpdateRole_LastAdminConflictAudited(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	repo := &fakeRepo{oldRole: RoleAdmin, updateErr: errs.ErrCannotDemoteLastAdmin}
	router, cfg, _ := setupRouter(repo, recorder)
	token, err := auth.GenerateToken(cfg, "33333333-3333-3333-3333-333333333333", RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/users/33333333-3333-3333-3333-333333333333/role", strings.NewReader(`{"role":"user"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 || recorder.events[0].result != audit.ResultFailed || recorder.events[0].metadata["reason"] != "last_admin" {
		t.Fatalf("audit events = %+v", recorder.events)
	}
}

func TestServiceValidation(t *testing.T) {
	service := NewService(&fakeRepo{})
	if _, _, err := service.List(context.Background(), ListFilter{Role: "owner"}, 10, 0); !errors.Is(err, errs.ErrInvalidUserRole) {
		t.Fatalf("invalid role err = %v, want ErrInvalidUserRole", err)
	}
	if _, _, err := service.List(context.Background(), ListFilter{Status: "deleted"}, 10, 0); !errors.Is(err, errs.ErrInvalidUserStatus) {
		t.Fatalf("invalid status err = %v, want ErrInvalidUserStatus", err)
	}
	if _, err := service.GetByID(context.Background(), "bad-id"); !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("invalid id err = %v, want ErrInvalidID", err)
	}
}

func setupRouter(repo *fakeRepo, recorder audit.Recorder) (*gin.Engine, auth.TokenConfig, *Service) {
	gin.SetMode(gin.TestMode)
	cfg := auth.TokenConfig{Secret: "test-secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
	service := NewService(repo)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(cfg), middleware.RequireRole(RoleAdmin))
	if recorder == nil {
		recorder = &fakeAuditRecorder{}
	}
	RegisterRoutes(admin, service, recorder)
	return router, cfg, service
}
