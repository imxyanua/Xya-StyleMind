package review

import (
	"context"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Service struct {
	repo ReviewRepository
}

type ReviewRepository interface {
	ProductExists(ctx context.Context, productID string) (bool, error)
	HasPurchasedProduct(ctx context.Context, userID, productID, orderID string) (bool, error)
	Create(ctx context.Context, userID, productID string, req CreateReviewRequest) (*Review, error)
	ListByProduct(ctx context.Context, productID string, limit, offset int) ([]Review, int64, error)
	SummaryByProduct(ctx context.Context, productID string) (*RatingSummary, error)
	GetByID(ctx context.Context, reviewID string) (*Review, error)
	Update(ctx context.Context, reviewID string, req UpdateReviewRequest) (*Review, error)
	Delete(ctx context.Context, reviewID string) error
}

func NewService(repo ReviewRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID, productID string, req CreateReviewRequest) (*Review, error) {
	if err := validateProductID(productID); err != nil {
		return nil, err
	}
	if _, err := uuid.Parse(req.OrderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	if err := validateRating(req.Rating); err != nil {
		return nil, err
	}
	if err := validateComment(req.Comment); err != nil {
		return nil, err
	}

	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errs.ErrProductNotFound
	}
	purchased, err := s.repo.HasPurchasedProduct(ctx, userID, productID, req.OrderID)
	if err != nil {
		return nil, err
	}
	if !purchased {
		return nil, errs.ErrProductNotPurchased
	}
	return s.repo.Create(ctx, userID, productID, req)
}

func (s *Service) ListByProduct(ctx context.Context, productID string, limit, offset int) ([]Review, int64, error) {
	if err := validateProductID(productID); err != nil {
		return nil, 0, err
	}
	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, errs.ErrProductNotFound
	}
	return s.repo.ListByProduct(ctx, productID, limit, offset)
}

func (s *Service) SummaryByProduct(ctx context.Context, productID string) (*RatingSummary, error) {
	if err := validateProductID(productID); err != nil {
		return nil, err
	}
	exists, err := s.repo.ProductExists(ctx, productID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errs.ErrProductNotFound
	}
	return s.repo.SummaryByProduct(ctx, productID)
}

func (s *Service) Update(ctx context.Context, userID, reviewID string, req UpdateReviewRequest) (*Review, error) {
	if _, err := uuid.Parse(reviewID); err != nil {
		return nil, errs.ErrInvalidID
	}
	if err := validateRating(req.Rating); err != nil {
		return nil, err
	}
	if err := validateComment(req.Comment); err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	if current.UserID != userID {
		return nil, errs.ErrForbidden
	}
	return s.repo.Update(ctx, reviewID, req)
}

func (s *Service) Delete(ctx context.Context, userID, reviewID string) error {
	if _, err := uuid.Parse(reviewID); err != nil {
		return errs.ErrInvalidID
	}
	current, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if current.UserID != userID {
		return errs.ErrForbidden
	}
	return s.repo.Delete(ctx, reviewID)
}

func validateProductID(productID string) error {
	if _, err := uuid.Parse(productID); err != nil {
		return errs.ErrInvalidID
	}
	return nil
}

func validateRating(rating int) error {
	if rating < 1 || rating > 5 {
		return errs.ErrValidationFailed
	}
	return nil
}

func validateComment(comment *string) error {
	if comment != nil && len(*comment) > 1000 {
		return errs.ErrValidationFailed
	}
	return nil
}
