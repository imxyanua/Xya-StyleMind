package product

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

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Product, int64, error) {
	if filter.CategoryID != "" {
		if _, err := uuid.Parse(filter.CategoryID); err != nil {
			return nil, 0, errs.ErrInvalidID
		}
	}
	return s.repo.List(ctx, filter, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, id string) (*Product, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	if _, err := uuid.Parse(req.CategoryID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.Create(ctx, req)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	if _, err := uuid.Parse(req.CategoryID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errs.ErrInvalidID
	}
	return s.repo.Delete(ctx, id)
}
