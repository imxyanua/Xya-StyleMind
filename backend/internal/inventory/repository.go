package inventory

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListActiveByUser(ctx context.Context, userID string, limit, offset int) ([]Reservation, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM inventory_reservations
		WHERE user_id = $1 AND expires_at > NOW()
	`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT ir.id, ir.user_id, ir.product_id, ir.quantity, ir.expires_at, ir.created_at,
		       p.id, p.name, p.price, p.stock, p.image_url, p.style, p.color
		FROM inventory_reservations ir
		JOIN products p ON p.id = ir.product_id
		WHERE ir.user_id = $1 AND ir.expires_at > NOW()
		ORDER BY ir.expires_at ASC, ir.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Reservation, 0)
	for rows.Next() {
		var item Reservation
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.ProductID, &item.Quantity, &item.ExpiresAt, &item.CreatedAt,
			&item.Product.ID, &item.Product.Name, &item.Product.Price, &item.Product.Stock,
			&item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
