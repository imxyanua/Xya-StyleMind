package coupon

import "time"

type ApplyCouponRequest struct {
	Code string `json:"code" validate:"required,min=2,max=64"`
}

type MutationRequest struct {
	Code              string     `json:"code" validate:"required,min=2,max=64"`
	Type              string     `json:"type" validate:"required,oneof=percent fixed"`
	Value             float64    `json:"value" validate:"required,gt=0"`
	MinOrderAmount    float64    `json:"min_order_amount" validate:"gte=0"`
	MaxDiscountAmount *float64   `json:"max_discount_amount" validate:"omitempty,gte=0"`
	UsageLimit        *int       `json:"usage_limit" validate:"omitempty,gt=0"`
	StartsAt          *time.Time `json:"starts_at"`
	ExpiresAt         *time.Time `json:"expires_at"`
	IsActive          *bool      `json:"is_active"`
}
