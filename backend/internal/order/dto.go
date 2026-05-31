package order

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending paid shipping completed cancelled"`
}
