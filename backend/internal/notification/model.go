package notification

import "time"

const (
	TypeOrderCreated          = "order.created"
	TypeOrderStatusUpdated    = "order.status_updated"
	TypePaymentStatusUpdated  = "order.payment_status_updated"
	TypeReturnRequestApproved = "return_request.approved"
	TypeReturnRequestRejected = "return_request.rejected"
)

type Notification struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type CreateInput struct {
	UserID   string
	Type     string
	Title    string
	Message  string
	Metadata map[string]any
}

type ListFilter struct {
	UnreadOnly bool
}

type Preferences struct {
	UserID                string    `json:"user_id"`
	OrderUpdatesEnabled   bool      `json:"order_updates_enabled"`
	PaymentUpdatesEnabled bool      `json:"payment_updates_enabled"`
	ReturnUpdatesEnabled  bool      `json:"return_updates_enabled"`
	PromotionEnabled      bool      `json:"promotion_enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type UpdatePreferencesInput struct {
	OrderUpdatesEnabled   *bool `json:"order_updates_enabled"`
	PaymentUpdatesEnabled *bool `json:"payment_updates_enabled"`
	ReturnUpdatesEnabled  *bool `json:"return_updates_enabled"`
	PromotionEnabled      *bool `json:"promotion_enabled"`
}
