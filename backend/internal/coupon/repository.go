package coupon

import (
	"context"
	"errors"
	"fmt"
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

func (r *Repository) GetOrCreateCartID(ctx context.Context, userID string) (string, error) {
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

func (r *Repository) GetCartSubtotal(ctx context.Context, cartID string) (float64, error) {
	var subtotal float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(p.price * ci.quantity), 0)
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
	`, cartID).Scan(&subtotal)
	if err != nil {
		return 0, err
	}
	if subtotal <= 0 {
		return 0, errs.ErrCartEmpty
	}
	return subtotal, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (*Coupon, error) {
	c := &Coupon{}
	err := r.db.QueryRow(ctx, `
		SELECT id, code, type, value, min_order_amount, max_discount_amount, usage_limit,
		       used_count, starts_at, expires_at, is_active, created_at, updated_at
		FROM coupons
		WHERE LOWER(code) = LOWER($1)
	`, code).Scan(
		&c.ID, &c.Code, &c.Type, &c.Value, &c.MinOrderAmount, &c.MaxDiscountAmount, &c.UsageLimit,
		&c.UsedCount, &c.StartsAt, &c.ExpiresAt, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrCouponNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Coupon, int64, error) {
	whereSQL, args := buildWhere(filter)
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM coupons`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	rows, err := r.db.Query(ctx, `
		SELECT id, code, type, value, min_order_amount, max_discount_amount, usage_limit,
		       used_count, starts_at, expires_at, is_active, created_at, updated_at
		FROM coupons
	`+whereSQL+`
		ORDER BY `+sortClause(filter.Sort)+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Coupon, 0)
	for rows.Next() {
		var c Coupon
		if err := scanCoupon(rows, &c); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Coupon, error) {
	c := &Coupon{}
	err := r.db.QueryRow(ctx, `
		SELECT id, code, type, value, min_order_amount, max_discount_amount, usage_limit,
		       used_count, starts_at, expires_at, is_active, created_at, updated_at
		FROM coupons
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.Code, &c.Type, &c.Value, &c.MinOrderAmount, &c.MaxDiscountAmount, &c.UsageLimit,
		&c.UsedCount, &c.StartsAt, &c.ExpiresAt, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrCouponNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *Repository) Create(ctx context.Context, req MutationRequest) (*Coupon, error) {
	id := uuid.NewString()
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO coupons (
			id, code, type, value, min_order_amount, max_discount_amount, usage_limit,
			starts_at, expires_at, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, id, NormalizeCode(req.Code), req.Type, req.Value, req.MinOrderAmount, req.MaxDiscountAmount,
		req.UsageLimit, req.StartsAt, req.ExpiresAt, isActive)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errs.ErrCouponAlreadyExists
		}
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Update(ctx context.Context, id string, req MutationRequest) (*Coupon, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE coupons
		SET code = $2,
		    type = $3,
		    value = $4,
		    min_order_amount = $5,
		    max_discount_amount = $6,
		    usage_limit = $7,
		    starts_at = $8,
		    expires_at = $9,
		    is_active = $10,
		    updated_at = NOW()
		WHERE id = $1
	`, id, NormalizeCode(req.Code), req.Type, req.Value, req.MinOrderAmount, req.MaxDiscountAmount,
		req.UsageLimit, req.StartsAt, req.ExpiresAt, isActive)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errs.ErrCouponAlreadyExists
		}
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errs.ErrCouponNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM coupons WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrCouponNotFound
	}
	return nil
}

func scanCoupon(rows pgx.Rows, c *Coupon) error {
	return rows.Scan(
		&c.ID, &c.Code, &c.Type, &c.Value, &c.MinOrderAmount, &c.MaxDiscountAmount, &c.UsageLimit,
		&c.UsedCount, &c.StartsAt, &c.ExpiresAt, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
}

func buildWhere(filter ListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		clauses = append(clauses, fmt.Sprintf("LOWER(code) LIKE $%d", len(args)))
	}
	if filter.Type != "" {
		args = append(args, filter.Type)
		clauses = append(clauses, fmt.Sprintf("type = $%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		clauses = append(clauses, fmt.Sprintf("is_active = $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func sortClause(sort string) string {
	switch sort {
	case SortOldest:
		return "created_at ASC, id ASC"
	case "", SortNewest:
		return "created_at DESC, id DESC"
	default:
		return "created_at DESC, id DESC"
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
