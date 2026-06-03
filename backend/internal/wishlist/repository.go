package wishlist

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ProductExists(ctx context.Context, productID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists)
	return exists, err
}

func (r *Repository) AddProduct(ctx context.Context, userID, productID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO wishlist_items (id, user_id, product_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id) DO NOTHING
	`, uuid.NewString(), userID, productID)
	return err
}

func (r *Repository) RemoveProduct(ctx context.Context, userID, productID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`, userID, productID)
	return err
}

func (r *Repository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]WishlistItem, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM wishlist_items WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT wi.id, wi.user_id, wi.product_id, wi.created_at,
		       p.id, p.name, p.description, p.price, p.stock, p.category_id, p.style, p.color, p.image_url
		FROM wishlist_items wi
		JOIN products p ON p.id = wi.product_id
		WHERE wi.user_id = $1
		ORDER BY wi.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]WishlistItem, 0)
	for rows.Next() {
		var item WishlistItem
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.ProductID, &item.CreatedAt,
			&item.Product.ID, &item.Product.Name, &item.Product.Description, &item.Product.Price,
			&item.Product.Stock, &item.Product.CategoryID, &item.Product.Style, &item.Product.Color, &item.Product.ImageURL,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
