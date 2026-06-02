//go:build integration

package order

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"stylemind/internal/database"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrderCheckoutFlowIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New error = %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations error = %v", err)
	}

	userID := uuid.NewString()
	otherUserID := uuid.NewString()
	categoryID := uuid.NewString()
	productID := uuid.NewString()
	cartID := uuid.NewString()

	defer cleanupOrderIntegrationRows(ctx, t, db, userID, otherUserID, categoryID, productID)

	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Integration User', 'hash', 'user'),
		       ($3, $4, 'Other User', 'hash', 'user')
	`, userID, "order-"+userID+"@example.com", otherUserID, "order-"+otherUserID+"@example.com"); err != nil {
		t.Fatalf("insert users error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
	`, categoryID, "Integration Category "+categoryID[:8], "integration-"+categoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1, 'Integration Product', 'Integration product description', 100000, 5, $2, 'minimal', 'black', 'https://example.com/image.jpg')
	`, productID, categoryID); err != nil {
		t.Fatalf("insert product error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO carts (id, user_id)
		VALUES ($1, $2)
	`, cartID, userID); err != nil {
		t.Fatalf("insert cart error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO cart_items (id, cart_id, product_id, quantity)
		VALUES ($1, $2, $3, 2)
	`, uuid.NewString(), cartID, productID); err != nil {
		t.Fatalf("insert cart item error = %v", err)
	}

	service := NewService(NewRepository(db))
	order, err := service.Checkout(ctx, userID)
	if err != nil {
		t.Fatalf("Checkout error = %v", err)
	}
	if order.Status != StatusPending {
		t.Fatalf("order.Status = %q, want pending", order.Status)
	}
	if order.TotalAmount != 200000 {
		t.Fatalf("order.TotalAmount = %v, want 200000", order.TotalAmount)
	}
	if len(order.Items) != 1 || order.Items[0].Quantity != 2 {
		t.Fatalf("order.Items = %+v, want one item quantity 2", order.Items)
	}

	var stock int
	if err := db.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, productID).Scan(&stock); err != nil {
		t.Fatalf("select stock error = %v", err)
	}
	if stock != 3 {
		t.Fatalf("stock = %d, want 3", stock)
	}

	var cartItemCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM cart_items WHERE cart_id = $1`, cartID).Scan(&cartItemCount); err != nil {
		t.Fatalf("select cart item count error = %v", err)
	}
	if cartItemCount != 0 {
		t.Fatalf("cart item count = %d, want 0", cartItemCount)
	}

	if _, err := service.GetMyOrder(ctx, otherUserID, order.ID); !errors.Is(err, errs.ErrOrderNotFound) {
		t.Fatalf("GetMyOrder for other user err = %v, want ErrOrderNotFound", err)
	}
	if _, err := service.UpdateStatus(ctx, order.ID, StatusShipping); !errors.Is(err, errs.ErrInvalidOrderStatusTransition) {
		t.Fatalf("UpdateStatus pending->shipping err = %v, want ErrInvalidOrderStatusTransition", err)
	}
	if _, err := service.UpdateStatus(ctx, order.ID, StatusPaid); err != nil {
		t.Fatalf("UpdateStatus pending->paid error = %v", err)
	}
	if _, err := service.UpdateStatus(ctx, order.ID, StatusShipping); err != nil {
		t.Fatalf("UpdateStatus paid->shipping error = %v", err)
	}
	if _, err := service.UpdateStatus(ctx, order.ID, StatusCompleted); err != nil {
		t.Fatalf("UpdateStatus shipping->completed error = %v", err)
	}
	if _, err := service.UpdateStatus(ctx, order.ID, StatusCancelled); !errors.Is(err, errs.ErrInvalidOrderStatusTransition) {
		t.Fatalf("UpdateStatus completed->cancelled err = %v, want ErrInvalidOrderStatusTransition", err)
	}
}

func cleanupOrderIntegrationRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID, otherUserID, categoryID, productID string) {
	t.Helper()

	_, _ = db.Exec(ctx, `DELETE FROM order_items WHERE product_id = $1`, productID)
	_, _ = db.Exec(ctx, `DELETE FROM orders WHERE user_id IN ($1, $2)`, userID, otherUserID)
	_, _ = db.Exec(ctx, `DELETE FROM cart_items WHERE product_id = $1`, productID)
	_, _ = db.Exec(ctx, `DELETE FROM carts WHERE user_id IN ($1, $2)`, userID, otherUserID)
	_, _ = db.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, categoryID)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, userID, otherUserID)
}
