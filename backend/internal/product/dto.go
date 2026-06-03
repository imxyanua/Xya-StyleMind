package product

type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	Description string  `json:"description" validate:"required,min=5,max=5000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	CategoryID  string  `json:"category_id" validate:"required,uuid"`
	Style       string  `json:"style" validate:"required,min=2,max=100"`
	Color       string  `json:"color" validate:"required,min=2,max=100"`
	ImageURL    string  `json:"image_url" validate:"required,url"`
}

type UpdateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=255"`
	Description string  `json:"description" validate:"required,min=5,max=5000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
	CategoryID  string  `json:"category_id" validate:"required,uuid"`
	Style       string  `json:"style" validate:"required,min=2,max=100"`
	Color       string  `json:"color" validate:"required,min=2,max=100"`
	ImageURL    string  `json:"image_url" validate:"required,url"`
}

type ListFilter struct {
	Query      string
	CategoryID string
	MinPrice   *float64
	MaxPrice   *float64
	Style      string
	Color      string
	MinRating  *float64
	InStock    *bool
	Sort       string
	Page       int
	Limit      int
	Offset     int
}

const (
	SortNewest     = "newest"
	SortPriceAsc   = "price_asc"
	SortPriceDesc  = "price_desc"
	SortRatingDesc = "rating_desc"
	SortPopular    = "popular"
)
