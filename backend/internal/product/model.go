package product

import (
	"time"
)

type Product struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	Stock         int       `json:"stock"`
	CategoryID    string    `json:"category_id"`
	Style         string    `json:"style"`
	Color         string    `json:"color"`
	ImageURL      string    `json:"image_url"`
	AverageRating float64   `json:"average_rating"`
	ReviewCount   int64     `json:"review_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
