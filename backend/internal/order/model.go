package order

import "time"

const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusShipping  = "shipping"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

type Order struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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

type OrderResponse struct {
	ID          string      `json:"id"`
	UserID      string      `json:"user_id"`
	Status      string      `json:"status"`
	TotalAmount float64     `json:"total_amount"`
	Items       []OrderItem `json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
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
