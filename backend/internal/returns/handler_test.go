package returns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/internal/middleware"
	"stylemind/internal/notification"
	"stylemind/internal/order"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeStore struct {
	createErr       error
	listErr         error
	getErr          error
	updateErr       error
	items           []Request
	current         *Request
	lastUserID      string
	lastOrderID     string
	lastReason      string
	lastStatus      string
	lastAdminNote   string
	lastAdminFilter AdminFilter
}

func (s *fakeStore) Create(_ context.Context, userID, orderID, reason string) (*Request, error) {
	s.lastUserID = userID
	s.lastOrderID = orderID
	s.lastReason = reason
	if s.createErr != nil {
		return nil, s.createErr
	}
	return sampleReturnRequest(userID, orderID, StatusRequested), nil
}

func (s *fakeStore) ListByUser(_ context.Context, userID string, _, _ int) ([]Request, int64, error) {
	s.lastUserID = userID
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.items, int64(len(s.items)), nil
}

func (s *fakeStore) ListAdmin(_ context.Context, filter AdminFilter, _, _ int) ([]Request, int64, error) {
	s.lastAdminFilter = filter
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.items, int64(len(s.items)), nil
}

func (s *fakeStore) GetAdmin(_ context.Context, id string) (*Request, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.current != nil {
		return s.current, nil
	}
	item := sampleReturnRequest("user-1", uuid.NewString(), StatusRequested)
	item.ID = id
	return item, nil
}

func (s *fakeStore) UpdateStatus(_ context.Context, id, status, adminNote string) (*Request, error) {
	s.lastStatus = status
	s.lastAdminNote = adminNote
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	item := sampleReturnRequest("user-1", uuid.NewString(), status)
	item.ID = id
	item.AdminNote = adminNote
	if status == StatusApproved {
		item.Order.PaymentStatus = order.PaymentStatusRefunded
	}
	return item, nil
}

func sampleReturnRequest(userID, orderID, status string) *Request {
	return &Request{
		ID:      uuid.NewString(),
		OrderID: orderID,
		UserID:  userID,
		Reason:  "Size does not fit and fabric feels different than expected.",
		Status:  status,
		Order: &OrderInfo{
			ID:            orderID,
			Status:        order.StatusPaid,
			PaymentStatus: order.PaymentStatusPaid,
			PaymentMethod: "demo_payment",
			TotalAmount:   250000,
			CreatedAt:     time.Now(),
		},
		User:      &UserInfo{ID: userID, Email: "buyer@example.com", FullName: "Buyer", Role: "user"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

type recordedReturnAudit struct {
	action       string
	resourceType string
	resourceID   string
	result       string
	metadata     map[string]any
}

type fakeAuditRecorder struct {
	events []recordedReturnAudit
}

func (r *fakeAuditRecorder) RecordAdmin(_ *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	r.events = append(r.events, recordedReturnAudit{action: action, resourceType: resourceType, resourceID: resourceID, result: result, metadata: metadata})
}

type fakeReturnNotifier struct {
	events []notification.CreateInput
}

func (n *fakeReturnNotifier) Create(_ context.Context, input notification.CreateInput) (*notification.Notification, error) {
	n.events = append(n.events, input)
	return &notification.Notification{ID: uuid.NewString(), UserID: input.UserID, Type: input.Type, Title: input.Title, Message: input.Message, Metadata: input.Metadata}, nil
}

func newReturnRouter(store *fakeStore, recorders ...*fakeAuditRecorder) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-return-test")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	if len(recorders) > 0 {
		RegisterRoutes(api, admin, authMiddleware, NewService(store), recorders[0])
	} else {
		RegisterRoutes(api, admin, authMiddleware, NewService(store))
	}
	return router
}

func newReturnRouterWithNotifier(store *fakeStore, recorder *fakeAuditRecorder, notifier *fakeReturnNotifier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "req-return-test")
		c.Next()
	})
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, admin, authMiddleware, NewService(store), recorder, notifier)
	return router
}

func newProtectedReturnRouter(store *fakeStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	jwtAuth := middleware.JWTAuth(returnTokenConfig())
	admin.Use(jwtAuth, middleware.RequireRole("admin"))
	RegisterRoutes(api, admin, jwtAuth, NewService(store))
	return router
}

