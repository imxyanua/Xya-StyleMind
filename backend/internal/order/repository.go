package order

import (
	"context"
	"errors"
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
	err := r.db.QueryRow(ctx, `SELECT id FROM carts WHERE user_id = $1`, userID).Scan(&cartID)
	if err == nil {
		return cartID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	cartID = uuid.NewString()
	_, err = r.db.Exec(ctx, `INSERT INTO carts (id, user_id) VALUES ($1, $2)`, cartID, userID)
	if err != nil {
		return "", err
	}
	return cartID, nil
}

func (r *Repository) GetCheckoutItems(ctx context.Context, cartID string) ([]CheckoutItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ci.id, ci.product_id, p.name, p.image_url, p.style, p.color, p.price, p.stock, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CheckoutItem, 0)
	for rows.Next() {
		var item CheckoutItem
		if err := rows.Scan(
			&item.CartItemID, &item.ProductID, &item.Name, &item.ImageURL, &item.Style, &item.Color,
			&item.Price, &item.Stock, &item.Quantity,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errs.ErrCartEmpty
	}
	return items, nil
}

func (r *Repository) CreateOrderFromCart(ctx context.Context, userID, cartID string, items []CheckoutItem) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

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
	for rows.Next() {
		var o OrderResponse
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.TotalAmount, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items, err := r.GetOrderItems(ctx, o.ID)
		if err != nil {
			return nil, 0, err
		}
		o.Items = items
		out = append(out, o)
	}
	return out, total, rows.Err()
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

func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID, status string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`, orderID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrOrderNotFound
	}
	return nil
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
