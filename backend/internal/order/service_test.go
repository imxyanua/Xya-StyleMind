package order

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type fakeOrderRepository struct {
	cartID                     string
	orderID                    string
	getOrCreateCartErr         error
	createOrderErr             error
	getOrderForUserErr         error
	getOrderErr                error
	listOrdersErr              error
	listAllOrdersErr           error
	updateStatusErr            error
	updatePaymentStatusErr     error
	currentStatus              string
	currentPaymentStatus       string
	lastUserID                 string
	lastCartID                 string
	lastCheckoutDetails        CheckoutDetails
	lastOrderID                string
	lastStatus                 string
	lastPaymentStatus          string
	lastAllowedStatuses        []string
	lastAllowedPaymentStatuses []string
	lastAdminFilter            AdminOrderFilter
	listLimit                  int
	listOffset                 int
	listAllLimit               int
	listAllOffset              int
	getOrderForUserCalled      bool
	contextHadDeadline         bool
	updateOrderStatusCalls     int
	updatePaymentStatusCalls   int
}

func (r *fakeOrderRepository) GetOrCreateCart(ctx context.Context, userID string) (string, error) {
	r.lastUserID = userID
	_, r.contextHadDeadline = ctx.Deadline()
	if r.getOrCreateCartErr != nil {
		return "", r.getOrCreateCartErr
	}
	if r.cartID == "" {
		r.cartID = "cart-1"
	}
	return r.cartID, nil
}

func (r *fakeOrderRepository) CreateOrderFromCart(_ context.Context, userID, cartID string, details CheckoutDetails) (string, error) {
	r.lastUserID = userID
	r.lastCartID = cartID
	r.lastCheckoutDetails = details
	if r.createOrderErr != nil {
		return "", r.createOrderErr
	}
	if r.orderID == "" {
		r.orderID = uuid.NewString()
	}
	return r.orderID, nil
}

func validCheckoutDetails() CheckoutDetails {
	return CheckoutDetails{
		RecipientName:  "Nguyen Van A",
		Phone:          "0901234567",
		AddressLine:    "123 Nguyen Trai",
		City:           "Ho Chi Minh City",
		District:       "District 1",
		Note:           "Call before delivery",
		ShippingMethod: "standard",
		PaymentMethod:  "cod",
	}
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

func (r *fakeOrderRepository) ListOrders(_ context.Context, filter AdminOrderFilter, limit, offset int) ([]OrderResponse, int64, error) {
	r.lastAdminFilter = filter
	r.listAllLimit = limit
	r.listAllOffset = offset
	if r.listAllOrdersErr != nil {
		return nil, 0, r.listAllOrdersErr
	}
	return []OrderResponse{{
		ID:     "admin-order-1",
		UserID: "user-1",
		User:   &OrderUser{ID: "user-1", Email: "buyer@example.com", FullName: "Buyer", Role: "user"},
		Items:  []OrderItem{},
	}}, 1, nil
}

func (r *fakeOrderRepository) UpdateOrderStatus(_ context.Context, orderID, status string, allowedCurrentStatuses []string) error {
	r.updateOrderStatusCalls++
	r.lastOrderID = orderID
	r.lastStatus = status
	r.lastAllowedStatuses = append([]string(nil), allowedCurrentStatuses...)
	return r.updateStatusErr
}

func (r *fakeOrderRepository) UpdatePaymentStatus(_ context.Context, orderID, paymentStatus string, allowedCurrentStatuses []string) error {
	r.updatePaymentStatusCalls++
	r.lastOrderID = orderID
	r.lastPaymentStatus = paymentStatus
	r.lastAllowedPaymentStatuses = append([]string(nil), allowedCurrentStatuses...)
	return r.updatePaymentStatusErr
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
	paymentStatus := r.lastPaymentStatus
	if paymentStatus == "" {
		paymentStatus = r.currentPaymentStatus
	}
	if paymentStatus == "" {
		paymentStatus = PaymentStatusUnpaid
	}
	return &OrderResponse{ID: orderID, Status: status, PaymentStatus: paymentStatus, Items: []OrderItem{}}, nil
}

func TestServiceCheckout_Success(t *testing.T) {
	orderID := uuid.NewString()
	repo := &fakeOrderRepository{cartID: "cart-1", orderID: orderID}
	service := NewService(repo)

	order, err := service.Checkout(context.Background(), "user-1", validCheckoutDetails())
	if err != nil {
		t.Fatalf("Checkout error = %v", err)
	}
	if order.ID != orderID || order.UserID != "user-1" {
		t.Fatalf("order = %+v, want id=%s user=user-1", order, orderID)
	}
	if repo.lastCartID != "cart-1" {
		t.Fatalf("lastCartID = %q, want cart-1", repo.lastCartID)
	}
	if repo.lastCheckoutDetails.PaymentMethod != "cod" || repo.lastCheckoutDetails.ShippingMethod != "standard" {
		t.Fatalf("checkout details = %+v, want cod/standard", repo.lastCheckoutDetails)
	}
	if !repo.getOrderForUserCalled {
		t.Fatal("GetOrderByIDForUser was not called")
	}
}

func TestServiceCheckout_NormalizesCouponCode(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)
	details := validCheckoutDetails()
	details.CouponCode = " save20 "

	_, err := service.Checkout(context.Background(), "user-1", details)
	if err != nil {
		t.Fatalf("Checkout error = %v", err)
	}
	if repo.lastCheckoutDetails.CouponCode != "SAVE20" {
		t.Fatalf("coupon code = %q, want SAVE20", repo.lastCheckoutDetails.CouponCode)
	}
}

