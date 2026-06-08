package user

type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=user admin"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active disabled"`
}
