package order

import "time"

const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusShipping  = "shipping"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

const (
	PaymentStatusUnpaid   = "unpaid"
	PaymentStatusPending  = "pending"
	PaymentStatusPaid     = "paid"
	PaymentStatusFailed   = "failed"
	PaymentStatusRefunded = "refunded"
)

type Order struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Status         string    `json:"status"`
	PaymentStatus  string    `json:"payment_status"`
	SubtotalAmount float64   `json:"subtotal_amount"`
	DiscountAmount float64   `json:"discount_amount"`
	CouponID       string    `json:"coupon_id,omitempty"`
	CouponCode     string    `json:"coupon_code,omitempty"`
	TotalAmount    float64   `json:"total_amount"`
	RecipientName  string    `json:"recipient_name,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	AddressLine    string    `json:"address_line,omitempty"`
	City           string    `json:"city,omitempty"`
	District       string    `json:"district,omitempty"`
	Note           string    `json:"note,omitempty"`
	ShippingMethod string    `json:"shipping_method,omitempty"`
	PaymentMethod  string    `json:"payment_method,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProductInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Style    string `json:"style"`
	Color    string `json:"color"`
}

type OrderItem struct {
	ID        string      `json:"id"`
	ProductID string      `json:"product_id"`
	Quantity  int         `json:"quantity"`
	UnitPrice float64     `json:"unit_price"`
	Subtotal  float64     `json:"subtotal"`
	Product   ProductInfo `json:"product"`
}

type OrderUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type OrderResponse struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	User           *OrderUser  `json:"user,omitempty"`
	Status         string      `json:"status"`
	PaymentStatus  string      `json:"payment_status"`
	SubtotalAmount float64     `json:"subtotal_amount"`
	DiscountAmount float64     `json:"discount_amount"`
	CouponID       string      `json:"coupon_id,omitempty"`
	CouponCode     string      `json:"coupon_code,omitempty"`
	TotalAmount    float64     `json:"total_amount"`
	RecipientName  string      `json:"recipient_name,omitempty"`
	Phone          string      `json:"phone,omitempty"`
	AddressLine    string      `json:"address_line,omitempty"`
	City           string      `json:"city,omitempty"`
	District       string      `json:"district,omitempty"`
	Note           string      `json:"note,omitempty"`
	ShippingMethod string      `json:"shipping_method,omitempty"`
	PaymentMethod  string      `json:"payment_method,omitempty"`
	Items          []OrderItem `json:"items"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type CheckoutDetails struct {
	RecipientName  string
	Phone          string
	AddressLine    string
	City           string
	District       string
	Note           string
	ShippingMethod string
	PaymentMethod  string
	CouponCode     string
}

type CheckoutItem struct {
	CartItemID string
	ProductID  string
	Name       string
	ImageURL   string
	Style      string
	Color      string
	Price      float64
	Stock      int
	Quantity   int
}

const (
	AdminOrderSortNewest = "newest"
	AdminOrderSortOldest = "oldest"
)

type AdminOrderFilter struct {
	Query  string
	Status string
	UserID string
	From   *time.Time
	To     *time.Time
	Sort   string
}
