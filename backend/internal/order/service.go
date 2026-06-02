package order

import (
	"context"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
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

func (s *Service) GetMyOrder(ctx context.Context, userID, orderID string) (*OrderResponse, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetOrderByIDForUser(ctx, orderID, userID)
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
