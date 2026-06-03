package wishlist

import (
	"context"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo WishlistRepository
}

type WishlistRepository interface {
	ProductExists(ctx context.Context, productID string) (bool, error)
	AddProduct(ctx context.Context, userID, productID string) error
	RemoveProduct(ctx context.Context, userID, productID string) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]WishlistItem, int64, error)
}

func NewService(repo WishlistRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) AddProduct(ctx context.Context, userID, productID string) error {
	if _, err := uuid.Parse(productID); err != nil {
		return errs.ErrInvalidID
	}
	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return err
	}
	if !exists {
		return errs.ErrProductNotFound
	}
	return s.repo.AddProduct(ctx, userID, productID)
}

func (s *Service) RemoveProduct(ctx context.Context, userID, productID string) error {
	if _, err := uuid.Parse(productID); err != nil {
		return errs.ErrInvalidID
	}
	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return err
	}
	if !exists {
		return errs.ErrProductNotFound
	}
	return s.repo.RemoveProduct(ctx, userID, productID)
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]WishlistItem, int64, error) {
	return s.repo.ListByUser(ctx, userID, limit, offset)
}
