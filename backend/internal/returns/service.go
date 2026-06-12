package returns

import (
	"context"
	"strings"
	"time"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, userID, orderID, reason string) (*Request, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]Request, int64, error)
	ListAdmin(ctx context.Context, filter AdminFilter, limit, offset int) ([]Request, int64, error)
	GetAdmin(ctx context.Context, id string) (*Request, error)
	UpdateStatus(ctx context.Context, id, status, adminNote string) (*Request, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, userID, orderID string, req CreateRequest) (*Request, error) {
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, errs.ErrInvalidID
	}
	reason := strings.TrimSpace(req.Reason)
	if len(reason) < 10 || len(reason) > 1000 {
		return nil, errs.ErrValidationFailed
	}
	return s.store.Create(ctx, userID, orderID, reason)
}

func (s *Service) ListMine(ctx context.Context, userID string, limit, offset int) ([]Request, int64, error) {
	return s.store.ListByUser(ctx, userID, limit, offset)
}

func (s *Service) ListAdmin(ctx context.Context, filter AdminFilter, limit, offset int) ([]Request, int64, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	filter.UserID = strings.TrimSpace(filter.UserID)
	filter.OrderID = strings.TrimSpace(filter.OrderID)
	filter.Sort = strings.TrimSpace(filter.Sort)
	if filter.Status != "" && !IsValidStatus(filter.Status) {
		return nil, 0, errs.ErrInvalidReturnRequestStatus
	}
	if filter.UserID != "" {
		if _, err := uuid.Parse(filter.UserID); err != nil {
			return nil, 0, errs.ErrInvalidID
		}
	}
	if filter.OrderID != "" {
		if _, err := uuid.Parse(filter.OrderID); err != nil {
			return nil, 0, errs.ErrInvalidID
		}
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, 0, errs.ErrValidationFailed
	}
	if filter.Sort != "" && filter.Sort != SortNewest && filter.Sort != SortOldest {
		return nil, 0, errs.ErrInvalidSort
	}
	return s.store.ListAdmin(ctx, filter, limit, offset)
}

func (s *Service) GetAdmin(ctx context.Context, id string) (*Request, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.store.GetAdmin(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, req UpdateStatusRequest) (*Request, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	status := strings.TrimSpace(req.Status)
	adminNote := strings.TrimSpace(req.AdminNote)
	if !IsAdminUpdateStatus(status) || len(adminNote) > 1000 {
		return nil, errs.ErrInvalidReturnRequestStatus
	}
	return s.store.UpdateStatus(ctx, id, status, adminNote)
}

func IsValidStatus(status string) bool {
	switch status {
	case StatusRequested, StatusApproved, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}

func IsAdminUpdateStatus(status string) bool {
	switch status {
	case StatusApproved, StatusRejected, StatusCancelled:
		return true
	default:
		return false
	}
}

func ParseOptionalTime(raw string, endOfDay bool) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
