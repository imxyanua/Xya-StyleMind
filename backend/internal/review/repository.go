package review

import (
	"context"
	"errors"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *Repository) HasPurchasedProduct(ctx context.Context, userID, productID, orderID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			WHERE o.id = $1
			  AND o.user_id = $2
			  AND oi.product_id = $3
			  AND o.status IN ('paid', 'shipping', 'completed')
		)
	`, orderID, userID, productID).Scan(&exists)
	return exists, err
}

func (r *Repository) Create(ctx context.Context, userID, productID string, req CreateReviewRequest) (*Review, error) {
	review := &Review{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO product_reviews (id, user_id, product_id, order_id, rating, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, product_id, order_id, rating, comment, created_at, updated_at
	`, uuid.NewString(), userID, productID, req.OrderID, req.Rating, normalizeComment(req.Comment)).Scan(
		&review.ID, &review.UserID, &review.ProductID, &review.OrderID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errs.ErrReviewAlreadyExists
		}
		return nil, err
	}
	return review, nil
}

func (r *Repository) ListByProduct(ctx context.Context, productID string, limit, offset int) ([]Review, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM product_reviews WHERE product_id = $1`, productID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, product_id, order_id, rating, comment, created_at, updated_at
		FROM product_reviews
		WHERE product_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, productID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	reviews := make([]Review, 0)
	for rows.Next() {
		var review Review
		if err := rows.Scan(&review.ID, &review.UserID, &review.ProductID, &review.OrderID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt); err != nil {
			return nil, 0, err
		}
		reviews = append(reviews, review)
	}
	return reviews, total, rows.Err()
}

func (r *Repository) SummaryByProduct(ctx context.Context, productID string) (*RatingSummary, error) {
	summary := &RatingSummary{RatingBreakdown: map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}}
	var one, two, three, four, five int64
	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(AVG(rating)::float, 0),
			COUNT(*),
			COUNT(*) FILTER (WHERE rating = 1),
			COUNT(*) FILTER (WHERE rating = 2),
			COUNT(*) FILTER (WHERE rating = 3),
			COUNT(*) FILTER (WHERE rating = 4),
			COUNT(*) FILTER (WHERE rating = 5)
		FROM product_reviews
		WHERE product_id = $1
	`, productID).Scan(
		&summary.AverageRating,
		&summary.ReviewCount,
		&one,
		&two,
		&three,
		&four,
		&five,
	)
	if err != nil {
		return nil, err
	}
	summary.RatingBreakdown[1] = one
	summary.RatingBreakdown[2] = two
	summary.RatingBreakdown[3] = three
	summary.RatingBreakdown[4] = four
	summary.RatingBreakdown[5] = five
	return summary, nil
}

func (r *Repository) GetByID(ctx context.Context, reviewID string) (*Review, error) {
	review := &Review{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, product_id, order_id, rating, comment, created_at, updated_at
		FROM product_reviews
		WHERE id = $1
	`, reviewID).Scan(&review.ID, &review.UserID, &review.ProductID, &review.OrderID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}

func (r *Repository) Update(ctx context.Context, reviewID string, req UpdateReviewRequest) (*Review, error) {
	review := &Review{}
	err := r.db.QueryRow(ctx, `
		UPDATE product_reviews
		SET rating = $2, comment = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, user_id, product_id, order_id, rating, comment, created_at, updated_at
	`, reviewID, req.Rating, normalizeComment(req.Comment)).Scan(
		&review.ID, &review.UserID, &review.ProductID, &review.OrderID, &review.Rating, &review.Comment, &review.CreatedAt, &review.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrReviewNotFound
		}
		return nil, err
	}
	return review, nil
}

func (r *Repository) Delete(ctx context.Context, reviewID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM product_reviews WHERE id = $1`, reviewID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrReviewNotFound
	}
	return nil
}

func normalizeComment(comment *string) *string {
	if comment == nil {
		return nil
	}
	value := strings.TrimSpace(*comment)
	if value == "" {
		return nil
	}
	return &value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
