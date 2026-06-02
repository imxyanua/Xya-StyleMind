package order

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type fakeOrderRepository struct {
	cartID                 string
	orderID                string
	getOrCreateCartErr     error
	createOrderErr         error
	getOrderForUserErr     error
	getOrderErr            error
	listOrdersErr          error
	listAllOrdersErr       error
	updateStatusErr        error
	currentStatus          string
	lastUserID             string
	lastCartID             string
	lastOrderID            string
	lastStatus             string
	lastAllowedStatuses    []string
	listLimit              int
	listOffset             int
	listAllLimit           int
	listAllOffset          int
	getOrderForUserCalled  bool
	updateOrderStatusCalls int
}

func (r *fakeOrderRepository) GetOrCreateCart(_ context.Context, userID string) (string, error) {
	r.lastUserID = userID
	if r.getOrCreateCartErr != nil {
		return "", r.getOrCreateCartErr
	}
	if r.cartID == "" {
		r.cartID = "cart-1"
	}
	return r.cartID, nil
}

func (r *fakeOrderRepository) CreateOrderFromCart(_ context.Context, userID, cartID string) (string, error) {
	r.lastUserID = userID
	r.lastCartID = cartID
	if r.createOrderErr != nil {
		return "", r.createOrderErr
	}
	if r.orderID == "" {
		r.orderID = uuid.NewString()
	}
	return r.orderID, nil
}

func (r *fakeOrderRepository) GetOrderByIDForUser(_ context.Context, orderID, userID string) (*OrderResponse, error) {
	r.getOrderForUserCalled = true
	r.lastOrderID = orderID
	r.lastUserID = userID
	if r.getOrderForUserErr != nil {
		return nil, r.getOrderForUserErr
	}
	return &OrderResponse{ID: orderID, UserID: userID, Status: StatusPending, Items: []OrderItem{}}, nil
}

func (r *fakeOrderRepository) ListOrdersByUser(_ context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error) {
	r.lastUserID = userID
	r.listLimit = limit
	r.listOffset = offset
	if r.listOrdersErr != nil {
		return nil, 0, r.listOrdersErr
	}
	return []OrderResponse{{ID: "order-1", UserID: userID, Items: []OrderItem{}}}, 1, nil
}

func (r *fakeOrderRepository) ListOrders(_ context.Context, limit, offset int) ([]OrderResponse, int64, error) {
	r.listAllLimit = limit
	r.listAllOffset = offset
	if r.listAllOrdersErr != nil {
		return nil, 0, r.listAllOrdersErr
	}
	return []OrderResponse{{ID: "admin-order-1", UserID: "user-1", Items: []OrderItem{}}}, 1, nil
}

func (r *fakeOrderRepository) UpdateOrderStatus(_ context.Context, orderID, status string, allowedCurrentStatuses []string) error {
	r.updateOrderStatusCalls++
	r.lastOrderID = orderID
	r.lastStatus = status
	r.lastAllowedStatuses = append([]string(nil), allowedCurrentStatuses...)
	return r.updateStatusErr
}

func (r *fakeOrderRepository) GetOrderByID(_ context.Context, orderID string) (*OrderResponse, error) {
	r.lastOrderID = orderID
	if r.getOrderErr != nil {
		return nil, r.getOrderErr
	}
	status := r.lastStatus
	if status == "" {
		status = r.currentStatus
	}
	if status == "" {
		status = StatusPending
	}
	return &OrderResponse{ID: orderID, Status: status, Items: []OrderItem{}}, nil
}

func TestServiceCheckout_Success(t *testing.T) {
	orderID := uuid.NewString()
	repo := &fakeOrderRepository{cartID: "cart-1", orderID: orderID}
	service := NewService(repo)

	order, err := service.Checkout(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Checkout error = %v", err)
	}
	if order.ID != orderID || order.UserID != "user-1" {
		t.Fatalf("order = %+v, want id=%s user=user-1", order, orderID)
	}
	if repo.lastCartID != "cart-1" {
		t.Fatalf("lastCartID = %q, want cart-1", repo.lastCartID)
	}
	if !repo.getOrderForUserCalled {
		t.Fatal("GetOrderByIDForUser was not called")
	}
}

