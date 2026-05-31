package cart

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCartItemNotFound = errors.New("cart item not found")
	ErrProductNotFound  = errors.New("product not found")
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
		SELECT id, user_id, created_at, updated_at
		FROM carts
		WHERE user_id = $1
	`, userID).Scan(&cart.ID, &cart.UserID, &cart.CreatedAt, &cart.UpdatedAt)
	if err == nil {
		return cart, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	cart.ID = uuid.NewString()
	cart.UserID = userID
	err = r.db.QueryRow(ctx, `
		INSERT INTO carts (id, user_id)
		VALUES ($1, $2)
		RETURNING created_at, updated_at
	`, cart.ID, cart.UserID).Scan(&cart.CreatedAt, &cart.UpdatedAt)
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
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *Repository) GetCartItemByProduct(ctx context.Context, cartID, productID string) (*CartItemRecord, error) {
	item := &CartItemRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT id, cart_id, product_id, quantity, created_at, updated_at
		FROM cart_items
		WHERE cart_id = $1 AND product_id = $2
	`, cartID, productID).Scan(&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCartItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *Repository) CreateCartItem(ctx context.Context, cartID, productID string, quantity int) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO cart_items (id, cart_id, product_id, quantity)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), cartID, productID, quantity)
	return err
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
		return ErrCartItemNotFound
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
			return nil, ErrCartItemNotFound
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
		return ErrCartItemNotFound
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
