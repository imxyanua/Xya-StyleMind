//go:build integration

package wishlist

import (
	"context"
	"os"
	"testing"
	"time"

	"stylemind/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWishlistRepositoryIntegration(t *testing.T) {
	ctx, db := openWishlistIntegrationDB(t)
	repo := NewRepository(db)

	fixture := newWishlistFixture()
	defer cleanupWishlistIntegrationRows(ctx, t, db, fixture)
	insertWishlistFixture(ctx, t, db, fixture)

	exists, err := repo.ProductExists(ctx, fixture.ProductID)
	if err != nil {
		t.Fatalf("ProductExists error = %v", err)
	}
	if !exists {
		t.Fatal("ProductExists = false, want true")
	}

	missingExists, err := repo.ProductExists(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("ProductExists missing error = %v", err)
	}
	if missingExists {
		t.Fatal("ProductExists missing = true, want false")
	}

	if err := repo.AddProduct(ctx, fixture.UserID, fixture.ProductID); err != nil {
		t.Fatalf("AddProduct first error = %v", err)
	}
	if err := repo.AddProduct(ctx, fixture.UserID, fixture.ProductID); err != nil {
		t.Fatalf("AddProduct duplicate should be idempotent, got error = %v", err)
	}

	var count int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM wishlist_items
		WHERE user_id = $1 AND product_id = $2
	`, fixture.UserID, fixture.ProductID).Scan(&count); err != nil {
		t.Fatalf("count wishlist items error = %v", err)
	}
	if count != 1 {
		t.Fatalf("wishlist duplicate count = %d, want 1", count)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO wishlist_items (id, user_id, product_id)
		VALUES ($1, $2, $3)
	`, uuid.NewString(), fixture.UserID, fixture.ProductID); err == nil {
		t.Fatal("direct duplicate insert expected unique constraint error, got nil")
	}

	if err := repo.AddProduct(ctx, fixture.OtherUserID, fixture.OtherProductID); err != nil {
		t.Fatalf("AddProduct other user error = %v", err)
	}
	items, total, err := repo.ListByUser(ctx, fixture.UserID, 20, 0)
	if err != nil {
		t.Fatalf("ListByUser error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != fixture.UserID || items[0].ProductID != fixture.ProductID {
		t.Fatalf("items/total = %+v/%d, want only user wishlist item", items, total)
	}

	otherItems, otherTotal, err := repo.ListByUser(ctx, fixture.OtherUserID, 20, 0)
	if err != nil {
		t.Fatalf("ListByUser other error = %v", err)
	}
	if otherTotal != 1 || len(otherItems) != 1 || otherItems[0].UserID != fixture.OtherUserID {
		t.Fatalf("other items/total = %+v/%d, want only other user wishlist item", otherItems, otherTotal)
	}

	if err := repo.RemoveProduct(ctx, fixture.UserID, fixture.ProductID); err != nil {
		t.Fatalf("RemoveProduct existing error = %v", err)
	}
	if err := repo.RemoveProduct(ctx, fixture.UserID, fixture.ProductID); err != nil {
		t.Fatalf("RemoveProduct missing should be idempotent, got error = %v", err)
	}
	items, total, err = repo.ListByUser(ctx, fixture.UserID, 20, 0)
	if err != nil {
		t.Fatalf("ListByUser after remove error = %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("items/total after remove = %+v/%d, want empty", items, total)
	}
}

type wishlistIntegrationFixture struct {
	UserID         string
	OtherUserID    string
	CategoryID     string
	ProductID      string
	OtherProductID string
}

func newWishlistFixture() wishlistIntegrationFixture {
	return wishlistIntegrationFixture{
		UserID:         uuid.NewString(),
		OtherUserID:    uuid.NewString(),
		CategoryID:     uuid.NewString(),
		ProductID:      uuid.NewString(),
		OtherProductID: uuid.NewString(),
	}
}

func openWishlistIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func insertWishlistFixture(ctx context.Context, t *testing.T, db *pgxpool.Pool, fixture wishlistIntegrationFixture) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Wishlist User', 'hash', 'user'),
		       ($3, $4, 'Other Wishlist User', 'hash', 'user')
	`, fixture.UserID, "wishlist-"+fixture.UserID+"@example.com", fixture.OtherUserID, "wishlist-"+fixture.OtherUserID+"@example.com"); err != nil {
		t.Fatalf("insert users error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
	`, fixture.CategoryID, "Wishlist Category "+fixture.CategoryID[:8], "wishlist-"+fixture.CategoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1, 'Wishlist Product', 'Wishlist product description', 320000, 5, $2, 'minimal', 'black', 'https://example.com/wishlist-product.jpg'),
		       ($3, 'Other Wishlist Product', 'Other wishlist product description', 450000, 3, $2, 'casual', 'white', 'https://example.com/other-wishlist-product.jpg')
	`, fixture.ProductID, fixture.CategoryID, fixture.OtherProductID); err != nil {
		t.Fatalf("insert products error = %v", err)
	}
}

func cleanupWishlistIntegrationRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, fixture wishlistIntegrationFixture) {
	t.Helper()

	_, _ = db.Exec(ctx, `DELETE FROM wishlist_items WHERE user_id IN ($1, $2)`, fixture.UserID, fixture.OtherUserID)
	_, _ = db.Exec(ctx, `DELETE FROM products WHERE id IN ($1, $2)`, fixture.ProductID, fixture.OtherProductID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, fixture.CategoryID)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, fixture.UserID, fixture.OtherUserID)
}
