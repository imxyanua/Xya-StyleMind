package dashboard

import (
	"context"

	"stylemind/internal/errs"
)

type Store interface {
	GetStats(ctx context.Context, filter StatsFilter) (*Stats, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) GetStats(ctx context.Context, filter StatsFilter) (*Stats, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, errs.ErrValidationFailed
	}
	return s.store.GetStats(ctx, filter)
}
