package order

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/internal/middleware"
	"stylemind/internal/notification"
	"stylemind/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func newOrderTestRouter(repo *fakeOrderRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-order-audit-1")
		c.Next()
	})
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

func newProtectedOrderTestRouter(repo *fakeOrderRepository, secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-order-audit-1")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	tokenConfig := orderTestTokenConfig()
	tokenConfig.Secret = secret
	jwtAuth := middleware.JWTAuth(tokenConfig)
	admin.Use(jwtAuth, middleware.RequireRole("admin"))
	RegisterRoutes(api, admin, jwtAuth, NewService(repo))
	return router
}

func newProtectedOrderTestRouterWithRecorder(repo *fakeOrderRepository, secret string, recorder *fakeOrderAuditRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-order-audit-1")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	tokenConfig := orderTestTokenConfig()
	tokenConfig.Secret = secret
	jwtAuth := middleware.JWTAuth(tokenConfig)
	admin.Use(jwtAuth, middleware.RequireRole("admin"))
	RegisterRoutes(api, admin, jwtAuth, NewService(repo), recorder)
	return router
}

func newOrderTestRouterWithNotifier(repo *fakeOrderRepository, notifier *fakeOrderNotifier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-order-audit-1")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, admin, authMiddleware, NewService(repo), notifier)
	return router
}

func newProtectedOrderTestRouterWithNotifier(repo *fakeOrderRepository, secret string, notifier *fakeOrderNotifier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-order-audit-1")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	tokenConfig := orderTestTokenConfig()
	tokenConfig.Secret = secret
	jwtAuth := middleware.JWTAuth(tokenConfig)
	admin.Use(jwtAuth, middleware.RequireRole("admin"))
	RegisterRoutes(api, admin, jwtAuth, NewService(repo), notifier)
	return router
}

type recordedOrderAudit struct {
	action       string
	resourceType string
	resourceID   string
	result       string
	metadata     map[string]any
}

type fakeOrderAuditRecorder struct {
	events []recordedOrderAudit
}

func (r *fakeOrderAuditRecorder) RecordAdmin(_ *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	r.events = append(r.events, recordedOrderAudit{action: action, resourceType: resourceType, resourceID: resourceID, result: result, metadata: metadata})
}

type fakeOrderNotifier struct {
	events []notification.CreateInput
}

func (n *fakeOrderNotifier) Create(_ context.Context, input notification.CreateInput) (*notification.Notification, error) {
	n.events = append(n.events, input)
	return &notification.Notification{ID: uuid.NewString(), UserID: input.UserID, Type: input.Type, Title: input.Title, Message: input.Message, Metadata: input.Metadata}, nil
}

func validCheckoutBody() *bytes.Buffer {
	return bytes.NewBufferString(`{
		"recipient_name":"Nguyen Van A",
		"phone":"0901234567",
		"address_line":"123 Nguyen Trai",
		"city":"Ho Chi Minh City",
		"district":"District 1",
		"note":"Call before delivery",
		"shipping_method":"standard",
		"payment_method":"cod"
	}`)
}

func TestHandlerCheckout_MissingPayload(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
}

func TestHandlerCheckout_MissingAddressValidation(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBufferString(`{"recipient_name":"A","payment_method":"cod"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "validation failed")
}

func TestHandlerCheckout_EmptyCart(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{createOrderErr: errs.ErrCartEmpty})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", validCheckoutBody())
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "cart is empty")
}

func TestHandlerCheckout_InvalidCoupon(t *testing.T) {
	router := newOrderTestRouter(&fakeOrderRepository{createOrderErr: errs.ErrCouponNotFound})

	body := bytes.NewBufferString(`{
		"recipient_name":"Nguyen Van A",
		"phone":"0901234567",
		"address_line":"123 Nguyen Trai",
		"city":"Ho Chi Minh City",
		"district":"District 1",
		"shipping_method":"standard",
		"payment_method":"cod",
		"coupon_code":"MISSING"
	}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusNotFound, "coupon not found")
}

