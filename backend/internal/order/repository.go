package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"stylemind/internal/coupon"
	"stylemind/internal/errs"
	"stylemind/internal/inventory"
	"time"

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

func (r *Repository) GetOrCreateCart(ctx context.Context, userID string) (string, error) {
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

func (r *Repository) CreateOrderFromCart(ctx context.Context, userID, cartID string, details CheckoutDetails) (string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT ci.id, ci.product_id, p.name, p.image_url, p.style, p.color, p.price, p.stock, ci.quantity
		FROM cart_items ci
		JOIN products p ON p.id = ci.product_id
		WHERE ci.cart_id = $1
		ORDER BY ci.created_at DESC
		FOR UPDATE OF ci, p
	`, cartID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	items := make([]CheckoutItem, 0)
	for rows.Next() {
		var item CheckoutItem
		if err := rows.Scan(
			&item.CartItemID, &item.ProductID, &item.Name, &item.ImageURL, &item.Style, &item.Color,
			&item.Price, &item.Stock, &item.Quantity,
		); err != nil {
			return "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", errs.ErrCartEmpty
	}

	now := time.Now()
	reservationExpiresAt := now.Add(inventory.DefaultReservationTTL)
	reservationIDs := make([]string, 0, len(items))
	subtotal := 0.0
	for _, item := range items {
		activeReservations, err := r.activeReservationsForProduct(ctx, tx, item.ProductID, now)
		if err != nil {
			return "", err
		}
		if !canReserveQuantity(item.Stock, activeReservations, item.Quantity) {
			return "", errs.ErrInsufficientStock
		}
		reservationID, err := r.createReservation(ctx, tx, userID, item.ProductID, item.Quantity, reservationExpiresAt)
		if err != nil {
			return "", err
		}
		reservationIDs = append(reservationIDs, reservationID)
		subtotal += item.Price * float64(item.Quantity)
	}

	discountAmount := 0.0
	couponID := sql.NullString{}
	couponCode := sql.NullString{}
	if details.CouponCode != "" {
		appliedCoupon, err := r.lockCouponByCode(ctx, tx, details.CouponCode)
		if err != nil {
			return "", err
		}
		if err := coupon.ValidateForSubtotal(*appliedCoupon, subtotal, time.Now()); err != nil {
			return "", err
		}
		discountAmount = coupon.CalculateDiscount(subtotal, *appliedCoupon)
		tag, err := tx.Exec(ctx, `
			UPDATE coupons
			SET used_count = used_count + 1, updated_at = NOW()
			WHERE id = $1 AND (usage_limit IS NULL OR used_count < usage_limit)
		`, appliedCoupon.ID)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return "", errs.ErrCouponUsageLimitReached
		}
		couponID = sql.NullString{String: appliedCoupon.ID, Valid: true}
		couponCode = sql.NullString{String: appliedCoupon.Code, Valid: true}
	}
	total := subtotal - discountAmount

	orderID := uuid.NewString()
	_, err = tx.Exec(ctx, `
		INSERT INTO orders (
			id, user_id, status, payment_status, subtotal_amount, discount_amount, coupon_id, coupon_code, total_amount,
			recipient_name, phone, address_line, city, district, note, shipping_method, payment_method
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, orderID, userID, StatusPending, InitialPaymentStatus(details.PaymentMethod), subtotal, discountAmount, couponID, couponCode, total,
		details.RecipientName, details.Phone, details.AddressLine, details.City, details.District, details.Note, details.ShippingMethod, details.PaymentMethod)
	if err != nil {
		return "", err
	}

	for _, item := range items {
		subtotal := item.Price * float64(item.Quantity)
		_, err := tx.Exec(ctx, `
			INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.NewString(), orderID, item.ProductID, item.Quantity, item.Price, subtotal)
		if err != nil {
			return "", err
		}

		tag, err := tx.Exec(ctx, `
			UPDATE products
			SET stock = stock - $2, updated_at = NOW()
			WHERE id = $1 AND stock >= $2
		`, item.ProductID, item.Quantity)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			return "", errs.ErrInsufficientStock
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM cart_items WHERE cart_id = $1`, cartID)
	if err != nil {
		return "", err
	}

	if err := r.deleteReservations(ctx, tx, reservationIDs); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return orderID, nil
}

func (r *Repository) activeReservationsForProduct(ctx context.Context, tx pgx.Tx, productID string, now time.Time) (int, error) {
	var quantity int
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(quantity), 0)::int
		FROM inventory_reservations
		WHERE product_id = $1 AND expires_at > $2
	`, productID, now).Scan(&quantity)
	return quantity, err
}

func (r *Repository) createReservation(ctx context.Context, tx pgx.Tx, userID, productID string, quantity int, expiresAt time.Time) (string, error) {
	id := uuid.NewString()
	_, err := tx.Exec(ctx, `
		INSERT INTO inventory_reservations (id, user_id, product_id, quantity, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, id, userID, productID, quantity, expiresAt)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) deleteReservations(ctx context.Context, tx pgx.Tx, ids []string) error {
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `DELETE FROM inventory_reservations WHERE id = $1`, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) lockCouponByCode(ctx context.Context, tx pgx.Tx, code string) (*coupon.Coupon, error) {
	c := &coupon.Coupon{}
	err := tx.QueryRow(ctx, `
		SELECT id, code, type, value, min_order_amount, max_discount_amount, usage_limit,
		       used_count, starts_at, expires_at, is_active, created_at, updated_at
		FROM coupons
		WHERE LOWER(code) = LOWER($1)
		FOR UPDATE
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

func (r *Repository) ListOrdersByUser(ctx context.Context, userID string, limit, offset int) ([]OrderResponse, int64, error) {
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, status, payment_status, subtotal_amount, discount_amount,
		       COALESCE(coupon_id::text, ''), COALESCE(coupon_code, ''), total_amount,
		       COALESCE(recipient_name, ''), COALESCE(phone, ''), COALESCE(address_line, ''),
		       COALESCE(city, ''), COALESCE(district, ''), COALESCE(note, ''),
		       COALESCE(shipping_method, ''), COALESCE(payment_method, ''),
		       created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]OrderResponse, 0)
	orderIDs := make([]string, 0)
	for rows.Next() {
		var o OrderResponse
		if err := scanOrderResponse(rows, &o); err != nil {
			return nil, 0, err
		}
		o.Items = make([]OrderItem, 0)
		out = append(out, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(orderIDs) == 0 {
		return out, total, nil
	}

	itemsByOrderID, err := r.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if items, ok := itemsByOrderID[out[i].ID]; ok {
			out[i].Items = items
		}
	}
	return out, total, nil
}

func (r *Repository) ListOrders(ctx context.Context, filter AdminOrderFilter, limit, offset int) ([]OrderResponse, int64, error) {
	var total int64
	whereSQL, args := buildAdminOrderWhere(filter)
	countQuery := `
		SELECT COUNT(*)
		FROM orders o
		JOIN users u ON u.id = o.user_id
	` + whereSQL
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	queryArgs := append(append([]any{}, args...), limit, offset)
	query := `
		SELECT o.id, o.user_id, u.email, u.full_name, u.role, o.status, o.payment_status,
		       o.subtotal_amount, o.discount_amount, COALESCE(o.coupon_id::text, ''), COALESCE(o.coupon_code, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.phone, ''), COALESCE(o.address_line, ''),
		       COALESCE(o.city, ''), COALESCE(o.district, ''), COALESCE(o.note, ''),
		       COALESCE(o.shipping_method, ''), COALESCE(o.payment_method, ''),
		       o.created_at, o.updated_at
		FROM orders o
		JOIN users u ON u.id = o.user_id
	` + whereSQL + `
		ORDER BY ` + adminOrderSortClause(filter.Sort) + `
		LIMIT ` + limitPlaceholder + ` OFFSET ` + offsetPlaceholder
	rows, err := r.db.Query(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]OrderResponse, 0)
	orderIDs := make([]string, 0)
	for rows.Next() {
		var o OrderResponse
		var user OrderUser
		if err := rows.Scan(
			&o.ID, &o.UserID, &user.Email, &user.FullName, &user.Role,
			&o.Status, &o.PaymentStatus, &o.SubtotalAmount, &o.DiscountAmount, &o.CouponID, &o.CouponCode, &o.TotalAmount,
			&o.RecipientName, &o.Phone, &o.AddressLine, &o.City, &o.District, &o.Note, &o.ShippingMethod, &o.PaymentMethod,
			&o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		user.ID = o.UserID
		o.User = &user
		o.Items = make([]OrderItem, 0)
		out = append(out, o)
		orderIDs = append(orderIDs, o.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(orderIDs) == 0 {
		return out, total, nil
	}

	itemsByOrderID, err := r.GetOrderItemsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if items, ok := itemsByOrderID[out[i].ID]; ok {
			out[i].Items = items
		}
	}
	return out, total, nil
}

func (r *Repository) GetOrderByIDForUser(ctx context.Context, orderID, userID string) (*OrderResponse, error) {
	o := &OrderResponse{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, status, payment_status, subtotal_amount, discount_amount,
		       COALESCE(coupon_id::text, ''), COALESCE(coupon_code, ''), total_amount,
		       COALESCE(recipient_name, ''), COALESCE(phone, ''), COALESCE(address_line, ''),
		       COALESCE(city, ''), COALESCE(district, ''), COALESCE(note, ''),
		       COALESCE(shipping_method, ''), COALESCE(payment_method, ''),
		       created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID).Scan(
		&o.ID, &o.UserID, &o.Status, &o.PaymentStatus, &o.SubtotalAmount, &o.DiscountAmount, &o.CouponID, &o.CouponCode, &o.TotalAmount,
		&o.RecipientName, &o.Phone, &o.AddressLine, &o.City, &o.District, &o.Note, &o.ShippingMethod, &o.PaymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, err
	}
	items, err := r.GetOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (r *Repository) GetOrderByID(ctx context.Context, orderID string) (*OrderResponse, error) {
	o := &OrderResponse{}
	var user OrderUser
	err := r.db.QueryRow(ctx, `
		SELECT o.id, o.user_id, u.email, u.full_name, u.role, o.status, o.payment_status,
		       o.subtotal_amount, o.discount_amount, COALESCE(o.coupon_id::text, ''), COALESCE(o.coupon_code, ''), o.total_amount,
		       COALESCE(o.recipient_name, ''), COALESCE(o.phone, ''), COALESCE(o.address_line, ''),
		       COALESCE(o.city, ''), COALESCE(o.district, ''), COALESCE(o.note, ''),
		       COALESCE(o.shipping_method, ''), COALESCE(o.payment_method, ''),
		       o.created_at, o.updated_at
		FROM orders o
		JOIN users u ON u.id = o.user_id
		WHERE o.id = $1
	`, orderID).Scan(
		&o.ID, &o.UserID, &user.Email, &user.FullName, &user.Role,
		&o.Status, &o.PaymentStatus, &o.SubtotalAmount, &o.DiscountAmount, &o.CouponID, &o.CouponCode, &o.TotalAmount,
		&o.RecipientName, &o.Phone, &o.AddressLine, &o.City, &o.District, &o.Note, &o.ShippingMethod, &o.PaymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrOrderNotFound
		}
		return nil, err
	}
	user.ID = o.UserID
	o.User = &user
	items, err := r.GetOrderItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID, status string, allowedCurrentStatuses []string) error {
	if len(allowedCurrentStatuses) == 0 {
		return errs.ErrInvalidOrderStatus
	}

	args := []any{orderID, status}
	placeholders := make([]string, len(allowedCurrentStatuses))
	for i, currentStatus := range allowedCurrentStatuses {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, currentStatus)
	}

	query := fmt.Sprintf(`
		UPDATE orders
		SET status = $2, updated_at = NOW()
		WHERE id = $1 AND status IN (%s)
	`, strings.Join(placeholders, ","))

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetOrderStatus(ctx, orderID); err != nil {
			return err
		}
		return errs.ErrInvalidOrderStatusTransition
	}
	return nil
}

func (r *Repository) UpdatePaymentStatus(ctx context.Context, orderID, paymentStatus string, allowedCurrentStatuses []string) error {
	if len(allowedCurrentStatuses) == 0 {
		return errs.ErrInvalidPaymentStatus
	}

	args := []any{orderID, paymentStatus}
	placeholders := make([]string, len(allowedCurrentStatuses))
	for i, currentStatus := range allowedCurrentStatuses {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args = append(args, currentStatus)
	}

	query := fmt.Sprintf(`
		UPDATE orders
		SET payment_status = $2, updated_at = NOW()
		WHERE id = $1 AND payment_status IN (%s)
	`, strings.Join(placeholders, ","))

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.GetPaymentStatus(ctx, orderID); err != nil {
			return err
		}
		return errs.ErrInvalidPaymentStatusTransition
	}
	return nil
}

func (r *Repository) GetOrderStatus(ctx context.Context, orderID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrOrderNotFound
		}
		return "", err
	}
	return status, nil
}

func (r *Repository) GetPaymentStatus(ctx context.Context, orderID string) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `SELECT payment_status FROM orders WHERE id = $1`, orderID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrOrderNotFound
		}
		return "", err
	}
	return status, nil
}

func scanOrderResponse(rows pgx.Rows, o *OrderResponse) error {
	return rows.Scan(
		&o.ID, &o.UserID, &o.Status, &o.PaymentStatus, &o.SubtotalAmount, &o.DiscountAmount, &o.CouponID, &o.CouponCode, &o.TotalAmount,
		&o.RecipientName, &o.Phone, &o.AddressLine, &o.City, &o.District, &o.Note, &o.ShippingMethod, &o.PaymentMethod,
		&o.CreatedAt, &o.UpdatedAt,
	)
}

func buildAdminOrderWhere(filter AdminOrderFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)

	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "LOWER(o.id::text) LIKE "+placeholder)
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("o.status = $%d", len(args)))
	}
	if filter.UserID != "" {
		args = append(args, filter.UserID)
		clauses = append(clauses, fmt.Sprintf("o.user_id = $%d::uuid", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("o.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("o.created_at <= $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func adminOrderSortClause(sort string) string {
	switch sort {
	case AdminOrderSortOldest:
		return "o.created_at ASC, o.id ASC"
	case "", AdminOrderSortNewest:
		return "o.created_at DESC, o.id DESC"
	default:
		return "o.created_at DESC, o.id DESC"
	}
}

func (r *Repository) GetOrderItems(ctx context.Context, orderID string) ([]OrderItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT oi.id, oi.product_id, oi.quantity, oi.unit_price, oi.subtotal,
		       p.id, p.name, p.image_url, p.style, p.color
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
		ORDER BY oi.created_at ASC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OrderItem, 0)
	for rows.Next() {
		var item OrderItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal,
			&item.Product.ID, &item.Product.Name, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) GetOrderItemsByOrderIDs(ctx context.Context, orderIDs []string) (map[string][]OrderItem, error) {
	itemsByOrderID := make(map[string][]OrderItem, len(orderIDs))
	if len(orderIDs) == 0 {
		return itemsByOrderID, nil
	}

	placeholders := make([]string, len(orderIDs))
	args := make([]any, len(orderIDs))
	for i, orderID := range orderIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orderID
		itemsByOrderID[orderID] = make([]OrderItem, 0)
	}

	query := fmt.Sprintf(`
		SELECT oi.order_id, oi.id, oi.product_id, oi.quantity, oi.unit_price, oi.subtotal,
		       p.id, p.name, p.image_url, p.style, p.color
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id IN (%s)
		ORDER BY oi.created_at ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var orderID string
		var item OrderItem
		if err := rows.Scan(
			&orderID, &item.ID, &item.ProductID, &item.Quantity, &item.UnitPrice, &item.Subtotal,
			&item.Product.ID, &item.Product.Name, &item.Product.ImageURL, &item.Product.Style, &item.Product.Color,
		); err != nil {
			return nil, err
		}
		itemsByOrderID[orderID] = append(itemsByOrderID[orderID], item)
	}
	return itemsByOrderID, rows.Err()
}
