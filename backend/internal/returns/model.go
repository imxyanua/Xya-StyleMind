package returns

import "time"

const (
	StatusRequested = "requested"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

const (
	SortNewest = "newest"
	SortOldest = "oldest"
)

type Request struct {
	ID        string     `json:"id"`
	OrderID   string     `json:"order_id"`
	UserID    string     `json:"user_id"`
	Reason    string     `json:"reason"`
	Status    string     `json:"status"`
	AdminNote string     `json:"admin_note,omitempty"`
	Order     *OrderInfo `json:"order,omitempty"`
	User      *UserInfo  `json:"user,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type OrderInfo struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"`
	PaymentStatus  string    `json:"payment_status"`
	PaymentMethod  string    `json:"payment_method,omitempty"`
	TotalAmount    float64   `json:"total_amount"`
	RecipientName  string    `json:"recipient_name,omitempty"`
	ShippingMethod string    `json:"shipping_method,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

type AdminFilter struct {
	Status  string
	UserID  string
	OrderID string
	From    *time.Time
	To      *time.Time
	Sort    string
}
