package product

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProductNotFound = errors.New("product not found")

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	query := `
		SELECT id, name, description, price, stock, category_id, style, color, image_url, created_at, updated_at
		FROM products
		WHERE ($1 = '' OR style = $1)
		  AND ($2 = '' OR color = $2)
		  AND ($3 = '' OR category_id = $3::uuid)
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, strings.ToLower(filter.Style), strings.ToLower(filter.Color), filter.CategoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
			&p.Style, &p.Color, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx, `
		SELECT id, name, description, price, stock, category_id, style, color, image_url, created_at, updated_at
		FROM products
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
		&p.Style, &p.Color, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	p := &Product{
		ID:          uuid.NewString(),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
		Style:       strings.ToLower(strings.TrimSpace(req.Style)),
		Color:       strings.ToLower(strings.TrimSpace(req.Color)),
		ImageURL:    strings.TrimSpace(req.ImageURL),
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at, updated_at
	`, p.ID, p.Name, p.Description, p.Price, p.Stock, p.CategoryID, p.Style, p.Color, p.ImageURL).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx, `
		UPDATE products
		SET name = $2, description = $3, price = $4, stock = $5, category_id = $6, style = $7, color = $8, image_url = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, description, price, stock, category_id, style, color, image_url, created_at, updated_at
	`, id,
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Description),
		req.Price,
		req.Stock,
		req.CategoryID,
		strings.ToLower(strings.TrimSpace(req.Style)),
		strings.ToLower(strings.TrimSpace(req.Color)),
		strings.TrimSpace(req.ImageURL),
	).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
		&p.Style, &p.Color, &p.ImageURL, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}
