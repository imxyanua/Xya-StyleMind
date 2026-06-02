package cart

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"
)

func TestServiceAddItem_InvalidQuantity(t *testing.T) {
	service := NewService(nil)

	_, err := service.AddItem(context.Background(), "user-id", AddCartItemRequest{
		ProductID: "bad-id",
		Quantity:  0,
	})
	if !errors.Is(err, errs.ErrInvalidQuantity) {
		t.Fatalf("err = %v, want ErrInvalidQuantity", err)
	}
}

func TestServiceAddItem_InvalidProductID(t *testing.T) {
	service := NewService(nil)

	_, err := service.AddItem(context.Background(), "user-id", AddCartItemRequest{
		ProductID: "bad-id",
		Quantity:  1,
	})
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdateItem_InvalidQuantity(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateItem(context.Background(), "user-id", "bad-id", 0)
	if !errors.Is(err, errs.ErrInvalidQuantity) {
		t.Fatalf("err = %v, want ErrInvalidQuantity", err)
	}
}

func TestServiceUpdateItem_InvalidItemID(t *testing.T) {
	service := NewService(nil)

	_, err := service.UpdateItem(context.Background(), "user-id", "bad-id", 1)
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceDeleteItem_InvalidItemID(t *testing.T) {
	service := NewService(nil)

	_, err := service.DeleteItem(context.Background(), "user-id", "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}
