package product

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"
)

func TestServiceList_InvalidCategoryID(t *testing.T) {
	service := NewService(nil)

	_, _, err := service.List(context.Background(), ListFilter{CategoryID: "bad-id"}, 20, 0)
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceGetByID_InvalidID(t *testing.T) {
	service := NewService(nil)

	_, err := service.GetByID(context.Background(), "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceCreate_InvalidCategoryID(t *testing.T) {
	service := NewService(nil)

	_, err := service.Create(context.Background(), CreateProductRequest{CategoryID: "bad-id"})
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceUpdate_InvalidIDs(t *testing.T) {
	service := NewService(nil)

	_, err := service.Update(context.Background(), "bad-id", UpdateProductRequest{CategoryID: "bad-id"})
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceDelete_InvalidID(t *testing.T) {
	service := NewService(nil)

	err := service.Delete(context.Background(), "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}