func TestServiceCheckout_EmptyCart(t *testing.T) {
	repo := &fakeOrderRepository{createOrderErr: errs.ErrCartEmpty}
	service := NewService(repo)

	_, err := service.Checkout(context.Background(), "user-1")
	if !errors.Is(err, errs.ErrCartEmpty) {
		t.Fatalf("err = %v, want ErrCartEmpty", err)
	}
}

func TestServiceListMyOrders_UsesAuthenticatedUserScope(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)

	orders, total, err := service.ListMyOrders(context.Background(), "user-1", 20, 40)
	if err != nil {
		t.Fatalf("ListMyOrders error = %v", err)
	}
	if total != 1 || len(orders) != 1 {
		t.Fatalf("orders len/total = %d/%d, want 1/1", len(orders), total)
	}
	if repo.lastUserID != "user-1" || repo.listLimit != 20 || repo.listOffset != 40 {
		t.Fatalf("repo scope = user=%s limit=%d offset=%d", repo.lastUserID, repo.listLimit, repo.listOffset)
	}
}

func TestServiceListOrders_AdminList(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)

	orders, total, err := service.ListOrders(context.Background(), 50, 100)
	if err != nil {
		t.Fatalf("ListOrders error = %v", err)
	}
	if total != 1 || len(orders) != 1 {
		t.Fatalf("orders len/total = %d/%d, want 1/1", len(orders), total)
	}
	if repo.listAllLimit != 50 || repo.listAllOffset != 100 {
		t.Fatalf("repo list all pagination = limit:%d offset:%d, want 50/100", repo.listAllLimit, repo.listAllOffset)
	}
}

func TestServiceGetMyOrder_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.GetMyOrder(context.Background(), "user-id", "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceGetMyOrder_UsesAuthenticatedUserScope(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)
	orderID := uuid.NewString()

	order, err := service.GetMyOrder(context.Background(), "user-1", orderID)
	if err != nil {
		t.Fatalf("GetMyOrder error = %v", err)
	}
	if order.ID != orderID || repo.lastUserID != "user-1" {
		t.Fatalf("order/repo = %+v/%s, want scoped user order", order, repo.lastUserID)
	}
}

func TestServiceGetOrder_AdminGetInvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.GetOrder(context.Background(), "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceGetOrder_AdminGet(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)
	orderID := uuid.NewString()

	order, err := service.GetOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("GetOrder error = %v", err)
	}
	if order.ID != orderID || repo.lastOrderID != orderID {
		t.Fatalf("order/repo = %+v/%s, want order id %s", order, repo.lastOrderID, orderID)
	}
}

func TestServiceUpdateStatus_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateStatus(context.Background(), "bad-id", StatusPending)
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdateStatus_InvalidStatus(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateStatus(context.Background(), uuid.NewString(), "shipped")
	if !errors.Is(err, errs.ErrInvalidOrderStatus) {
		t.Fatalf("err = %v, want ErrInvalidOrderStatus", err)
	}
}

func TestServiceUpdateStatus_PassesAllowedTransitionsToRepository(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)
	orderID := uuid.NewString()

	order, err := service.UpdateStatus(context.Background(), orderID, StatusShipping)
	if err != nil {
		t.Fatalf("UpdateStatus error = %v", err)
	}
	if order.Status != StatusShipping {
		t.Fatalf("order.Status = %q, want %q", order.Status, StatusShipping)
	}
	if repo.updateOrderStatusCalls != 1 {
		t.Fatalf("update calls = %d, want 1", repo.updateOrderStatusCalls)
	}
	if len(repo.lastAllowedStatuses) != 2 || repo.lastAllowedStatuses[0] != StatusPaid || repo.lastAllowedStatuses[1] != StatusShipping {
		t.Fatalf("allowed statuses = %+v, want [paid shipping]", repo.lastAllowedStatuses)
	}
}

func TestServiceUpdateStatus_InvalidTransition(t *testing.T) {
	repo := &fakeOrderRepository{updateStatusErr: errs.ErrInvalidOrderStatusTransition}
	service := NewService(repo)

	_, err := service.UpdateStatus(context.Background(), uuid.NewString(), StatusPaid)
	if !errors.Is(err, errs.ErrInvalidOrderStatusTransition) {
		t.Fatalf("err = %v, want ErrInvalidOrderStatusTransition", err)
	}
}
