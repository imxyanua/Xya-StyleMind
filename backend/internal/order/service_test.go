package order

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

func TestServiceGetMyOrder_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.GetMyOrder(context.Background(), "user-id", "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdateStatus_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateStatus(context.Background(), "bad-id", StatusPending)
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdateStatus_InvalidStatus(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateStatus(context.Background(), uuid.NewString(), "shipped")
	if !errors.Is(err, errs.ErrInvalidOrderStatus) {
		t.Fatalf("err = %v, want ErrInvalidOrderStatus", err)
	}
}
