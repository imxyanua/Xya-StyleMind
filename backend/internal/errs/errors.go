package errs

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidPayload     = errors.New("invalid payload")
	ErrValidationFailed   = errors.New("validation failed")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrProductNotFound    = errors.New("product not found")
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCartEmpty          = errors.New("cart is empty")
	ErrCartItemNotFound   = errors.New("cart item not found")
	ErrInvalidQuantity    = errors.New("quantity must be greater than 0")
	ErrInsufficientStock  = errors.New("insufficient stock")
	ErrOrderNotFound      = errors.New("order not found")
	ErrForbidden          = errors.New("forbidden")
)