func TestServiceCheckout_EmptyCart(t *testing.T) {
	repo := &fakeOrderRepository{createOrderErr: errs.ErrCartEmpty}
	service := NewService(repo)

	_, err := service.Checkout(context.Background(), "user-1", validCheckoutDetails())
	if !errors.Is(err, errs.ErrCartEmpty) {
		t.Fatalf("err = %v, want ErrCartEmpty", err)
	}
}

func TestServiceCheckout_PropagatesCouponValidationError(t *testing.T) {
	repo := &fakeOrderRepository{createOrderErr: errs.ErrCouponUsageLimitReached}
	service := NewService(repo)
	details := validCheckoutDetails()
	details.CouponCode = "LIMITED"

	_, err := service.Checkout(context.Background(), "user-1", details)
	if !errors.Is(err, errs.ErrCouponUsageLimitReached) {
		t.Fatalf("err = %v, want ErrCouponUsageLimitReached", err)
	}
}

func TestServiceCheckout_RejectsMissingShippingDetails(t *testing.T) {
	service := NewService(&fakeOrderRepository{})

	_, err := service.Checkout(context.Background(), "user-1", CheckoutDetails{PaymentMethod: "cod"})
	if !errors.Is(err, errs.ErrValidationFailed) {
		t.Fatalf("err = %v, want ErrValidationFailed", err)
	}
}

