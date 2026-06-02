package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stylemind/internal/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newOrderTestRouter(repo *fakeOrderRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, admin, authMiddleware, NewService(repo))
	return router
}

func TestHandlerCheckout_EmptyCart(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{createOrderErr: errs.ErrCartEmpty})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "cart is empty")
}

func TestHandlerListMine_ReturnsPaginationMeta(t *testing.T) {
	repo := &fakeOrderRepository{}
	router := newOrderTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders?page=2&limit=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastUserID != "user-1" || repo.listLimit != 10 || repo.listOffset != 10 {
		t.Fatalf("repo scope = user=%s limit=%d offset=%d, want user-1/10/10", repo.lastUserID, repo.listLimit, repo.listOffset)
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != true {
		t.Fatalf("success = %v, want true", body["success"])
	}
	if _, ok := body["meta"]; !ok {
		t.Fatal("response missing pagination meta")
	}
}

func TestHandlerGetMine_InvalidID(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/bad-id", nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid order id")
}

func TestHandlerGetMine_NotFound(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{getOrderForUserErr: errs.ErrOrderNotFound})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+uuid.NewString(), nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusNotFound, "order not found")
}

func TestHandlerUpdateStatus_InvalidPayload(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+uuid.NewString()+"/status", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
}

func TestHandlerUpdateStatus_ValidationFailed(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"shipped"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "validation failed")
}

func TestHandlerUpdateStatus_InvalidTransition(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{updateStatusErr: errs.ErrInvalidOrderStatusTransition})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid order status transition")
}

func assertOrderErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()

	if w.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, status, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != message {
		t.Fatalf("message = %v, want %s", body["message"], message)
	}
	if _, ok := body["data"]; ok {
		t.Fatal("error response should not include data")
	}
}
