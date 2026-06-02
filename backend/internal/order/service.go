package order

import (
	"context"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo OrderRepository
}

type OrderRepository interface {
	GetOrCreateCart(ctx context.Context, userID string) (string, error)
	CreateOrderFromCart(ctx context.Context, userID, cartID string) (string, error)
	GetOrderByIDForUser(ctx context.Context, orderID, userID string) (*OrderResponse, error)
	ListOrdersByUser(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error)
	ListOrders(ctx context.Context, limit, offset int) ([]OrderResponse, int64, error)
	UpdateOrderStatus(ctx context.Context, orderID, status string, allowedCurrentStatuses []string) error
	GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error)
}

func NewService(repo OrderRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Checkout(ctx context.Context, userID string) (*OrderResponse, error) {
	cartID, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	orderID, err := s.repo.CreateOrderFromCart(ctx, userID, cartID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetOrderByIDForUser(ctx, orderID, userID)
}

func (s *Service) ListMyOrders(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error) {
	return s.repo.ListOrdersByUser(ctx, userID, limit, offset)
}

func (s *Service) ListOrders(ctx context.Context, limit, offset int) ([]OrderResponse, int64, error) {
	return s.repo.ListOrders(ctx, limit, offset)
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
