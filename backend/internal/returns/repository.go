package returns

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"stylemind/internal/errs"
	"stylemind/internal/order"

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

func (r *Repository) Create(ctx context.Context, userID, orderID, reason string) (*Request, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	orderInfo, err := getOrderForUser(ctx, tx, orderID, userID)
	if err != nil {
		return nil, err
	}
	if !isReturnEligibleOrderStatus(orderInfo.Status) {
		return nil, errs.ErrReturnRequestNotAllowed
	}

	item := &Request{Order: orderInfo}
	err = tx.QueryRow(ctx, `
		INSERT INTO return_requests (id, order_id, user_id, reason, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, order_id, user_id, reason, status, COALESCE(admin_note, ''), created_at, updated_at
	`, uuid.NewString(), orderID, userID, reason, StatusRequested).Scan(
		&item.ID, &item.OrderID, &item.UserID, &item.Reason, &item.Status, &item.AdminNote, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, errs.ErrReturnRequestAlreadyExists
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Request, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM return_requests WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT rr.id, rr.order_id, rr.user_id, rr.reason, rr.status, COALESCE(rr.admin_note, ''), rr.created_at, rr.updated_at,
		       o.id, o.status, o.payment_status, COALESCE(o.payment_method, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.shipping_method, ''), o.created_at
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		WHERE rr.user_id = $1
		ORDER BY rr.created_at DESC, rr.id DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Request, 0)
	for rows.Next() {
		item, err := scanRequestWithOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) ListAdmin(ctx context.Context, filter AdminFilter, limit, offset int) ([]Request, int64, error) {
	whereSQL, args := buildAdminWhere(filter)
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM return_requests rr`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(ctx, `
		SELECT rr.id, rr.order_id, rr.user_id, rr.reason, rr.status, COALESCE(rr.admin_note, ''), rr.created_at, rr.updated_at,
		       o.id, o.status, o.payment_status, COALESCE(o.payment_method, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.shipping_method, ''), o.created_at,
		       u.id, u.email, u.full_name, u.role
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		JOIN users u ON u.id = rr.user_id
	`+whereSQL+`
		ORDER BY `+sortClause(filter.Sort)+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Request, 0)
	for rows.Next() {
		item, err := scanRequestWithOrderAndUser(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetAdmin(ctx context.Context, id string) (*Request, error) {
	row := r.db.QueryRow(ctx, `
		SELECT rr.id, rr.order_id, rr.user_id, rr.reason, rr.status, COALESCE(rr.admin_note, ''), rr.created_at, rr.updated_at,
		       o.id, o.status, o.payment_status, COALESCE(o.payment_method, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.shipping_method, ''), o.created_at,
		       u.id, u.email, u.full_name, u.role
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		JOIN users u ON u.id = rr.user_id
		WHERE rr.id = $1
	`, id)
	item, err := scanRequestWithOrderAndUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrReturnRequestNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status, adminNote string) (*Request, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := getAdminForUpdate(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if current.Status == status {
		return current, tx.Commit(ctx)
	}
	if current.Status != StatusRequested {
		return nil, errs.ErrInvalidReturnRequestStatus
	}

	if status == StatusApproved {
		tag, err := tx.Exec(ctx, `
			UPDATE orders
			SET payment_status = $2, updated_at = NOW()
			WHERE id = $1 AND payment_status IN ($3, $4)
		`, current.OrderID, order.PaymentStatusRefunded, order.PaymentStatusPaid, order.PaymentStatusRefunded)
		if err != nil {
			return nil, err
		}
		if tag.RowsAffected() == 0 {
			return nil, errs.ErrInvalidPaymentStatusTransition
		}
	}

	if _, err := updateReturnStatus(ctx, tx, id, status, adminNote); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetAdmin(ctx, id)
}

func getOrderForUser(ctx context.Context, tx pgx.Tx, orderID, userID string) (*OrderInfo, error) {
	item := &OrderInfo{}
	err := tx.QueryRow(ctx, `
		SELECT id, status, payment_status, COALESCE(payment_method, ''), total_amount,
		       COALESCE(recipient_name, ''), COALESCE(shipping_method, ''), created_at
		FROM orders
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, orderID, userID).Scan(&item.ID, &item.Status, &item.PaymentStatus, &item.PaymentMethod, &item.TotalAmount, &item.RecipientName, &item.ShippingMethod, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, err
	}
	return item, nil
}