func TestServiceCheckout_RejectsInvalidPaymentOrShippingMethod(t *testing.T) {
	service := NewService(&fakeOrderRepository{})
	details := validCheckoutDetails()
	details.ShippingMethod = "drone"

	if _, err := service.Checkout(context.Background(), "user-1", details); !errors.Is(err, errs.ErrValidationFailed) {
		t.Fatalf("invalid shipping err = %v, want ErrValidationFailed", err)
	}

	details = validCheckoutDetails()
	details.PaymentMethod = "card_pan"
	if _, err := service.Checkout(context.Background(), "user-1", details); !errors.Is(err, errs.ErrValidationFailed) {
		t.Fatalf("invalid payment err = %v, want ErrValidationFailed", err)
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

	orders, total, err := service.ListOrders(context.Background(), AdminOrderFilter{Status: StatusPaid, Sort: AdminOrderSortOldest}, 50, 100)
	if err != nil {
		t.Fatalf("ListOrders error = %v", err)
	}
	if total != 1 || len(orders) != 1 {
		t.Fatalf("orders len/total = %d/%d, want 1/1", len(orders), total)
	}
	if repo.listAllLimit != 50 || repo.listAllOffset != 100 {
		t.Fatalf("repo list all pagination = limit:%d offset:%d, want 50/100", repo.listAllLimit, repo.listAllOffset)
	}
	if repo.lastAdminFilter.Status != StatusPaid || repo.lastAdminFilter.Sort != AdminOrderSortOldest {
		t.Fatalf("admin filter = %+v, want paid/oldest", repo.lastAdminFilter)
	}
}

func TestServiceListOrders_RejectsInvalidAdminFilters(t *testing.T) {
	service := NewService(&fakeOrderRepository{})

	if _, _, err := service.ListOrders(context.Background(), AdminOrderFilter{Status: "shipped"}, 20, 0); !errors.Is(err, errs.ErrInvalidOrderStatus) {
		t.Fatalf("invalid status err = %v, want ErrInvalidOrderStatus", err)
	}
	if _, _, err := service.ListOrders(context.Background(), AdminOrderFilter{UserID: "bad-id"}, 20, 0); !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("invalid user_id err = %v, want ErrInvalidID", err)
	}
	if _, _, err := service.ListOrders(context.Background(), AdminOrderFilter{Sort: "price_desc"}, 20, 0); !errors.Is(err, errs.ErrInvalidSort) {
		t.Fatalf("invalid sort err = %v, want ErrInvalidSort", err)
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

func TestServiceUpdatePaymentStatus_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdatePaymentStatus(context.Background(), "bad-id", PaymentStatusPaid)
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdatePaymentStatus_InvalidStatus(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdatePaymentStatus(context.Background(), uuid.NewString(), "captured")
	if !errors.Is(err, errs.ErrInvalidPaymentStatus) {
		t.Fatalf("err = %v, want ErrInvalidPaymentStatus", err)
	}
}

func TestServiceUpdatePaymentStatus_PassesAllowedTransitionsToRepository(t *testing.T) {
	repo := &fakeOrderRepository{}
	service := NewService(repo)
	orderID := uuid.NewString()

	order, err := service.UpdatePaymentStatus(context.Background(), orderID, PaymentStatusPaid)
	if err != nil {
		t.Fatalf("UpdatePaymentStatus error = %v", err)
	}
	if order.PaymentStatus != PaymentStatusPaid {
		t.Fatalf("order.PaymentStatus = %q, want %q", order.PaymentStatus, PaymentStatusPaid)
	}
	if repo.updatePaymentStatusCalls != 1 {
		t.Fatalf("update payment calls = %d, want 1", repo.updatePaymentStatusCalls)
	}
	want := []string{PaymentStatusUnpaid, PaymentStatusPending, PaymentStatusPaid}
	if len(repo.lastAllowedPaymentStatuses) != len(want) {
		t.Fatalf("allowed payment statuses = %+v, want %+v", repo.lastAllowedPaymentStatuses, want)
	}
	for i := range want {
		if repo.lastAllowedPaymentStatuses[i] != want[i] {
			t.Fatalf("allowed payment statuses = %+v, want %+v", repo.lastAllowedPaymentStatuses, want)
		}
	}
}

func TestServiceUpdatePaymentStatus_InvalidTransition(t *testing.T) {
	repo := &fakeOrderRepository{updatePaymentStatusErr: errs.ErrInvalidPaymentStatusTransition}
	service := NewService(repo)

	_, err := service.UpdatePaymentStatus(context.Background(), uuid.NewString(), PaymentStatusRefunded)
	if !errors.Is(err, errs.ErrInvalidPaymentStatusTransition) {
		t.Fatalf("err = %v, want ErrInvalidPaymentStatusTransition", err)
	}
}
