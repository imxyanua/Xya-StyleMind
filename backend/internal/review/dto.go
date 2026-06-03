package review

type CreateReviewRequest struct {
	OrderID string  `json:"order_id" validate:"required,uuid"`
	Rating  int     `json:"rating" validate:"required,min=1,max=5"`
	Comment *string `json:"comment" validate:"omitempty,max=1000"`
}

type UpdateReviewRequest struct {
	Rating  int     `json:"rating" validate:"required,min=1,max=5"`
	Comment *string `json:"comment" validate:"omitempty,max=1000"`
}
