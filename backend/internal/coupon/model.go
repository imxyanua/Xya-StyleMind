package coupon

import (
	"math"
	"strings"
	"time"
)

const (
	TypePercent = "percent"
	TypeFixed   = "fixed"

	SortNewest = "newest"
	SortOldest = "oldest"
)

type Coupon struct {
	ID                string     `json:"id"`
	Code              string     `json:"code"`
	Type              string     `json:"type"`
	Value             float64    `json:"value"`
	MinOrderAmount    float64    `json:"min_order_amount"`
	MaxDiscountAmount *float64   `json:"max_discount_amount,omitempty"`
	UsageLimit        *int       `json:"usage_limit,omitempty"`
	UsedCount         int        `json:"used_count"`
	StartsAt          *time.Time `json:"starts_at,omitempty"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ApplyResult struct {
	CouponID       string  `json:"coupon_id"`
	CouponCode     string  `json:"coupon_code"`
	Type           string  `json:"type"`
	Value          float64 `json:"value"`
	SubtotalAmount float64 `json:"subtotal_amount"`
	DiscountAmount float64 `json:"discount_amount"`
	TotalAmount    float64 `json:"total_amount"`
}

type ListFilter struct {
	Query    string
	Type     string
	IsActive *bool
	Sort     string
}

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func CalculateDiscount(subtotal float64, c Coupon) float64 {
	if subtotal <= 0 {
		return 0
	}

	var discount float64
	switch c.Type {
	case TypePercent:
		discount = subtotal * (c.Value / 100)
	case TypeFixed:
		discount = c.Value
	default:
		return 0
	}

	if c.MaxDiscountAmount != nil && discount > *c.MaxDiscountAmount {
		discount = *c.MaxDiscountAmount
	}
	if discount > subtotal {
		discount = subtotal
	}
	return roundMoney(discount)
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
