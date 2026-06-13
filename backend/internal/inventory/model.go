package inventory

import "time"

const DefaultReservationTTL = 15 * time.Minute

type Reservation struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	ProductID string      `json:"product_id"`
	Quantity  int         `json:"quantity"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
	Product   ProductInfo `json:"product"`
}

type ProductInfo struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Stock    int     `json:"stock"`
	ImageURL string  `json:"image_url"`
	Style    string  `json:"style"`
	Color    string  `json:"color"`
}
