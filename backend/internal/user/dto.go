package user

type UpdateRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=user admin"`
}