func getAdminForUpdate(ctx context.Context, tx pgx.Tx, id string) (*Request, error) {
	row := tx.QueryRow(ctx, `
		SELECT rr.id, rr.order_id, rr.user_id, rr.reason, rr.status, COALESCE(rr.admin_note, ''), rr.created_at, rr.updated_at,
		       o.id, o.status, o.payment_status, COALESCE(o.payment_method, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.shipping_method, ''), o.created_at,
		       u.id, u.email, u.full_name, u.role
		FROM return_requests rr
		JOIN orders o ON o.id = rr.order_id
		JOIN users u ON u.id = rr.user_id
		WHERE rr.id = $1
		FOR UPDATE OF rr, o
	`, id)
	item, err := scanRequestWithOrderAndUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrReturnRequestNotFound
		}
		return nil, err
	}
	return &item, nil
}

func updateReturnStatus(ctx context.Context, tx pgx.Tx, id, status, adminNote string) (*Request, error) {
	row := tx.QueryRow(ctx, `
		UPDATE return_requests
		SET status = $2, admin_note = NULLIF($3, ''), updated_at = NOW()
		WHERE id = $1
		RETURNING id, order_id, user_id, reason, status, COALESCE(admin_note, ''), created_at, updated_at
	`, id, status, adminNote)
	item := &Request{}
	if err := row.Scan(&item.ID, &item.OrderID, &item.UserID, &item.Reason, &item.Status, &item.AdminNote, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrReturnRequestNotFound
		}
		return nil, err
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRequestWithOrder(row rowScanner) (Request, error) {
	var item Request
	var orderInfo OrderInfo
	if err := row.Scan(
		&item.ID, &item.OrderID, &item.UserID, &item.Reason, &item.Status, &item.AdminNote, &item.CreatedAt, &item.UpdatedAt,
		&orderInfo.ID, &orderInfo.Status, &orderInfo.PaymentStatus, &orderInfo.PaymentMethod, &orderInfo.TotalAmount,
		&orderInfo.RecipientName, &orderInfo.ShippingMethod, &orderInfo.CreatedAt,
	); err != nil {
		return Request{}, err
	}
	item.Order = &orderInfo
	return item, nil
}

func scanRequestWithOrderAndUser(row rowScanner) (Request, error) {
	var item Request
	var orderInfo OrderInfo
	var user UserInfo
	if err := row.Scan(
		&item.ID, &item.OrderID, &item.UserID, &item.Reason, &item.Status, &item.AdminNote, &item.CreatedAt, &item.UpdatedAt,
		&orderInfo.ID, &orderInfo.Status, &orderInfo.PaymentStatus, &orderInfo.PaymentMethod, &orderInfo.TotalAmount,
		&orderInfo.RecipientName, &orderInfo.ShippingMethod, &orderInfo.CreatedAt,
		&user.ID, &user.Email, &user.FullName, &user.Role,
	); err != nil {
		return Request{}, err
	}
	item.Order = &orderInfo
	item.User = &user
	return item, nil
}

func buildAdminWhere(filter AdminFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("rr.status = $%d", len(args)))
	}
	if filter.UserID != "" {
		args = append(args, filter.UserID)
		clauses = append(clauses, fmt.Sprintf("rr.user_id = $%d::uuid", len(args)))
	}
	if filter.OrderID != "" {
		args = append(args, filter.OrderID)
		clauses = append(clauses, fmt.Sprintf("rr.order_id = $%d::uuid", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("rr.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("rr.created_at <= $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func sortClause(sort string) string {
	if sort == SortOldest {
		return "rr.created_at ASC, rr.id ASC"
	}
	return "rr.created_at DESC, rr.id DESC"
}

func isReturnEligibleOrderStatus(status string) bool {
	return status == order.StatusPaid || status == order.StatusShipping || status == order.StatusCompleted
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
