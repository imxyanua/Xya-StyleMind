package coupon

import (
	"context"
	"errors"
	"stylemind/internal/errs"
	"testing"
	"time"
)

type fakeCouponRepo struct {
	cartID       string
	subtotal     float64
	coupon       *Coupon
	getByCodeErr error
	created      *Coupon
	updated      *Coupon
	deletedID    string
}

func (r *fakeCouponRepo) GetOrCreateCartID(context.Context, string) (string, error) {
	if r.cartID == "" {
		return "cart-1", nil
	}
	return r.cartID, nil
}

func (r *fakeCouponRepo) GetCartSubtotal(context.Context, string) (float64, error) {
	if r.subtotal <= 0 {
		return 0, errs.ErrCartEmpty
	}
	return r.subtotal, nil
}

func (r *fakeCouponRepo) GetByCode(context.Context, string) (*Coupon, error) {
	if r.getByCodeErr != nil {
		return nil, r.getByCodeErr
	}
	return r.coupon, nil
}

func (r *fakeCouponRepo) List(context.Context, ListFilter, int, int) ([]Coupon, int64, error) {
	return nil, 0, nil
}

func (r *fakeCouponRepo) GetByID(context.Context, string) (*Coupon, error) {
	return r.coupon, nil
}

func (r *fakeCouponRepo) Create(_ context.Context, req MutationRequest) (*Coupon, error) {
	r.created = &Coupon{ID: "coupon-1", Code: req.Code, Type: req.Type, Value: req.Value, MinOrderAmount: req.MinOrderAmount}
	return r.created, nil
}

func (r *fakeCouponRepo) Update(_ context.Context, id string, req MutationRequest) (*Coupon, error) {
	r.updated = &Coupon{ID: id, Code: req.Code, Type: req.Type, Value: req.Value, IsActive: true}
	return r.updated, nil
}

func (r *fakeCouponRepo) Delete(_ context.Context, id string) error {
	r.deletedID = id
	return nil
}

func TestApplyToCartPercentCoupon(t *testing.T) {
	repo := &fakeCouponRepo{
		subtotal: 200000,
		coupon:   activeCoupon(TypePercent, 20),
	}
	service := NewService(repo)
	service.now = func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }

	result, err := service.ApplyToCart(context.Background(), "user-1", " save20 ")
	if err != nil {
		t.Fatalf("ApplyToCart error = %v", err)
	}
	if result.CouponCode != "SAVE20" || result.DiscountAmount != 40000 || result.TotalAmount != 160000 {
		t.Fatalf("result = %+v, want SAVE20 discount 40000 total 160000", result)
	}
}

func TestApplyToCartFixedCouponCapsAtSubtotal(t *testing.T) {
	repo := &fakeCouponRepo{
		subtotal: 50000,
		coupon:   activeCoupon(TypeFixed, 100000),
	}
	service := NewService(repo)

	result, err := service.ApplyToCart(context.Background(), "user-1", "fixed")
	if err != nil {
		t.Fatalf("ApplyToCart error = %v", err)
	}
	if result.DiscountAmount != 50000 || result.TotalAmount != 0 {
		t.Fatalf("result = %+v, want discount capped at subtotal", result)
	}
}

func TestApplyToCartRejectsInvalidCouponStates(t *testing.T) {
	now := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	expiredAt := now.Add(-time.Hour)
	futureStart := now.Add(time.Hour)
	limit := 1

	tests := []struct {
		name   string
		coupon Coupon
		want   error
	}{
		{name: "inactive", coupon: Coupon{Code: "OFF", Type: TypeFixed, Value: 10000, IsActive: false}, want: errs.ErrCouponInactive},
		{name: "expired", coupon: Coupon{Code: "OFF", Type: TypeFixed, Value: 10000, IsActive: true, ExpiresAt: &expiredAt}, want: errs.ErrCouponExpired},
		{name: "not_started", coupon: Coupon{Code: "OFF", Type: TypeFixed, Value: 10000, IsActive: true, StartsAt: &futureStart}, want: errs.ErrCouponExpired},
		{name: "usage_limit", coupon: Coupon{Code: "OFF", Type: TypeFixed, Value: 10000, IsActive: true, UsageLimit: &limit, UsedCount: 1}, want: errs.ErrCouponUsageLimitReached},
		{name: "min_order", coupon: Coupon{Code: "OFF", Type: TypeFixed, Value: 10000, IsActive: true, MinOrderAmount: 300000}, want: errs.ErrCouponMinOrderNotMet},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCouponRepo{subtotal: 200000, coupon: &tc.coupon}
			service := NewService(repo)
			service.now = func() time.Time { return now }
			_, err := service.ApplyToCart(context.Background(), "user-1", "off")
			if !errors.Is(err, tc.want) {
				t.Fatalf("ApplyToCart error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreateNormalizesCouponCode(t *testing.T) {
	repo := &fakeCouponRepo{}
	service := NewService(repo)
	item, err := service.Create(context.Background(), MutationRequest{
		Code:  " summer10 ",
		Type:  TypePercent,
		Value: 10,
	})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if item.Code != "SUMMER10" {
		t.Fatalf("Code = %q, want SUMMER10", item.Code)
	}
}

func TestCreateRejectsInvalidPercent(t *testing.T) {
	service := NewService(&fakeCouponRepo{})
	if _, err := service.Create(context.Background(), MutationRequest{Code: "BAD", Type: TypePercent, Value: 101}); !errors.Is(err, errs.ErrValidationFailed) {
		t.Fatalf("Create error = %v, want validation", err)
	}
}

func activeCoupon(kind string, value float64) *Coupon {
	return &Coupon{
		ID:       "coupon-1",
		Code:     "SAVE20",
		Type:     kind,
		Value:    value,
		IsActive: true,
	}
}
