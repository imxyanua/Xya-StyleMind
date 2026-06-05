package category

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeCategoryService struct{}

func (s *fakeCategoryService) List(context.Context, int, int) ([]Category, int64, error) {
	return nil, 0, nil
}
func (s *fakeCategoryService) Create(_ context.Context, req CreateCategoryRequest) (*Category, error) {
	return &Category{ID: uuid.NewString(), Name: req.Name, Slug: req.Slug, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}

type recordedCategoryAudit struct {
	action       string
	resourceType string
	resourceID   string
	result       string
	metadata     map[string]any
}

type fakeCategoryAuditRecorder struct {
	events []recordedCategoryAudit
}

func (r *fakeCategoryAuditRecorder) RecordAdmin(_ *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	r.events = append(r.events, recordedCategoryAudit{action: action, resourceType: resourceType, resourceID: resourceID, result: result, metadata: metadata})
}

func TestHandlerCreateCategory_WritesAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &fakeCategoryAuditRecorder{}
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	RegisterRoutes(api, admin, &fakeCategoryService{}, recorder)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/categories", strings.NewReader(`{"name":"Audit Category","slug":"audit-category"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.category.create" || event.resourceType != "category" || event.resourceID == "" || event.result != "success" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["category_name"] != "Audit Category" || event.metadata["slug"] != "audit-category" {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}
