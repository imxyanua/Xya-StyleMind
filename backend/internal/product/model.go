package product

import (
	"time"
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CategoryID  string    `json:"category_id"`
	Style       string    `json:"style"`
	Color       string    `json:"color"`
	ImageURL    string    `json:"image_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	Description string  `json:"description" validate:"required,min=5,max=5000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	CategoryID  string  `json:"category_id" validate:"required"`
	Style       string  `json:"style" validate:"required,min=2,max=100"`
	Color       string  `json:"color" validate:"required,min=2,max=100"`
	ImageURL    string  `json:"image_url" validate:"required,url"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	Description string  `json:"description" validate:"required,min=5,max=5000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	CategoryID  string  `json:"category_id" validate:"required"`
	Style       string  `json:"style" validate:"required,min=2,max=100"`
	Color       string  `json:"color" validate:"required,min=2,max=100"`
	ImageURL    string  `json:"image_url" validate:"required,url"`
}

type ListFilter struct {
	Style      string
	Color      string
	CategoryID string
}
