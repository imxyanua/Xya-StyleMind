package wishlist

import "time"

type ProductSnapshot struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	CategoryID  string  `json:"category_id"`
	Style       string  `json:"style"`
	Color       string  `json:"color"`
	ImageURL    string  `json:"image_url"`
}

type WishlistItem struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	ProductID string          `json:"product_id"`
	Product   ProductSnapshot `json:"product"`
	CreatedAt time.Time       `json:"created_at"`
}
