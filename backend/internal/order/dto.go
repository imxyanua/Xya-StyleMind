package order

type CheckoutRequest struct {
	RecipientName  string `json:"recipient_name" validate:"required,min=2,max=120"`
	Phone          string `json:"phone" validate:"required,min=8,max=32"`
	AddressLine    string `json:"address_line" validate:"required,min=5,max=255"`
	City           string `json:"city" validate:"required,min=2,max=120"`
	District       string `json:"district" validate:"required,min=2,max=120"`
	Note           string `json:"note" validate:"omitempty,max=1000"`
	ShippingMethod string `json:"shipping_method" validate:"required,oneof=standard express"`
	PaymentMethod  string `json:"payment_method" validate:"required,oneof=cod demo_payment"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending paid shipping completed cancelled"`
}