func TestCreateReturnRequestSuccess(t *testing.T) {
	store := &fakeStore{}
	router := newReturnRouter(store)
	orderID := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+orderID+"/return-requests", bytes.NewBufferString(`{"reason":"The jacket size does not fit me well."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 body=%s", w.Code, w.Body.String())
	}
	if store.lastUserID != "user-1" || store.lastOrderID != orderID {
		t.Fatalf("store scope = user:%s order:%s", store.lastUserID, store.lastOrderID)
	}
}

func TestCreateReturnRequestValidation(t *testing.T) {
	router := newReturnRouter(&fakeStore{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/return-requests", bytes.NewBufferString(`{"reason":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusBadRequest, "validation failed")
}

func TestCreateReturnRequestRejectsOtherUserOrder(t *testing.T) {
	router := newReturnRouter(&fakeStore{createErr: errs.ErrOrderNotFound})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/return-requests", bytes.NewBufferString(`{"reason":"The order belongs to another user."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusNotFound, "order not found")
}

func TestCreateReturnRequestRejectsIneligibleOrder(t *testing.T) {
	router := newReturnRouter(&fakeStore{createErr: errs.ErrReturnRequestNotAllowed})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/return-requests", bytes.NewBufferString(`{"reason":"I want to return this pending order."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusBadRequest, "order is not eligible for return")
}

func TestCreateReturnRequestRejectsDuplicateActive(t *testing.T) {
	router := newReturnRouter(&fakeStore{createErr: errs.ErrReturnRequestAlreadyExists})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/return-requests", bytes.NewBufferString(`{"reason":"The same order already has a return request."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusConflict, "active return request already exists")
}

func TestProtectedReturnRoutesRequireToken(t *testing.T) {
	router := newProtectedReturnRouter(&fakeStore{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/return-requests", nil)
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestAdminApproveReturnRequestWritesAudit(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	store := &fakeStore{}
	router := newReturnRouter(store, recorder)
	id := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/return-requests/"+id+"/status", bytes.NewBufferString(`{"status":"approved","admin_note":"Refund approved."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if store.lastStatus != StatusApproved || store.lastAdminNote != "Refund approved." {
		t.Fatalf("update = status:%s note:%s", store.lastStatus, store.lastAdminNote)
	}
	if len(recorder.events) != 1 {
		t.Fatalf("audit events = %d, want 1", len(recorder.events))
	}
	event := recorder.events[0]
	if event.action != "admin.return_request.approve" || event.resourceType != "return_request" || event.result != "success" {
		t.Fatalf("event = %+v", event)
	}
	if event.metadata["new_payment_status"] != order.PaymentStatusRefunded {
		t.Fatalf("metadata = %+v", event.metadata)
	}
}

func TestAdminApproveReturnRequestWritesNotification(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	notifier := &fakeReturnNotifier{}
	store := &fakeStore{}
	router := newReturnRouterWithNotifier(store, recorder, notifier)
	id := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/return-requests/"+id+"/status", bytes.NewBufferString(`{"status":"approved","admin_note":"Refund approved."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if len(notifier.events) != 2 {
		t.Fatalf("notifications = %d, want 2", len(notifier.events))
	}
	event := notifier.events[0]
	if event.UserID != "user-1" || event.Type != notification.TypeReturnRequestApproved || event.Metadata["order_id"] == "" {
		t.Fatalf("notification = %+v, want return approved event", event)
	}
	paymentEvent := notifier.events[1]
	if paymentEvent.Type != notification.TypePaymentStatusUpdated || paymentEvent.Metadata["new_payment_status"] != order.PaymentStatusRefunded {
		t.Fatalf("notification = %+v, want payment status event", paymentEvent)
	}
}

func TestAdminRejectReturnRequestWritesAudit(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	store := &fakeStore{}
	router := newReturnRouter(store, recorder)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/return-requests/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"rejected","admin_note":"Item condition is not eligible."}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%s", w.Code, w.Body.String())
	}
	if len(recorder.events) != 1 || recorder.events[0].action != "admin.return_request.reject" {
		t.Fatalf("audit events = %+v", recorder.events)
	}
}

func TestAdminApproveReturnRequestFailedWritesAudit(t *testing.T) {
	recorder := &fakeAuditRecorder{}
	store := &fakeStore{updateErr: errs.ErrInvalidPaymentStatusTransition}
	router := newReturnRouter(store, recorder)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/admin/return-requests/"+uuid.NewString()+"/status", bytes.NewBufferString(`{"status":"approved"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusBadRequest, "invalid payment status transition")
	if len(recorder.events) != 1 || recorder.events[0].result != "failed" {
		t.Fatalf("audit events = %+v", recorder.events)
	}
	if recorder.events[0].metadata["reason"] != "invalid_status_transition" {
		t.Fatalf("metadata = %+v", recorder.events[0].metadata)
	}
}

func TestAdminReturnRoutesUserForbidden(t *testing.T) {
	router := newProtectedReturnRouter(&fakeStore{})
	token, err := auth.GenerateToken(returnTokenConfig(), "user-1", "user")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/return-requests", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assertReturnError(t, w, http.StatusForbidden, "forbidden")
}

func TestServiceUpdateStatusInvalidStatus(t *testing.T) {
	service := NewService(&fakeStore{})
	_, err := service.UpdateStatus(context.Background(), uuid.NewString(), UpdateStatusRequest{Status: StatusRequested})
	if !errors.Is(err, errs.ErrInvalidReturnRequestStatus) {
		t.Fatalf("err = %v, want ErrInvalidReturnRequestStatus", err)
	}
}

func assertReturnError(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if w.Code != wantStatus {
		t.Fatalf("status = %d, want %d body=%s", w.Code, wantStatus, w.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body.Success || body.Message != wantMessage {
		t.Fatalf("body = %+v, want success false message %q", body, wantMessage)
	}
}

func returnTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{Secret: "test-secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
}
