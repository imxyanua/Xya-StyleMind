package category

import "context"

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Category, int64, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	return s.repo.Create(ctx, req)
}
