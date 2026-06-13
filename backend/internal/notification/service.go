package notification

import (
	"context"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, input CreateInput) (*Notification, error)
	GetPreferences(ctx context.Context, userID string) (*Preferences, error)
	UpdatePreferences(ctx context.Context, userID string, input UpdatePreferencesInput) (*Preferences, error)
	ListByUser(ctx context.Context, userID string, filter ListFilter, limit, offset int) ([]Notification, int64, error)
	MarkRead(ctx context.Context, userID, id string) (*Notification, error)
	MarkAllRead(ctx context.Context, userID string) (int64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Notification, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.Title = strings.TrimSpace(input.Title)
	input.Message = strings.TrimSpace(input.Message)
	if input.UserID == "" || input.Type == "" || input.Title == "" || input.Message == "" {
		return nil, errs.ErrValidationFailed
	}
	if len(input.Type) > 100 || len(input.Title) > 255 || len(input.Message) > 2000 {
		return nil, errs.ErrValidationFailed
	}
	prefs, err := s.store.GetPreferences(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if !notificationAllowed(input.Type, prefs) {
		return nil, nil
	}
	input.Metadata = sanitizeMetadata(input.Metadata)
	return s.store.Create(ctx, input)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	if userID == "" {
		return nil, errs.ErrUnauthorized
	}
	return s.store.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, input UpdatePreferencesInput) (*Preferences, error) {
	if userID == "" {
		return nil, errs.ErrUnauthorized
	}
	return s.store.UpdatePreferences(ctx, userID, input)
}

func (s *Service) ListMine(ctx context.Context, userID string, filter ListFilter, limit, offset int) ([]Notification, int64, error) {
	return s.store.ListByUser(ctx, userID, filter, limit, offset)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (*Notification, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.store.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	return s.store.MarkAllRead(ctx, userID)
}

func sanitizeMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		normalized := strings.ToLower(strings.TrimSpace(key))
		switch normalized {
		case "password", "password_hash", "token", "access_token", "refresh_token", "authorization", "secret":
			continue
		default:
			out[key] = value
		}
	}
	return out
}

func notificationAllowed(notificationType string, prefs *Preferences) bool {
	if prefs == nil {
		return true
	}
	switch notificationType {
	case TypeOrderCreated, TypeOrderStatusUpdated:
		return prefs.OrderUpdatesEnabled
	case TypePaymentStatusUpdated:
		return prefs.PaymentUpdatesEnabled
	case TypeReturnRequestApproved, TypeReturnRequestRejected:
		return prefs.ReturnUpdatesEnabled
	default:
		if strings.HasPrefix(notificationType, "promotion.") || strings.HasPrefix(notificationType, "coupon.") {
			return prefs.PromotionEnabled
		}
		return true
	}
}
