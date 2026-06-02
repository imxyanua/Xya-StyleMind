//go:build integration

package order

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"stylemind/internal/database"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrderCheckoutFlowIntegration(t *testing.T) {
	ctx, db := openOrderIntegrationDB(t)

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
	orders, total, err := service.ListMyOrders(ctx, userID, 20, 0)
	if err != nil {
		t.Fatalf("ListMyOrders error = %v", err)
	}
	if total != 1 || len(orders) != 1 || len(orders[0].Items) != 1 {
		t.Fatalf("orders total/list = %d/%+v, want one order with one item", total, orders)
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

func TestOrderCheckoutRejectsInsufficientStockIntegration(t *testing.T) {
	ctx, db := openOrderIntegrationDB(t)

	userID := uuid.NewString()
	categoryID := uuid.NewString()
	productID := uuid.NewString()
	cartID := uuid.NewString()

	defer cleanupOrderIntegrationRows(ctx, t, db, userID, "", categoryID, productID)
	insertCheckoutFixture(ctx, t, db, checkoutFixture{
		UserID:     userID,
		CategoryID: categoryID,
		ProductID:  productID,
		CartID:     cartID,
		Stock:      1,
		Quantity:   2,
		Price:      100000,
	})

	service := NewService(NewRepository(db))
	if _, err := service.Checkout(ctx, userID); !errors.Is(err, errs.ErrInsufficientStock) {
		t.Fatalf("Checkout err = %v, want ErrInsufficientStock", err)
	}

	assertOrderCountForUser(ctx, t, db, userID, 0)
	assertProductStock(ctx, t, db, productID, 1)
	assertCartItemCount(ctx, t, db, cartID, 1)
}

func TestOrderCheckoutRollsBackWhenOrderItemInsertFailsIntegration(t *testing.T) {
	ctx, db := openOrderIntegrationDB(t)

	userID := uuid.NewString()
	categoryID := uuid.NewString()
	productID := uuid.NewString()
	cartID := uuid.NewString()
	triggerSuffix := strings.ReplaceAll(uuid.NewString(), "-", "_")

	defer cleanupOrderIntegrationRows(ctx, t, db, userID, "", categoryID, productID)
	defer dropOrderItemFailureTrigger(ctx, t, db, triggerSuffix)
	insertCheckoutFixture(ctx, t, db, checkoutFixture{
		UserID:     userID,
		CategoryID: categoryID,
		ProductID:  productID,
		CartID:     cartID,
		Stock:      5,
		Quantity:   2,
		Price:      100000,
	})
	createOrderItemFailureTrigger(ctx, t, db, triggerSuffix, productID)

	service := NewService(NewRepository(db))
	if _, err := service.Checkout(ctx, userID); err == nil {
		t.Fatal("Checkout expected forced order_items insert error, got nil")
	}

	assertOrderCountForUser(ctx, t, db, userID, 0)
	assertProductStock(ctx, t, db, productID, 5)
	assertCartItemCount(ctx, t, db, cartID, 1)
}

func TestListOrdersByUserEmptyHistoryIntegration(t *testing.T) {
	ctx, db := openOrderIntegrationDB(t)

	userID := uuid.NewString()
	defer cleanupOrderIntegrationRows(ctx, t, db, userID, "", "", "")
	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Empty Orders User', 'hash', 'user')
	`, userID, "empty-"+userID+"@example.com"); err != nil {
		t.Fatalf("insert user error = %v", err)
	}

	service := NewService(NewRepository(db))
	orders, total, err := service.ListMyOrders(ctx, userID, 20, 0)
	if err != nil {
		t.Fatalf("ListMyOrders error = %v", err)
	}
	if total != 0 || len(orders) != 0 {
		t.Fatalf("orders total/list = %d/%+v, want empty", total, orders)
	}
}

func TestConcurrentCheckoutDoesNotMakeStockNegativeIntegration(t *testing.T) {
	ctx, db := openOrderIntegrationDB(t)

	userAID := uuid.NewString()
	userBID := uuid.NewString()
	categoryID := uuid.NewString()
	productID := uuid.NewString()
	cartAID := uuid.NewString()
	cartBID := uuid.NewString()

	defer cleanupOrderIntegrationRows(ctx, t, db, userAID, userBID, categoryID, productID)
	insertCheckoutFixture(ctx, t, db, checkoutFixture{
		UserID:     userAID,
		CategoryID: categoryID,
		ProductID:  productID,
		CartID:     cartAID,
		Stock:      3,
		Quantity:   2,
		Price:      100000,
	})
	insertUserCartItem(ctx, t, db, userBID, "concurrent-"+userBID+"@example.com", cartBID, productID, 2)

	service := NewService(NewRepository(db))
	var wg sync.WaitGroup
	errsCh := make(chan error, 2)
	for _, userID := range []string{userAID, userBID} {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			_, err := service.Checkout(ctx, userID)
			errsCh <- err
		}(userID)
	}
	wg.Wait()
	close(errsCh)

	successCount := 0
	insufficientStockCount := 0
	for err := range errsCh {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, errs.ErrInsufficientStock):
			insufficientStockCount++
		default:
			t.Fatalf("unexpected checkout error = %v", err)
		}
	}
	if successCount != 1 || insufficientStockCount != 1 {
		t.Fatalf("checkout results success/insufficient = %d/%d, want 1/1", successCount, insufficientStockCount)
	}
	assertProductStock(ctx, t, db, productID, 1)
}

func openOrderIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pgxpool.New error = %v", err)
	}
	t.Cleanup(db.Close)

	if err := database.RunMigrations(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("RunMigrations error = %v", err)
	}
	return ctx, db
}

type checkoutFixture struct {
	UserID     string
	CategoryID string
	ProductID  string
	CartID     string
	Stock      int
	Quantity   int
	Price      float64
}

func insertCheckoutFixture(ctx context.Context, t *testing.T, db *pgxpool.Pool, fixture checkoutFixture) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Integration User', 'hash', 'user')
	`, fixture.UserID, "order-"+fixture.UserID+"@example.com"); err != nil {
		t.Fatalf("insert user error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
	`, fixture.CategoryID, "Integration Category "+fixture.CategoryID[:8], "integration-"+fixture.CategoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1, 'Integration Product', 'Integration product description', $2, $3, $4, 'minimal', 'black', 'https://example.com/image.jpg')
	`, fixture.ProductID, fixture.Price, fixture.Stock, fixture.CategoryID); err != nil {
		t.Fatalf("insert product error = %v", err)
	}
	insertUserCartItem(ctx, t, db, fixture.UserID, "order-"+fixture.UserID+"@example.com", fixture.CartID, fixture.ProductID, fixture.Quantity)
}

func insertUserCartItem(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID, email, cartID, productID string, quantity int) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Integration User', 'hash', 'user')
		ON CONFLICT (id) DO NOTHING
	`, userID, email); err != nil {
		t.Fatalf("insert user cart owner error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO carts (id, user_id)
		VALUES ($1, $2)
	`, cartID, userID); err != nil {
		t.Fatalf("insert cart error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO cart_items (id, cart_id, product_id, quantity)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), cartID, productID, quantity); err != nil {
		t.Fatalf("insert cart item error = %v", err)
	}
}

func createOrderItemFailureTrigger(ctx context.Context, t *testing.T, db *pgxpool.Pool, suffix, productID string) {
	t.Helper()

	functionName := "fail_order_items_insert_" + suffix
	triggerName := "trg_fail_order_items_insert_" + suffix
	sql := fmt.Sprintf(`
		CREATE FUNCTION %s() RETURNS trigger AS $$
		BEGIN
			IF NEW.product_id = '%s' THEN
				RAISE EXCEPTION 'forced order item insert failure';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER %s
		BEFORE INSERT ON order_items
		FOR EACH ROW EXECUTE FUNCTION %s();
	`, functionName, productID, triggerName, functionName)
	if _, err := db.Exec(ctx, sql); err != nil {
		t.Fatalf("create order item failure trigger error = %v", err)
	}
}

func dropOrderItemFailureTrigger(ctx context.Context, t *testing.T, db *pgxpool.Pool, suffix string) {
	t.Helper()

	functionName := "fail_order_items_insert_" + suffix
	triggerName := "trg_fail_order_items_insert_" + suffix
	sql := fmt.Sprintf(`
		DROP TRIGGER IF EXISTS %s ON order_items;
		DROP FUNCTION IF EXISTS %s();
	`, triggerName, functionName)
	if _, err := db.Exec(ctx, sql); err != nil {
		t.Fatalf("drop order item failure trigger error = %v", err)
	}
}

func assertOrderCountForUser(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID string, want int) {
	t.Helper()

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID).Scan(&count); err != nil {
		t.Fatalf("select order count error = %v", err)
	}
	if count != want {
		t.Fatalf("order count = %d, want %d", count, want)
	}
}

