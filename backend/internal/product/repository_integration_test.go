//go:build integration

package product

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

func TestProductRepositoryIntegration(t *testing.T) {
	ctx, db := openProductIntegrationDB(t)
	repo := NewRepository(db)

	categoryID := uuid.NewString()
	defer cleanupProductIntegrationRows(ctx, t, db, categoryID)
	insertProductCategory(ctx, t, db, categoryID)

	product, err := repo.Create(ctx, CreateProductRequest{
		Name:        "  Integration Hoodie  ",
		Description: "Warm oversized hoodie for integration tests",
		Price:       350000,
		Stock:       7,
		CategoryID:  categoryID,
		Style:       " Streetwear ",
		Color:       " Black ",
		ImageURL:    "https://example.com/hoodie.jpg",
	})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if product.Name != "Integration Hoodie" || product.Style != "streetwear" || product.Color != "black" {
		t.Fatalf("created product normalization failed: %+v", product)
	}

	byID, err := repo.GetByID(ctx, product.ID)
	if err != nil {
		t.Fatalf("GetByID error = %v", err)
	}
	if byID.ID != product.ID {
		t.Fatalf("GetByID id = %s, want %s", byID.ID, product.ID)
	}

	items, total, err := repo.List(ctx, ListFilter{Style: "STREETWEAR", Color: "BLACK", CategoryID: categoryID}, 20, 0)
	if err != nil {
		t.Fatalf("List filtered error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != product.ID {
		t.Fatalf("filtered list total/items = %d/%+v, want created product", total, items)
	}

	injectedStyle := "streetwear' OR '1'='1"
	items, total, err = repo.List(ctx, ListFilter{Style: injectedStyle}, 20, 0)
	if err != nil {
		t.Fatalf("List SQL injection-like style error = %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("SQL injection-like style matched rows total/items = %d/%d", total, len(items))
	}

	updated, err := repo.Update(ctx, product.ID, UpdateProductRequest{
		Name:        "Updated Hoodie",
		Description: "Updated integration hoodie description",
		Price:       420000,
		Stock:       4,
		CategoryID:  categoryID,
		Style:       "Minimal",
		Color:       "Gray",
		ImageURL:    "https://example.com/updated-hoodie.jpg",
	})
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if updated.Price != 420000 || updated.Stock != 4 || updated.Style != "minimal" || updated.Color != "gray" {
		t.Fatalf("updated product = %+v, want normalized updated values", updated)
	}

	if err := repo.Delete(ctx, product.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := repo.GetByID(ctx, product.ID); !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("GetByID deleted err = %v, want ErrProductNotFound", err)
	}
	if err := repo.Delete(ctx, product.ID); !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("Delete missing err = %v, want ErrProductNotFound", err)
	}
}

func TestProductRepositoryRejectsInvalidDatabaseValuesIntegration(t *testing.T) {
	ctx, db := openProductIntegrationDB(t)
	repo := NewRepository(db)

	categoryID := uuid.NewString()
	defer cleanupProductIntegrationRows(ctx, t, db, categoryID)
	insertProductCategory(ctx, t, db, categoryID)

	_, err := repo.Create(ctx, CreateProductRequest{
		Name:        "Invalid Product",
		Description: "Invalid product description",
		Price:       -1,
		Stock:       1,
		CategoryID:  categoryID,
		Style:       "minimal",
		Color:       "black",
		ImageURL:    "https://example.com/invalid.jpg",
	})
	if err == nil {
		t.Fatal("Create negative price expected database constraint error, got nil")
	}
}

func TestProductRepositoryAdvancedSearchFiltersIntegration(t *testing.T) {
	ctx, db := openProductIntegrationDB(t)
	repo := NewRepository(db)

	fixture := newProductSearchFixture()
	defer cleanupProductSearchRows(ctx, t, db, fixture)
	insertProductSearchFixture(ctx, t, db, fixture)

	items, total, err := repo.List(ctx, ListFilter{Query: "hoodie", Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List search query error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != fixture.HoodieID {
		t.Fatalf("search hoodie total/items = %d/%+v, want hoodie", total, items)
	}

	items, total, err = repo.List(ctx, ListFilter{CategoryID: fixture.CategoryAID, Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List category filter error = %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("category filter total/items = %d/%d, want 2", total, len(items))
	}

	minPrice := 250000.0
	maxPrice := 350000.0
	items, total, err = repo.List(ctx, ListFilter{MinPrice: &minPrice, MaxPrice: &maxPrice, Sort: SortPriceAsc}, 20, 0)
	if err != nil {
		t.Fatalf("List price filter error = %v", err)
	}
	if total != 2 || len(items) != 2 || items[0].ID != fixture.ShirtID || items[1].ID != fixture.HoodieID {
		t.Fatalf("price filter sorted items = total:%d items:%+v, want shirt then hoodie", total, items)
	}

	inStock := true
	items, total, err = repo.List(ctx, ListFilter{InStock: &inStock, Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List in_stock=true error = %v", err)
	}
	if total != 2 {
		t.Fatalf("in_stock total = %d, want 2", total)
	}
	inStock = false
	items, total, err = repo.List(ctx, ListFilter{InStock: &inStock, Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List in_stock=false error = %v", err)
	}
	if total != 1 || items[0].ID != fixture.PantsID {
		t.Fatalf("out of stock items = total:%d items:%+v, want pants", total, items)
	}

	minRating := 4.0
	items, total, err = repo.List(ctx, ListFilter{MinRating: &minRating, Sort: SortRatingDesc}, 20, 0)
	if err != nil {
		t.Fatalf("List min_rating error = %v", err)
	}
	if total != 1 || items[0].ID != fixture.HoodieID || items[0].AverageRating != 4.5 || items[0].ReviewCount != 2 {
		t.Fatalf("rating filter items = total:%d items:%+v, want hoodie with avg 4.5 count 2", total, items)
	}

	items, _, err = repo.List(ctx, ListFilter{Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List newest error = %v", err)
	}
	if len(items) < 3 || items[0].ID != fixture.ShirtID {
		t.Fatalf("newest first = %+v, want shirt first", items)
	}

	items, _, err = repo.List(ctx, ListFilter{Sort: SortPriceDesc}, 20, 0)
	if err != nil {
		t.Fatalf("List price_desc error = %v", err)
	}
	if len(items) < 3 || items[0].ID != fixture.PantsID {
		t.Fatalf("price_desc first = %+v, want pants first", items)
	}

	items, total, err = repo.List(ctx, ListFilter{Sort: SortPriceAsc}, 1, 1)
	if err != nil {
		t.Fatalf("List pagination error = %v", err)
	}
	if total != 3 || len(items) != 1 || items[0].ID != fixture.HoodieID {
		t.Fatalf("pagination total/items = %d/%+v, want hoodie on page 2 with limit 1", total, items)
	}

	injected := "streetwear' OR '1'='1"
	items, total, err = repo.List(ctx, ListFilter{Query: injected, Sort: SortNewest}, 20, 0)
	if err != nil {
		t.Fatalf("List SQL injection-like query error = %v", err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("SQL injection-like query matched rows total/items = %d/%d", total, len(items))
	}
}

func openProductIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func insertProductCategory(ctx context.Context, t *testing.T, db *pgxpool.Pool, categoryID string) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3)
	`, categoryID, "Product Test Category "+categoryID[:8], "product-test-"+categoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
}

func cleanupProductIntegrationRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, categoryID string) {
	t.Helper()

	_, _ = db.Exec(ctx, `DELETE FROM products WHERE category_id = $1`, categoryID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, categoryID)
}

type productSearchFixture struct {
	UserAID     string
	UserBID     string
	CategoryAID string
	CategoryBID string
	HoodieID    string
	PantsID     string
	ShirtID     string
	OrderAID    string
	OrderBID    string
	OrderCID    string
}

func newProductSearchFixture() productSearchFixture {
	return productSearchFixture{
		UserAID:     uuid.NewString(),
		UserBID:     uuid.NewString(),
		CategoryAID: uuid.NewString(),
		CategoryBID: uuid.NewString(),
		HoodieID:    uuid.NewString(),
		PantsID:     uuid.NewString(),
		ShirtID:     uuid.NewString(),
		OrderAID:    uuid.NewString(),
		OrderBID:    uuid.NewString(),
		OrderCID:    uuid.NewString(),
	}
}

func insertProductSearchFixture(ctx context.Context, t *testing.T, db *pgxpool.Pool, f productSearchFixture) {
	t.Helper()
	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Search User A', 'hash', 'user'),
		       ($3, $4, 'Search User B', 'hash', 'user')
	`, f.UserAID, "search-"+f.UserAID+"@example.com", f.UserBID, "search-"+f.UserBID+"@example.com"); err != nil {
		t.Fatalf("insert users error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO categories (id, name, slug)
		VALUES ($1, $2, $3),
		       ($4, $5, $6)
	`, f.CategoryAID, "Search Category A "+f.CategoryAID[:8], "search-a-"+f.CategoryAID[:8], f.CategoryBID, "Search Category B "+f.CategoryBID[:8], "search-b-"+f.CategoryBID[:8]); err != nil {
		t.Fatalf("insert categories error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url, created_at)
		VALUES ($1, 'Black Street Hoodie', 'Oversized cotton hoodie', 300000, 10, $2, 'streetwear', 'black', 'https://example.com/hoodie.jpg', NOW() - INTERVAL '2 hours'),
		       ($3, 'Beige Minimal Pants', 'Relaxed tapered trousers', 450000, 0, $2, 'minimal', 'beige', 'https://example.com/pants.jpg', NOW() - INTERVAL '3 hours'),
		       ($4, 'Blue Formal Shirt', 'Office shirt with crisp collar', 250000, 5, $5, 'formal', 'blue', 'https://example.com/shirt.jpg', NOW() - INTERVAL '1 hour')
	`, f.HoodieID, f.CategoryAID, f.PantsID, f.ShirtID, f.CategoryBID); err != nil {
		t.Fatalf("insert products error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO orders (id, user_id, status, total_amount)
		VALUES ($1, $2, 'completed', 300000),
		       ($3, $4, 'completed', 300000),
		       ($5, $2, 'completed', 450000)
	`, f.OrderAID, f.UserAID, f.OrderBID, f.UserBID, f.OrderCID); err != nil {
		t.Fatalf("insert orders error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, subtotal)
		VALUES ($1, $2, $3, 1, 300000, 300000),
		       ($4, $5, $3, 1, 300000, 300000),
		       ($6, $7, $8, 1, 450000, 450000)
	`, uuid.NewString(), f.OrderAID, f.HoodieID, uuid.NewString(), f.OrderBID, uuid.NewString(), f.OrderCID, f.PantsID); err != nil {
		t.Fatalf("insert order items error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO product_reviews (id, user_id, product_id, order_id, rating, comment)
		VALUES ($1, $2, $3, $4, 5, 'Great hoodie'),
		       ($5, $6, $3, $7, 4, 'Solid hoodie'),
		       ($8, $2, $9, $10, 3, 'Okay pants')
	`, uuid.NewString(), f.UserAID, f.HoodieID, f.OrderAID, uuid.NewString(), f.UserBID, f.OrderBID, uuid.NewString(), f.PantsID, f.OrderCID); err != nil {
		t.Fatalf("insert reviews error = %v", err)
	}
}

func cleanupProductSearchRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, f productSearchFixture) {
	t.Helper()
	_, _ = db.Exec(ctx, `DELETE FROM product_reviews WHERE product_id IN ($1, $2, $3)`, f.HoodieID, f.PantsID, f.ShirtID)
	_, _ = db.Exec(ctx, `DELETE FROM order_items WHERE product_id IN ($1, $2, $3)`, f.HoodieID, f.PantsID, f.ShirtID)
	_, _ = db.Exec(ctx, `DELETE FROM orders WHERE id IN ($1, $2, $3)`, f.OrderAID, f.OrderBID, f.OrderCID)
	_, _ = db.Exec(ctx, `DELETE FROM products WHERE id IN ($1, $2, $3)`, f.HoodieID, f.PantsID, f.ShirtID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id IN ($1, $2)`, f.CategoryAID, f.CategoryBID)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, f.UserAID, f.UserBID)
}
