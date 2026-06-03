package review

import "time"

type Review struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	OrderID   string    `json:"order_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RatingSummary struct {
	AverageRating   float64       `json:"average_rating"`
	ReviewCount     int64         `json:"review_count"`
	RatingBreakdown map[int]int64 `json:"rating_breakdown"`
}
