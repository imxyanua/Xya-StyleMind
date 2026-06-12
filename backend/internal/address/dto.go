package address

type AddressRequest struct {
	RecipientName string `json:"recipient_name" validate:"required,min=2,max=120"`
	Phone         string `json:"phone" validate:"required,min=8,max=32"`
	AddressLine   string `json:"address_line" validate:"required,min=5,max=255"`
	City          string `json:"city" validate:"required,min=2,max=120"`
	District      string `json:"district" validate:"required,min=2,max=120"`
	Note          string `json:"note" validate:"omitempty,max=1000"`
	IsDefault     bool   `json:"is_default"`
}
