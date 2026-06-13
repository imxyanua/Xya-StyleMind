package dashboard

import "time"

const (
	RevenueStatusPaid      = "paid"
	RevenueStatusShipping  = "shipping"
	RevenueStatusCompleted = "completed"
)

type StatsFilter struct {
	From *time.Time
	To   *time.Time
}

type OrdersByStatus struct {
	Pending   int64 `json:"pending"`
	Paid      int64 `json:"paid"`
	Shipping  int64 `json:"shipping"`
	Completed int64 `json:"completed"`
	Cancelled int64 `json:"cancelled"`
}

type RecentOrder struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	UserEmail   string    `json:"user_email"`
	UserName    string    `json:"user_name"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	CreatedAt   time.Time `json:"created_at"`
}

type LowStockProduct struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Stock            int     `json:"stock"`
	ReservedQuantity int     `json:"reserved_quantity"`
	AvailableStock   int     `json:"available_stock"`
	Price            float64 `json:"price"`
	ImageURL         string  `json:"image_url"`
}

type RevenueByDay struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
}

type TopProduct struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ImageURL     string  `json:"image_url"`
	QuantitySold int64   `json:"quantity_sold"`
	Revenue      float64 `json:"revenue"`
}

type Stats struct {
	TotalRevenue       float64           `json:"total_revenue"`
	TotalOrders        int64             `json:"total_orders"`
	TotalProducts      int64             `json:"total_products"`
	TotalUsers         int64             `json:"total_users"`
	ActiveReservations int64             `json:"active_reservations"`
	OrdersByStatus     OrdersByStatus    `json:"orders_by_status"`
	RecentOrders       []RecentOrder     `json:"recent_orders"`
	LowStockProducts   []LowStockProduct `json:"low_stock_products"`
	RevenueByDay       []RevenueByDay    `json:"revenue_by_day"`
	TopProducts        []TopProduct      `json:"top_products"`
}
