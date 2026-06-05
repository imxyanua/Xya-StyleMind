package product

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

type fakeProductService struct {
	created   *Product
	updated   *Product
	deleteErr error
}

func (s *fakeProductService) List(context.Context, ListFilter, int, int) ([]Product, int64, error) {
	return nil, 0, nil
}
func (s *fakeProductService) GetByID(context.Context, string) (*Product, error) { return nil, nil }
func (s *fakeProductService) Create(_ context.Context, req CreateProductRequest) (*Product, error) {
	if s.created != nil {
		return s.created, nil
	}
	return &Product{ID: uuid.NewString(), Name: req.Name, CategoryID: req.CategoryID, Price: req.Price, Stock: req.Stock, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (s *fakeProductService) Update(_ context.Context, id string, req UpdateProductRequest) (*Product, error) {
	if s.updated != nil {
		return s.updated, nil
	}
	return &Product{ID: id, Name: req.Name, CategoryID: req.CategoryID, Price: req.Price, Stock: req.Stock, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
}
func (s *fakeProductService) Delete(context.Context, string) error { return s.deleteErr }

type recordedProductAudit struct {
	action       string
	resourceType string
	resourceID   string
	result       string
	metadata     map[string]any
}

type fakeProductAuditRecorder struct {
	events []recordedProductAudit
}

func (r *fakeProductAuditRecorder) RecordAdmin(_ *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	r.events = append(r.events, recordedProductAudit{action: action, resourceType: resourceType, resourceID: resourceID, result: result, metadata: metadata})
}

func TestHandlerCreateProduct_WritesAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &fakeProductAuditRecorder{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.NewString())
		c.Set("user_role", "admin")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	RegisterRoutes(api, admin, &fakeProductService{}, recorder)

	categoryID := uuid.NewString()
	body := `{"name":"Audit Jacket","description":"Audit product description","price":1200000,"stock":5,"category_id":"` + categoryID + `","style":"formal","color":"black","image_url":"https://example.com/image.jpg"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.product.create" || event.resourceType != "product" || event.result != "success" || event.resourceID == "" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["product_name"] != "Audit Jacket" || event.metadata["stock"] != 5 {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}

func TestHandlerDeleteProduct_WritesAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &fakeProductAuditRecorder{}
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	RegisterRoutes(api, admin, &fakeProductService{}, recorder)

	productID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/products/"+productID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 || recorder.events[0].action != "admin.product.delete" || recorder.events[0].resourceID != productID || recorder.events[0].result != "success" {
		t.Fatalf("events = %+v", recorder.events)
	}
}
