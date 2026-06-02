package order

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetOrCreateCart(ctx context.Context, userID string) (string, error) {
	var cartID string
	err := r.db.QueryRow(ctx, `
		INSERT INTO carts (id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id)
		DO UPDATE SET updated_at = carts.updated_at
		RETURNING id
	`, uuid.NewString(), userID).Scan(&cartID)
	if err != nil {
		return "", err
	}
	return cartID, nil
}

func (r *Repository) CreateOrderFromCart(ctx context.Context, userID, cartID string) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT ci.id, ci.product_id, p.name, p.image_url, p.style, p.color, p.price, p.stock, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
		FOR UPDATE OF ci, p
	`, cartID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	items := make([]CheckoutItem, 0)
	for rows.Next() {
		var item CheckoutItem
		if err := rows.Scan(
			&item.CartItemID, &item.ProductID, &item.Name, &item.ImageURL, &item.Style, &item.Color,
			&item.Price, &item.Stock, &item.Quantity,
		); err != nil {
			return "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", errs.ErrCartEmpty
	}

	total := 0.0
	for _, item := range items {
		if item.Quantity > item.Stock {
			return "", errs.ErrInsufficientStock
		}
		total += item.Price * float64(item.Quantity)
	}

	orderID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, status, total_amount)
		VALUES ($1, $2, $3, $4)
	`, orderID, userID, StatusPending, total)
	if err != nil {
		return "", err
	}

	for _, item := range items {
		subtotal := item.Price * float64(item.Quantity)
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.NewString(), orderID, item.ProductID, item.Quantity, item.Price, subtotal)
		if err != nil {
			return "", err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE products
			SET stock = stock - $2, updated_at = NOW()
			WHERE id = $1 AND stock >= $2
		`, item.ProductID, item.Quantity)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return "", errs.ErrInsufficientStock
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return orderID, nil
}

func (r *Repository) ListOrdersByUser(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]OrderResponse, 0)
	orderIDs := make([]string, 0)
	for rows.Next() {
		var o OrderResponse
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		o.Items = make([]OrderItem, 0)
		out = append(out, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(orderIDs) == 0 {
		return out, total, nil
	}

	itemsByOrderID, err := r.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if items, ok := itemsByOrderID[out[i].ID]; ok {
			out[i].Items = items
		}
	}
	return out, total, nil
}

func (r *Repository) ListOrders(ctx context.Context, limit, offset int) ([]OrderResponse, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]OrderResponse, 0)
	orderIDs := make([]string, 0)
	for rows.Next() {
		var o OrderResponse
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		o.Items = make([]OrderItem, 0)
		out = append(out, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(orderIDs) == 0 {
		return out, total, nil
	}

	itemsByOrderID, err := r.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if items, ok := itemsByOrderID[out[i].ID]; ok {
			out[i].Items = items
		}
	}
	return out, total, nil
}

func (r *Repository) GetOrderByIDForUser(ctx context.Context, orderID, userID string) (*OrderResponse, error) {
	o := &OrderResponse{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := r.GetOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (r *Repository) GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	o := &OrderResponse{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, total_amount, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, orderID).Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := r.GetOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID, status string, allowedCurrentStatuses []string) error {
	if len(allowedCurrentStatuses) == 0 {
		return errs.ErrInvalidOrderStatus
	}

	args := []any{orderID, status}
	placeholders := make([]string, len(allowedCurrentStatuses))
	for i, currentStatus := range allowedCurrentStatuses {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, currentStatus)
	}

	query := fmt.Sprintf(`
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND status IN (%s)
	`, strings.Join(placeholders, ","))

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetOrderStatus(ctx, orderID); err != nil {
			return err
		}
		return errs.ErrInvalidOrderStatusTransition
	}
	return nil
}

func (r *Repository) GetOrderStatus(ctx context.Context, orderID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrOrderNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT oi.id, oi.product_id, oi.quantity, oi.unit_price, oi.subtotal,
		       p.id, p.name, p.image_url, p.style, p.color
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
		ORDER BY oi.created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal,
			&item.Product.ID, &item.Product.Name, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetOrderItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]OrderItem, error) {
	itemsByOrderID := make(map[string][]OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return itemsByOrderID, nil
	}

	placeholders := make([]string, len(orderIDs))
	args := make([]any, len(orderIDs))
	for i, orderID := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orderID
		itemsByOrderID[orderID] = make([]OrderItem, 0)
	}

	query := fmt.Sprintf(`
		SELECT oi.order_id, oi.id, oi.product_id, oi.quantity, oi.unit_price, oi.subtotal,
		       p.id, p.name, p.image_url, p.style, p.color
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id IN (%s)
		ORDER BY oi.created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item OrderItem
		if err := rows.Scan(
			&orderID, &item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal,
			&item.Product.ID, &item.Product.Name, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, err
		}
		itemsByOrderID[orderID] = append(itemsByOrderID[orderID], item)
	}
	return itemsByOrderID, rows.Err()
}
