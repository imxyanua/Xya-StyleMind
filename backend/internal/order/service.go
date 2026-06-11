package order

import (
	"context"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo OrderRepository
}

type OrderRepository interface {
	GetOrCreateCart(ctx context.Context, userID string) (string, error)
	CreateOrderFromCart(ctx context.Context, userID, cartID string, details CheckoutDetails) (string, error)
	GetOrderByIDForUser(ctx context.Context, orderID, userID string) (*OrderResponse, error)
	ListOrdersByUser(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error)
	ListOrders(ctx context.Context, filter AdminOrderFilter, limit, offset int) ([]OrderResponse, int64, error)
	UpdateOrderStatus(ctx context.Context, orderID, status string, allowedCurrentStatuses []string) error
	GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error)
}

func NewService(repo OrderRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Checkout(ctx context.Context, userID string, details CheckoutDetails) (*OrderResponse, error) {
	details, err := normalizeCheckoutDetails(details)
	if err != nil {
		return nil, err
	}

	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	orderID, err := s.repo.CreateOrderFromCart(ctx, userID, cartID, details)
	if err != nil {
		return nil, err
	}

	return s.repo.GetOrderByIDForUser(ctx, orderID, userID)
}

func normalizeCheckoutDetails(details CheckoutDetails) (CheckoutDetails, error) {
	details.RecipientName = strings.TrimSpace(details.RecipientName)
	details.Phone = strings.TrimSpace(details.Phone)
	details.AddressLine = strings.TrimSpace(details.AddressLine)
	details.City = strings.TrimSpace(details.City)
	details.District = strings.TrimSpace(details.District)
	details.Note = strings.TrimSpace(details.Note)
	details.ShippingMethod = strings.TrimSpace(details.ShippingMethod)
	details.PaymentMethod = strings.TrimSpace(details.PaymentMethod)

	if details.RecipientName == "" ||
		details.Phone == "" ||
		details.AddressLine == "" ||
		details.City == "" ||
		details.District == "" ||
		details.ShippingMethod == "" ||
		details.PaymentMethod == "" {
		return CheckoutDetails{}, errs.ErrValidationFailed
	}
	if len(details.RecipientName) > 120 ||
		len(details.Phone) > 32 ||
		len(details.AddressLine) > 255 ||
		len(details.City) > 120 ||
		len(details.District) > 120 ||
		len(details.Note) > 1000 {
		return CheckoutDetails{}, errs.ErrValidationFailed
	}
	if details.ShippingMethod != "standard" && details.ShippingMethod != "express" {
		return CheckoutDetails{}, errs.ErrValidationFailed
	}
	if details.PaymentMethod != "cod" && details.PaymentMethod != "demo_payment" {
		return CheckoutDetails{}, errs.ErrValidationFailed
	}
	return details, nil
}

func (s *Service) ListMyOrders(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error) {
	return s.repo.ListOrdersByUser(ctx, userID, limit, offset)
}

func (s *Service) ListOrders(ctx context.Context, filter AdminOrderFilter, limit, offset int) ([]OrderResponse, int64, error) {
	if filter.Status != "" && !IsValidStatus(filter.Status) {
		return nil, 0, errs.ErrInvalidOrderStatus
	}
	if filter.UserID != "" {
		if _, err := uuid.Parse(filter.UserID); err != nil {
			return nil, 0, errs.ErrInvalidID
		}
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, 0, errs.ErrValidationFailed
	}
	if !isAllowedAdminOrderSort(filter.Sort) {
		return nil, 0, errs.ErrInvalidSort
	}
	return s.repo.ListOrders(ctx, filter, limit, offset)
}

func (s *Service) GetMyOrder(ctx context.Context, userID, orderID string) (*OrderResponse, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetOrderByIDForUser(ctx, orderID, userID)
}

func (s *Service) GetOrder(ctx context.Context, orderID string) (*OrderResponse, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetOrderByID(ctx, orderID)
}

func (s *Service) UpdateStatus(ctx context.Context, orderID, status string) (*OrderResponse, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	if !IsValidStatus(status) {
		return nil, errs.ErrInvalidOrderStatus
	}
	if err := s.repo.UpdateOrderStatus(ctx, orderID, status, AllowedCurrentStatuses(status)); err != nil {
		return nil, err
	}
	return s.repo.GetOrderByID(ctx, orderID)
}

func isAllowedAdminOrderSort(sort string) bool {
	switch sort {
	case "", AdminOrderSortNewest, AdminOrderSortOldest:
		return true
	default:
		return false
	}
}