func TestHandlerCheckout_PropagatesRequestDeadlineToRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeOrderRepository{}
	router := gin.New()
	router.Use(middleware.RequestTimeout(time.Second))
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, admin, authMiddleware, NewService(repo))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", validCheckoutBody())
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
	if !repo.contextHadDeadline {
		t.Fatal("repository did not receive request context deadline")
	}
}

func TestHandlerCheckout_WritesNotification(t *testing.T) {
	orderID := uuid.NewString()
	notifier := &fakeOrderNotifier{}
	router := newOrderTestRouterWithNotifier(&fakeOrderRepository{orderID: orderID}, notifier)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", validCheckoutBody())
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
	if len(notifier.events) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.events))
	}
	event := notifier.events[0]
	if event.UserID != "user-1" || event.Type != notification.TypeOrderCreated || event.Metadata["order_id"] != orderID {
		t.Fatalf("notification = %+v, want order.created for user/order", event)
	}
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

func TestProtectedOrderRoutes_RequireToken(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedOrderDetail_RequireToken(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+uuid.NewString(), nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedOrderRoutes_MissingBearerPrefix(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedOrderRoutes_WrongSignature(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	wrongConfig := orderTestTokenConfig()
	wrongConfig.Secret = "other-secret"
	token, err := auth.GenerateToken(wrongConfig, "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedOrderRoutes_ExpiredToken(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: "user-1",
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    orderTestTokenConfig().Issuer,
			Subject:   "user-1",
			Audience:  jwt.ClaimStrings{orderTestTokenConfig().Audience},
			ID:        "token-id",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	})
	tokenString, err := token.SignedString([]byte(orderTestTokenConfig().Secret))
	if err != nil {
		t.Fatalf("SignedString error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedAdminOrderStatus_UserForbidden(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusForbidden, "forbidden")
}

func TestProtectedAdminOrderPaymentStatus_UserForbidden(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+uuid.NewString()+"/payment-status", bytes.NewBufferString(`{"payment_status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusForbidden, "forbidden")
}

func TestProtectedAdminOrdersList_UserForbidden(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusForbidden, "forbidden")
}

func TestProtectedAdminOrdersList_RequireToken(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders", nil)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedAdminOrderPaymentStatus_RequireToken(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+uuid.NewString()+"/payment-status", bytes.NewBufferString(`{"payment_status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestProtectedAdminOrdersList_AdminAllowed(t *testing.T) {
	repo := &fakeOrderRepository{}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders?page=2&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.listAllLimit != 10 || repo.listAllOffset != 10 {
		t.Fatalf("repo list all pagination = limit:%d offset:%d, want 10/10", repo.listAllLimit, repo.listAllOffset)
	}
}

func TestProtectedAdminOrdersList_FiltersAndSorts(t *testing.T) {
	repo := &fakeOrderRepository{}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	userID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders?q=abc&status=paid&user_id="+userID+"&from=2026-01-01&to=2026-01-31&sort=oldest&page=2&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastAdminFilter.Query != "abc" || repo.lastAdminFilter.Status != StatusPaid || repo.lastAdminFilter.UserID != userID || repo.lastAdminFilter.Sort != AdminOrderSortOldest {
		t.Fatalf("admin filter = %+v, want q/status/user/sort", repo.lastAdminFilter)
	}
	if repo.lastAdminFilter.From == nil || repo.lastAdminFilter.To == nil {
		t.Fatalf("admin date filter = %+v, want from/to", repo.lastAdminFilter)
	}
	if strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "hash") || strings.Contains(w.Body.String(), "token") {
		t.Fatalf("admin order response leaked sensitive field: %s", w.Body.String())
	}
}

func TestProtectedAdminOrdersList_InvalidFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		query   string
		message string
	}{
		{name: "status", query: "?status=shipped", message: "invalid status"},
		{name: "user_id", query: "?user_id=bad-id", message: "invalid user_id"},
		{name: "sort", query: "?sort=price_desc", message: "invalid sort"},
		{name: "date_range", query: "?from=2026-02-01&to=2026-01-01", message: "validation failed"},
		{name: "from", query: "?from=not-a-date", message: "invalid from"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")
			token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
			if err != nil {
				t.Fatalf("GenerateToken error = %v", err)
			}

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders"+tc.query, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			router.ServeHTTP(w, req)

			assertOrderErrorResponse(t, w, http.StatusBadRequest, tc.message)
		})
	}
}

func TestProtectedAdminOrderDetail_AdminAllowed(t *testing.T) {
	repo := &fakeOrderRepository{}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/"+orderID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastOrderID != orderID {
		t.Fatalf("repo lastOrderID = %s, want %s", repo.lastOrderID, orderID)
	}
}

func TestProtectedAdminOrderDetail_InvalidID(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders/bad-id", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid order id")
}

