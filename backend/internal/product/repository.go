package product

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

func (r *Repository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Product, int64, error) {
	var total int64
	whereSQL, args := buildProductWhere(filter)
	countQuery := `
		WITH ratings AS (
			SELECT product_id, AVG(rating)::float AS average_rating, COUNT(*)::bigint AS review_count
			FROM product_reviews
			GROUP BY product_id
		)
		SELECT COUNT(*)
		FROM products p
		LEFT JOIN ratings r ON r.product_id = p.id
	` + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	queryArgs := append(append([]any{}, args...), limit, offset)
	query := `
		WITH ratings AS (
			SELECT product_id, AVG(rating)::float AS average_rating, COUNT(*)::bigint AS review_count
			FROM product_reviews
			GROUP BY product_id
		)
		SELECT p.id, p.name, p.description, p.price, p.stock, p.category_id, p.style, p.color, p.image_url,
		       COALESCE(r.average_rating, 0), COALESCE(r.review_count, 0),
		       p.created_at, p.updated_at
		FROM products p
		LEFT JOIN ratings r ON r.product_id = p.id
	` + whereSQL + `
		ORDER BY ` + productSortClause(filter.Sort) + `
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Product, 0)
	for rows.Next() {
		var p Product
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
			&p.Style, &p.Color, &p.ImageURL, &p.AverageRating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Product, error) {
	var p Product
	err := r.db.QueryRow(ctx, `
		WITH ratings AS (
			SELECT product_id, AVG(rating)::float AS average_rating, COUNT(*)::bigint AS review_count
			FROM product_reviews
			WHERE product_id = $1
			GROUP BY product_id
		)
		SELECT p.id, p.name, p.description, p.price, p.stock, p.category_id, p.style, p.color, p.image_url,
		       COALESCE(r.average_rating, 0), COALESCE(r.review_count, 0),
		       p.created_at, p.updated_at
		FROM products p
		LEFT JOIN ratings r ON r.product_id = p.id
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.CategoryID,
		&p.Style, &p.Color, &p.ImageURL, &p.AverageRating, &p.ReviewCount, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrProductNotFound
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
			return nil, errs.ErrProductNotFound
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
		return errs.ErrProductNotFound
	}
	return nil
}

func buildProductWhere(filter ListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(filter.Query)+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(LOWER(p.name) LIKE "+placeholder+" OR LOWER(p.description) LIKE "+placeholder+")")
	}
	if filter.CategoryID != "" {
		args = append(args, filter.CategoryID)
		clauses = append(clauses, fmt.Sprintf("p.category_id = $%d::uuid", len(args)))
	}
	if filter.MinPrice != nil {
		args = append(args, *filter.MinPrice)
		clauses = append(clauses, fmt.Sprintf("p.price >= $%d", len(args)))
	}
	if filter.MaxPrice != nil {
		args = append(args, *filter.MaxPrice)
		clauses = append(clauses, fmt.Sprintf("p.price <= $%d", len(args)))
	}
	if filter.Style != "" {
		args = append(args, strings.ToLower(filter.Style))
		clauses = append(clauses, fmt.Sprintf("p.style = $%d", len(args)))
	}
	if filter.Color != "" {
		args = append(args, strings.ToLower(filter.Color))
		clauses = append(clauses, fmt.Sprintf("p.color = $%d", len(args)))
	}
	if filter.MinRating != nil {
		args = append(args, *filter.MinRating)
		clauses = append(clauses, fmt.Sprintf("COALESCE(r.average_rating, 0) >= $%d", len(args)))
	}
	if filter.InStock != nil {
		if *filter.InStock {
			clauses = append(clauses, "p.stock > 0")
		} else {
			clauses = append(clauses, "p.stock = 0")
		}
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func productSortClause(sort string) string {
	switch sort {
	case SortPriceAsc:
		return "p.price ASC, p.created_at DESC"
	case SortPriceDesc:
		return "p.price DESC, p.created_at DESC"
	case SortRatingDesc:
		return "COALESCE(r.average_rating, 0) DESC, COALESCE(r.review_count, 0) DESC, p.created_at DESC"
	case SortPopular:
		return "COALESCE(r.review_count, 0) DESC, COALESCE(r.average_rating, 0) DESC, p.created_at DESC"
	case SortNewest, "":
		return "p.created_at DESC"
	default:
		return "p.created_at DESC"
	}
}
