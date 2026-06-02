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