func TestProtectedAdminOrderStatus_AdminAllowed(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	repo := &fakeOrderRepository{currentStatus: StatusPending}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastOrderID != orderID || repo.lastStatus != StatusPaid {
		t.Fatalf("repo update = order:%s status:%s, want %s/%s", repo.lastOrderID, repo.lastStatus, orderID, StatusPaid)
	}
	assertOrderAuditEvent(t, audit.String(), map[string]any{
		"type":       "audit",
		"event":      "admin.order_status.update",
		"result":     "success",
		"user_id":    "admin-1",
		"role":       "admin",
		"order_id":   orderID,
		"old_status": StatusPending,
		"new_status": StatusPaid,
		"request_id": "req-order-audit-1",
	})
	assertOrderAuditDoesNotContain(t, audit.String(), token, "Authorization", "Bearer")
}

func TestProtectedAdminOrderStatus_WritesNotification(t *testing.T) {
	notifier := &fakeOrderNotifier{}
	repo := &fakeOrderRepository{currentStatus: StatusPending}
	router := newProtectedOrderTestRouterWithNotifier(repo, "test-secret", notifier)

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if len(notifier.events) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.events))
	}
	event := notifier.events[0]
	if event.Type != notification.TypeOrderStatusUpdated || event.UserID != "user-1" || event.Metadata["new_status"] != StatusPaid {
		t.Fatalf("notification = %+v, want order status update", event)
	}
}

func TestProtectedAdminOrderStatus_PatchAllowed(t *testing.T) {
	repo := &fakeOrderRepository{currentStatus: StatusPending}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastStatus != StatusPaid {
		t.Fatalf("repo lastStatus = %s, want paid", repo.lastStatus)
	}
}

func TestProtectedAdminOrderPaymentStatus_AdminAllowed(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	repo := &fakeOrderRepository{currentPaymentStatus: PaymentStatusUnpaid}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/payment-status", bytes.NewBufferString(`{"payment_status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if repo.lastOrderID != orderID || repo.lastPaymentStatus != PaymentStatusPaid {
		t.Fatalf("repo update = order:%s payment:%s, want %s/%s", repo.lastOrderID, repo.lastPaymentStatus, orderID, PaymentStatusPaid)
	}
	assertOrderAuditEvent(t, audit.String(), map[string]any{
		"type":               "audit",
		"event":              "admin.order_payment_status.update",
		"result":             "success",
		"user_id":            "admin-1",
		"role":               "admin",
		"order_id":           orderID,
		"old_payment_status": PaymentStatusUnpaid,
		"new_payment_status": PaymentStatusPaid,
		"request_id":         "req-order-audit-1",
	})
	assertOrderAuditDoesNotContain(t, audit.String(), token, "Authorization", "Bearer")
}

