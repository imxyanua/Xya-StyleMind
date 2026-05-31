package cart

import "time"

type Cart struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ProductSnapshot struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	ImageURL string  `json:"image_url"`
	Style    string  `json:"style"`
	Color    string  `json:"color"`
}

type CartItemRecord struct {
	ID        string
	CartID    string
	ProductID string
	Quantity  int
	CreatedAt time.Time
	UpdatedAt time.Time
	Product   ProductSnapshot
}

type CartItemResponse struct {
	ID       string          `json:"id"`
	Product  ProductSnapshot `json:"product"`
	Quantity int             `json:"quantity"`
	Subtotal float64         `json:"subtotal"`
}

type CartResponse struct {
	CartID string             `json:"cart_id"`
	UserID string             `json:"user_id"`
	Items  []CartItemResponse `json:"items"`
	Total  float64            `json:"total"`
}

type AddCartItemRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" validate:"required,gt=0"`
}
