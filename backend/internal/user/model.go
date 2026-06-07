package user

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"

	StatusActive   = "active"
	StatusDisabled = "disabled"

	SortNewest = "newest"
	SortOldest = "oldest"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ListFilter struct {
	Query  string
	Role   string
	Status string
	Sort   string
}