func TestProtectedAdminOrderPaymentStatus_InvalidStatus(t *testing.T) {
	router := newProtectedOrderTestRouter(&fakeOrderRepository{}, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+uuid.NewString()+"/payment-status", bytes.NewBufferString(`{"payment_status":"captured"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid payment status")
}

func TestProtectedAdminOrderPaymentStatusFailed_WritesPersistentAudit(t *testing.T) {
	repo := &fakeOrderRepository{currentPaymentStatus: PaymentStatusPaid, updatePaymentStatusErr: errs.ErrInvalidPaymentStatusTransition}
	recorder := &fakeOrderAuditRecorder{}
	router := newProtectedOrderTestRouterWithRecorder(repo, "test-secret", recorder)

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/payment-status", bytes.NewBufferString(`{"payment_status":"failed"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid payment status transition")
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.order_payment_status.update" || event.resourceType != "order" || event.resourceID != orderID || event.result != "failed" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["old_payment_status"] != PaymentStatusPaid ||
		event.metadata["new_payment_status"] != PaymentStatusFailed ||
		event.metadata["reason"] != "invalid_status_transition" {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}

func TestProtectedAdminOrderStatus_WritesPersistentAudit(t *testing.T) {
	repo := &fakeOrderRepository{currentStatus: StatusPending}
	recorder := &fakeOrderAuditRecorder{}
	router := newProtectedOrderTestRouterWithRecorder(repo, "test-secret", recorder)

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.order_status.update" || event.resourceType != "order" || event.resourceID != orderID || event.result != "success" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["old_status"] != StatusPending || event.metadata["new_status"] != StatusPaid {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}

func TestProtectedAdminOrderStatusFailed_WritesPersistentAudit(t *testing.T) {
	repo := &fakeOrderRepository{currentStatus: StatusCompleted, updateStatusErr: errs.ErrInvalidOrderStatusTransition}
	recorder := &fakeOrderAuditRecorder{}
	router := newProtectedOrderTestRouterWithRecorder(repo, "test-secret", recorder)

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid order status transition")
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.order_status.update" || event.resourceType != "order" || event.resourceID != orderID || event.result != "failed" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["reason"] != "invalid_status_transition" {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}

func TestProtectedAdminOrderStatusFailed_WritesSafeAuditReason(t *testing.T) {
	var audit bytes.Buffer
	restore := logger.SetAuditOutput(&audit)
	defer restore()

	repo := &fakeOrderRepository{currentStatus: StatusCompleted, updateStatusErr: errs.ErrInvalidOrderStatusTransition}
	router := newProtectedOrderTestRouter(repo, "test-secret")

	token, err := auth.GenerateToken(orderTestTokenConfig(), "admin-1", "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	orderID := uuid.NewString()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/orders/"+orderID+"/status", bytes.NewBufferString(`{"status":"paid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertOrderErrorResponse(t, w, http.StatusBadRequest, "invalid order status transition")
	assertOrderAuditEvent(t, audit.String(), map[string]any{
		"event":      "admin.order_status.update",
		"result":     "failed",
		"user_id":    "admin-1",
		"role":       "admin",
		"order_id":   orderID,
		"old_status": StatusCompleted,
		"new_status": StatusPaid,
		"reason":     "invalid_status_transition",
	})
	assertOrderAuditDoesNotContain(t, audit.String(), token, "Authorization", "Bearer")
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

func assertOrderAuditEvent(t *testing.T, raw string, expected map[string]any) {
	t.Helper()

	entries := strings.Split(strings.TrimSpace(raw), "\n")
	if len(entries) == 0 || entries[0] == "" {
		t.Fatal("expected audit log entry, got empty output")
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(entries[len(entries)-1]), &entry); err != nil {
		t.Fatalf("audit json unmarshal error = %v, raw=%s", err, raw)
	}
	for key, want := range expected {
		if got := entry[key]; got != want {
			t.Fatalf("audit[%s] = %v, want %v; entry=%+v", key, got, want, entry)
		}
	}
}

func assertOrderAuditDoesNotContain(t *testing.T, raw string, forbidden ...string) {
	t.Helper()

	for _, value := range forbidden {
		if strings.Contains(raw, value) {
			t.Fatalf("audit output contained forbidden value %q: %s", value, raw)
		}
	}
}

func orderTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		Secret:   "test-secret",
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
	}
}
