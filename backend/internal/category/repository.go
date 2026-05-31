package category

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]Category, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM categories
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *Repository) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	category := &Category{
		ID:   uuid.NewString(),
		Name: strings.TrimSpace(req.Name),
		Slug: strings.ToLower(strings.TrimSpace(req.Slug)),
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING created_at, updated_at
	`, category.ID, category.Name, category.Slug).Scan(&category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return category, nil
}
