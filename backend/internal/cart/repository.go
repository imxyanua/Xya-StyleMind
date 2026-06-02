package cart

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

func (r *Repository) GetOrCreateCart(ctx context.Context, userID string) (*Cart, error) {
	cart := &Cart{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO carts (id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id)
		DO UPDATE SET updated_at = carts.updated_at
		RETURNING id, user_id, created_at, updated_at
	`, uuid.NewString(), userID).Scan(&cart.ID, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return cart, nil
}

func (r *Repository) GetProductSnapshot(ctx context.Context, productID string) (*ProductSnapshot, error) {
	p := &ProductSnapshot{}
	err := r.db.QueryRow(ctx, `
		SELECT id, name, price, stock, image_url, style, color
		FROM products
		WHERE id = $1
	`, productID).Scan(&p.ID, &p.Name, &p.Price, &p.Stock, &p.ImageURL, &p.Style, &p.Color)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrProductNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) AddOrIncrementCartItem(ctx context.Context, cartID, productID string, quantity int) error {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO cart_items (id, cart_id, product_id, quantity)
		SELECT $1, $2, $3, $4
		WHERE EXISTS (
			SELECT 1
			FROM products
			WHERE id = $3 AND stock >= $4
		)
		ON CONFLICT (cart_id, product_id)
		DO UPDATE
		SET quantity = cart_items.quantity + EXCLUDED.quantity,
		    updated_at = NOW()
		WHERE cart_items.quantity + EXCLUDED.quantity <= (
			SELECT stock
			FROM products
			WHERE id = cart_items.product_id
		)
		RETURNING id
	`, uuid.NewString(), cartID, productID, quantity).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errs.ErrInsufficientStock
		}
		return err
	}
	return nil
}

func (r *Repository) UpdateCartItemQuantity(ctx context.Context, itemID string, quantity int) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE cart_items
		SET quantity = $2, updated_at = NOW()
		WHERE id = $1
	`, itemID, quantity)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrCartItemNotFound
	}
	return nil
}

func (r *Repository) GetCartItemByID(ctx context.Context, cartID, itemID string) (*CartItemRecord, error) {
	item := &CartItemRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.created_at, ci.updated_at,
		       p.id, p.name, p.price, p.stock, p.image_url, p.style, p.color
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1 AND ci.id = $2
	`, cartID, itemID).Scan(
		&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
		&item.Product.ID, &item.Product.Name, &item.Product.Price, &item.Product.Stock, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrCartItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *Repository) DeleteCartItem(ctx context.Context, cartID, itemID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1 AND id = $2`, cartID, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrCartItemNotFound
	}
	return nil
}

func (r *Repository) ListCartItems(ctx context.Context, cartID string) ([]CartItemRecord, error) {
	rows, err := r.db.Query(ctx, `
		SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.created_at, ci.updated_at,
		       p.id, p.name, p.price, p.stock, p.image_url, p.style, p.color
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CartItemRecord, 0)
	for rows.Next() {
		var item CartItemRecord
		if err := rows.Scan(
			&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt,
			&item.Product.ID, &item.Product.Name, &item.Product.Price, &item.Product.Stock, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
