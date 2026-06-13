package notification

import (
	"context"
	"errors"
	"stylemind/internal/errs"
	"testing"

	"github.com/google/uuid"
)

type fakeNotificationStore struct {
	items       []Notification
	created     []CreateInput
	markReadID  string
	markAllUser string
}

func (s *fakeNotificationStore) Create(_ context.Context, input CreateInput) (*Notification, error) {
	s.created = append(s.created, input)
	item := Notification{
		ID:       uuid.NewString(),
		UserID:   input.UserID,
		Type:     input.Type,
		Title:    input.Title,
		Message:  input.Message,
		Metadata: input.Metadata,
	}
	return &item, nil
}

func (s *fakeNotificationStore) ListByUser(_ context.Context, userID string, filter ListFilter, limit, offset int) ([]Notification, int64, error) {
	out := make([]Notification, 0)
	for _, item := range s.items {
		if item.UserID != userID {
			continue
		}
		if filter.UnreadOnly && item.ReadAt != nil {
			continue
		}
		out = append(out, item)
	}
	return out, int64(len(out)), nil
}

func (s *fakeNotificationStore) MarkRead(_ context.Context, userID, id string) (*Notification, error) {
	s.markReadID = id
	for _, item := range s.items {
		if item.ID == id && item.UserID == userID {
			return &item, nil
		}
	}
	return nil, errs.ErrNotificationNotFound
}

func (s *fakeNotificationStore) MarkAllRead(_ context.Context, userID string) (int64, error) {
	s.markAllUser = userID
	return 3, nil
}

func TestCreateSanitizesSensitiveMetadata(t *testing.T) {
	store := &fakeNotificationStore{}
	service := NewService(store)

	_, err := service.Create(context.Background(), CreateInput{
		UserID:  "user-1",
		Type:    TypeOrderCreated,
		Title:   "Order created",
		Message: "Your order was created.",
		Metadata: map[string]any{
			"order_id":      "order-1",
			"token":         "secret-token",
			"password_hash": "hash",
		},
	})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if _, ok := store.created[0].Metadata["token"]; ok {
		t.Fatal("metadata should not contain token")
	}
	if _, ok := store.created[0].Metadata["password_hash"]; ok {
		t.Fatal("metadata should not contain password_hash")
	}
	if store.created[0].Metadata["order_id"] != "order-1" {
		t.Fatalf("metadata = %+v, want order_id preserved", store.created[0].Metadata)
	}
}

func TestListMineScopesToUser(t *testing.T) {
	store := &fakeNotificationStore{items: []Notification{
		{ID: "n1", UserID: "user-1"},
		{ID: "n2", UserID: "user-2"},
	}}
	service := NewService(store)

	items, total, err := service.ListMine(context.Background(), "user-1", ListFilter{}, 20, 0)
	if err != nil {
		t.Fatalf("ListMine error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != "user-1" {
		t.Fatalf("items/total = %+v/%d, want only user-1", items, total)
	}
}

func TestMarkReadValidatesIDAndOwnership(t *testing.T) {
	store := &fakeNotificationStore{items: []Notification{{ID: uuid.NewString(), UserID: "user-1"}}}
	service := NewService(store)

	if _, err := service.MarkRead(context.Background(), "user-1", "bad-id"); !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("MarkRead invalid id err = %v, want ErrInvalidID", err)
	}
	if _, err := service.MarkRead(context.Background(), "user-2", store.items[0].ID); !errors.Is(err, errs.ErrNotificationNotFound) {
		t.Fatalf("MarkRead other user err = %v, want ErrNotificationNotFound", err)
	}
	if _, err := service.MarkRead(context.Background(), "user-1", store.items[0].ID); err != nil {
		t.Fatalf("MarkRead owner error = %v", err)
	}
}

func TestMarkAllReadUsesAuthenticatedUser(t *testing.T) {
	store := &fakeNotificationStore{}
	service := NewService(store)

	updated, err := service.MarkAllRead(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("MarkAllRead error = %v", err)
	}
	if updated != 3 || store.markAllUser != "user-1" {
		t.Fatalf("updated/user = %d/%s, want 3/user-1", updated, store.markAllUser)
	}
}
