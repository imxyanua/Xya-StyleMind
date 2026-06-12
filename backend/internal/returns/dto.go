package returns

type CreateRequest struct {
	Reason string `json:"reason" validate:"required,min=10,max=1000"`
}

type UpdateStatusRequest struct {
	Status    string `json:"status" validate:"required,oneof=approved rejected cancelled"`
	AdminNote string `json:"admin_note" validate:"omitempty,max=1000"`
}
