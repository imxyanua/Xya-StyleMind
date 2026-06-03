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

func TestServiceList_InvalidFilters(t *testing.T) {
	service := NewService(nil)
	minPrice := 200.0
	maxPrice := 100.0
	minRating := 6.0

	tests := []struct {
		name   string
		filter ListFilter
		want   error
	}{
		{name: "invalid sort", filter: ListFilter{Sort: "bad_sort"}, want: errs.ErrInvalidSort},
		{name: "price range", filter: ListFilter{MinPrice: &minPrice, MaxPrice: &maxPrice, Sort: SortNewest}, want: errs.ErrValidationFailed},
		{name: "min rating", filter: ListFilter{MinRating: &minRating, Sort: SortNewest}, want: errs.ErrValidationFailed},
		{name: "invalid category id with valid sort", filter: ListFilter{CategoryID: "bad-id", Sort: SortNewest}, want: errs.ErrInvalidID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := service.List(context.Background(), tt.filter, 20, 0)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
		})
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