func assertProductStock(ctx context.Context, t *testing.T, db *pgxpool.Pool, productID string, want int) {
	t.Helper()

	var stock int
	if err := db.QueryRow(ctx, `SELECT stock FROM products WHERE id = $1`, productID).Scan(&stock); err != nil {
		t.Fatalf("select stock error = %v", err)
	}
	if stock != want {
		t.Fatalf("stock = %d, want %d", stock, want)
	}
}

func assertCartItemCount(ctx context.Context, t *testing.T, db *pgxpool.Pool, cartID string, want int) {
	t.Helper()

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM cart_items WHERE cart_id = $1`, cartID).Scan(&count); err != nil {
		t.Fatalf("select cart item count error = %v", err)
	}
	if count != want {
		t.Fatalf("cart item count = %d, want %d", count, want)
	}
}

func cleanupOrderIntegrationRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID, otherUserID, categoryID, productID string) {
	t.Helper()

	userIDs := []string{userID}
	if otherUserID != "" {
		userIDs = append(userIDs, otherUserID)
	}

	if productID != "" {
		_, _ = db.Exec(ctx, `DELETE FROM order_items WHERE product_id = $1`, productID)
		_, _ = db.Exec(ctx, `DELETE FROM cart_items WHERE product_id = $1`, productID)
		_, _ = db.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
	}
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		_, _ = db.Exec(ctx, `DELETE FROM orders WHERE user_id = $1`, id)
		_, _ = db.Exec(ctx, `DELETE FROM carts WHERE user_id = $1`, id)
		_, _ = db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	}
	if categoryID != "" {
		_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, categoryID)
	}
}
