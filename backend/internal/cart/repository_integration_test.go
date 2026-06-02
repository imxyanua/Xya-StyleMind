//go:build integration

package cart

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

func TestCartRepositoryIntegration(t *testing.T) {
	ctx, db := openCartIntegrationDB(t)
	repo := NewRepository(db)

	fixture := newCartFixture()
	defer cleanupCartIntegrationRows(ctx, t, db, fixture)
	insertCartFixture(ctx, t, db, fixture, 5)

	cartA, err := repo.GetOrCreateCart(ctx, fixture.UserID)
	if err != nil {
		t.Fatalf("GetOrCreateCart first error = %v", err)
	}
	cartB, err := repo.GetOrCreateCart(ctx, fixture.UserID)
	if err != nil {
		t.Fatalf("GetOrCreateCart second error = %v", err)
	}
	if cartA.ID != cartB.ID {
		t.Fatalf("cart IDs = %s/%s, want same active cart", cartA.ID, cartB.ID)
	}

	snapshot, err := repo.GetProductSnapshot(ctx, fixture.ProductID)
	if err != nil {
		t.Fatalf("GetProductSnapshot error = %v", err)
	}
	if snapshot.ID != fixture.ProductID || snapshot.Stock != 5 {
		t.Fatalf("snapshot = %+v, want product stock 5", snapshot)
	}

	if err := repo.AddOrIncrementCartItem(ctx, cartA.ID, fixture.ProductID, 2); err != nil {
		t.Fatalf("AddOrIncrementCartItem first error = %v", err)
	}
	if err := repo.AddOrIncrementCartItem(ctx, cartA.ID, fixture.ProductID, 3); err != nil {
		t.Fatalf("AddOrIncrementCartItem increment error = %v", err)
	}
	if err := repo.AddOrIncrementCartItem(ctx, cartA.ID, fixture.ProductID, 1); !errors.Is(err, errs.ErrInsufficientStock) {
		t.Fatalf("AddOrIncrementCartItem over stock err = %v, want ErrInsufficientStock", err)
	}

	items, err := repo.ListCartItems(ctx, cartA.ID)
	if err != nil {
		t.Fatalf("ListCartItems error = %v", err)
	}
	if len(items) != 1 || items[0].Quantity != 5 {
		t.Fatalf("items = %+v, want one item quantity 5", items)
	}

	if err := repo.UpdateCartItemQuantity(ctx, items[0].ID, 4); err != nil {
		t.Fatalf("UpdateCartItemQuantity error = %v", err)
	}
	item, err := repo.GetCartItemByID(ctx, cartA.ID, items[0].ID)
	if err != nil {
		t.Fatalf("GetCartItemByID error = %v", err)
	}
	if item.Quantity != 4 {
		t.Fatalf("item.Quantity = %d, want 4", item.Quantity)
	}

	if err := repo.DeleteCartItem(ctx, cartA.ID, item.ID); err != nil {
		t.Fatalf("DeleteCartItem error = %v", err)
	}
	if _, err := repo.GetCartItemByID(ctx, cartA.ID, item.ID); !errors.Is(err, errs.ErrCartItemNotFound) {
		t.Fatalf("GetCartItemByID deleted err = %v, want ErrCartItemNotFound", err)
	}
	if err := repo.DeleteCartItem(ctx, cartA.ID, item.ID); !errors.Is(err, errs.ErrCartItemNotFound) {
		t.Fatalf("DeleteCartItem missing err = %v, want ErrCartItemNotFound", err)
	}
}

func TestCartRepositoryRejectsMissingProductIntegration(t *testing.T) {
	ctx, db := openCartIntegrationDB(t)
	repo := NewRepository(db)

	fixture := newCartFixture()
	defer cleanupCartIntegrationRows(ctx, t, db, fixture)
	insertCartUser(ctx, t, db, fixture.UserID)

	cart, err := repo.GetOrCreateCart(ctx, fixture.UserID)
	if err != nil {
		t.Fatalf("GetOrCreateCart error = %v", err)
	}
	missingProductID := uuid.NewString()
	if _, err := repo.GetProductSnapshot(ctx, missingProductID); !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("GetProductSnapshot missing err = %v, want ErrProductNotFound", err)
	}
	if err := repo.AddOrIncrementCartItem(ctx, cart.ID, missingProductID, 1); !errors.Is(err, errs.ErrInsufficientStock) {
		t.Fatalf("AddOrIncrementCartItem missing product err = %v, want ErrInsufficientStock", err)
	}
}

type cartIntegrationFixture struct {
	UserID     string
	CategoryID string
	ProductID  string
}

func newCartFixture() cartIntegrationFixture {
	return cartIntegrationFixture{
		UserID:     uuid.NewString(),
		CategoryID: uuid.NewString(),
		ProductID:  uuid.NewString(),
	}
}

func openCartIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func insertCartFixture(ctx context.Context, t *testing.T, db *pgxpool.Pool, fixture cartIntegrationFixture, stock int) {
	t.Helper()

	insertCartUser(ctx, t, db, fixture.UserID)
	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
	`, fixture.CategoryID, "Cart Test Category "+fixture.CategoryID[:8], "cart-test-"+fixture.CategoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1, 'Cart Product', 'Cart product description', 250000, $2, $3, 'casual', 'beige', 'https://example.com/cart-product.jpg')
	`, fixture.ProductID, stock, fixture.CategoryID); err != nil {
		t.Fatalf("insert product error = %v", err)
	}
}

func insertCartUser(ctx context.Context, t *testing.T, db *pgxpool.Pool, userID string) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Cart User', 'hash', 'user')
	`, userID, "cart-"+userID+"@example.com"); err != nil {
		t.Fatalf("insert user error = %v", err)
	}
}

func cleanupCartIntegrationRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, fixture cartIntegrationFixture) {
	t.Helper()

	_, _ = db.Exec(ctx, `DELETE FROM cart_items WHERE product_id = $1`, fixture.ProductID)
	_, _ = db.Exec(ctx, `DELETE FROM carts WHERE user_id = $1`, fixture.UserID)
	_, _ = db.Exec(ctx, `DELETE FROM products WHERE id = $1`, fixture.ProductID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, fixture.CategoryID)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id = $1`, fixture.UserID)
}
