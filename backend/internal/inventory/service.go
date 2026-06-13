package inventory

import "context"

type Store interface {
	ListActiveByUser(ctx context.Context, userID string, limit, offset int) ([]Reservation, int64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListMine(ctx context.Context, userID string, limit, offset int) ([]Reservation, int64, error) {
	return s.store.ListActiveByUser(ctx, userID, limit, offset)
}
