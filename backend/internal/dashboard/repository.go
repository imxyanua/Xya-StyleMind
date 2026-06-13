package dashboard

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetStats(ctx context.Context, filter StatsFilter) (*Stats, error) {
	stats := &Stats{
		RecentOrders:     make([]RecentOrder, 0),
		LowStockProducts: make([]LowStockProduct, 0),
		RevenueByDay:     make([]RevenueByDay, 0),
		TopProducts:      make([]TopProduct, 0),
	}
	whereSQL, args := buildOrderDateWhere(filter, "o")
	revenueWhere := appendRevenueStatusWhere(whereSQL, "o")

	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders o`+whereSQL, args...).Scan(&stats.TotalOrders); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COALESCE(SUM(o.total_amount), 0) FROM orders o`+revenueWhere, args...).Scan(&stats.TotalRevenue); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&stats.TotalProducts); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers); err != nil {
		return nil, err
	}
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM inventory_reservations
		WHERE expires_at > NOW()
	`).Scan(&stats.ActiveReservations); err != nil {
		return nil, err
	}
	if err := r.loadOrdersByStatus(ctx, stats, whereSQL, args); err != nil {
		return nil, err
	}
	if err := r.loadRecentOrders(ctx, stats, whereSQL, args, 5); err != nil {
		return nil, err
	}
	if err := r.loadLowStockProducts(ctx, stats, 5); err != nil {
		return nil, err
	}
	if err := r.loadRevenueByDay(ctx, stats, revenueWhere, args); err != nil {
		return nil, err
	}
	if err := r.loadTopProducts(ctx, stats, revenueWhere, args, 5); err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *Repository) loadOrdersByStatus(ctx context.Context, stats *Stats, whereSQL string, args []any) error {
	rows, err := r.db.Query(ctx, `SELECT o.status, COUNT(*) FROM orders o`+whereSQL+` GROUP BY o.status`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return err
		}
		switch status {
		case "pending":
			stats.OrdersByStatus.Pending = count
		case "paid":
			stats.OrdersByStatus.Paid = count
		case "shipping":
			stats.OrdersByStatus.Shipping = count
		case "completed":
			stats.OrdersByStatus.Completed = count
		case "cancelled":
			stats.OrdersByStatus.Cancelled = count
		}
	}
	return rows.Err()
}

func (r *Repository) loadRecentOrders(ctx context.Context, stats *Stats, whereSQL string, args []any, limit int) error {
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	queryArgs := append(append([]any{}, args...), limit)
	rows, err := r.db.Query(ctx, `
		SELECT o.id, o.user_id, u.email, u.full_name, o.status, o.total_amount, o.created_at
		FROM orders o
		JOIN users u ON u.id = o.user_id
	`+whereSQL+`
		ORDER BY o.created_at DESC, o.id DESC
		LIMIT `+limitPlaceholder, queryArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item RecentOrder
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserEmail, &item.UserName, &item.Status, &item.TotalAmount, &item.CreatedAt); err != nil {
			return err
		}
		stats.RecentOrders = append(stats.RecentOrders, item)
	}
	return rows.Err()
}

func (r *Repository) loadLowStockProducts(ctx context.Context, stats *Stats, limit int) error {
	rows, err := r.db.Query(ctx, `
		WITH active_reservations AS (
			SELECT product_id, COALESCE(SUM(quantity), 0)::int AS reserved_quantity
			FROM inventory_reservations
			WHERE expires_at > NOW()
			GROUP BY product_id
		)
		SELECT p.id, p.name, p.stock, COALESCE(ar.reserved_quantity, 0),
		       p.stock - COALESCE(ar.reserved_quantity, 0) AS available_stock,
		       p.price, p.image_url
		FROM products p
		LEFT JOIN active_reservations ar ON ar.product_id = p.id
		WHERE p.stock - COALESCE(ar.reserved_quantity, 0) <= 5
		ORDER BY available_stock ASC, p.updated_at DESC, p.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item LowStockProduct
		if err := rows.Scan(&item.ID, &item.Name, &item.Stock, &item.ReservedQuantity, &item.AvailableStock, &item.Price, &item.ImageURL); err != nil {
			return err
		}
		stats.LowStockProducts = append(stats.LowStockProducts, item)
	}
	return rows.Err()
}

func (r *Repository) loadRevenueByDay(ctx context.Context, stats *Stats, revenueWhere string, args []any) error {
	rows, err := r.db.Query(ctx, `
		SELECT TO_CHAR(DATE(o.created_at), 'YYYY-MM-DD') AS day, COALESCE(SUM(o.total_amount), 0)
		FROM orders o
	`+revenueWhere+`
		GROUP BY DATE(o.created_at)
		ORDER BY DATE(o.created_at) DESC
		LIMIT 14
	`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item RevenueByDay
		if err := rows.Scan(&item.Date, &item.Revenue); err != nil {
			return err
		}
		stats.RevenueByDay = append(stats.RevenueByDay, item)
	}
	return rows.Err()
}

func (r *Repository) loadTopProducts(ctx context.Context, stats *Stats, revenueWhere string, args []any, limit int) error {
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	queryArgs := append(append([]any{}, args...), limit)
	rows, err := r.db.Query(ctx, `
		SELECT p.id, p.name, p.image_url, COALESCE(SUM(oi.quantity), 0)::bigint, COALESCE(SUM(oi.subtotal), 0)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN products p ON p.id = oi.product_id
	`+revenueWhere+`
		GROUP BY p.id, p.name, p.image_url
		ORDER BY COALESCE(SUM(oi.quantity), 0) DESC, COALESCE(SUM(oi.subtotal), 0) DESC
		LIMIT `+limitPlaceholder, queryArgs...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item TopProduct
		if err := rows.Scan(&item.ID, &item.Name, &item.ImageURL, &item.QuantitySold, &item.Revenue); err != nil {
			return err
		}
		stats.TopProducts = append(stats.TopProducts, item)
	}
	return rows.Err()
}

func buildOrderDateWhere(filter StatsFilter, alias string) (string, []any) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	prefix := alias + "."
	if alias == "" {
		prefix = ""
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("%screated_at >= $%d", prefix, len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("%screated_at <= $%d", prefix, len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func appendRevenueStatusWhere(whereSQL, alias string) string {
	statusClause := alias + ".status IN ('" + RevenueStatusPaid + "','" + RevenueStatusShipping + "','" + RevenueStatusCompleted + "')"
	if strings.TrimSpace(whereSQL) == "" {
		return " WHERE " + statusClause
	}
	return whereSQL + " AND " + statusClause
}
