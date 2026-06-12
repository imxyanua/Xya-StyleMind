package address

import "time"

type Address struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	RecipientName string    `json:"recipient_name"`
	Phone         string    `json:"phone"`
	AddressLine   string    `json:"address_line"`
	City          string    `json:"city"`
	District      string    `json:"district"`
	Note          string    `json:"note,omitempty"`
	IsDefault     bool      `json:"is_default"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
