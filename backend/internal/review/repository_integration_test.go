//go:build integration

package review

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

func TestReviewRepositoryIntegration(t *testing.T) {
	ctx, db := openReviewIntegrationDB(t)
	repo := NewRepository(db)
	fixture := newReviewFixture()
	defer cleanupReviewRows(ctx, t, db, fixture)
	insertReviewFixture(ctx, t, db, fixture)

	exists, err := repo.ProductExists(ctx, fixture.ProductID)
	if err != nil || !exists {
		t.Fatalf("ProductExists = %v/%v, want true/nil", exists, err)
	}
	purchased, err := repo.HasPurchasedProduct(ctx, fixture.UserID, fixture.ProductID, fixture.OrderID)
	if err != nil || !purchased {
		t.Fatalf("HasPurchasedProduct = %v/%v, want true/nil", purchased, err)
	}
	notPurchased, err := repo.HasPurchasedProduct(ctx, fixture.OtherUserID, fixture.ProductID, fixture.OrderID)
	if err != nil {
		t.Fatalf("HasPurchasedProduct other user error = %v", err)
	}
	if notPurchased {
		t.Fatal("other user should not be verified as purchased")
	}

	comment := "Excellent"
	item, err := repo.Create(ctx, fixture.UserID, fixture.ProductID, CreateReviewRequest{OrderID: fixture.OrderID, Rating: 5, Comment: &comment})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if item.Rating != 5 || item.Comment == nil || *item.Comment != "Excellent" {
		t.Fatalf("review = %+v, want rating/comment", item)
	}

	if _, err := repo.Create(ctx, fixture.UserID, fixture.ProductID, CreateReviewRequest{OrderID: fixture.OrderID, Rating: 4}); !errors.Is(err, errs.ErrReviewAlreadyExists) {
		t.Fatalf("duplicate create err = %v, want ErrReviewAlreadyExists", err)
	}

	if _, err := repo.Create(ctx, fixture.OtherUserID, fixture.ProductID, CreateReviewRequest{OrderID: fixture.OtherOrderID, Rating: 3}); err != nil {
		t.Fatalf("Create other user review error = %v", err)
	}

	reviews, total, err := repo.ListByProduct(ctx, fixture.ProductID, 20, 0)
	if err != nil {
		t.Fatalf("ListByProduct error = %v", err)
	}
	if total != 2 || len(reviews) != 2 {
		t.Fatalf("reviews/total = %+v/%d, want 2", reviews, total)
	}

	summary, err := repo.SummaryByProduct(ctx, fixture.ProductID)
	if err != nil {
		t.Fatalf("SummaryByProduct error = %v", err)
	}
	if summary.ReviewCount != 2 || summary.AverageRating != 4 || summary.RatingBreakdown[5] != 1 || summary.RatingBreakdown[3] != 1 {
		t.Fatalf("summary = %+v, want avg 4 count 2 breakdown", summary)
	}

	updatedComment := "Updated"
	updated, err := repo.Update(ctx, item.ID, UpdateReviewRequest{Rating: 4, Comment: &updatedComment})
	if err != nil {
		t.Fatalf("Update error = %v", err)
	}
	if updated.Rating != 4 || updated.Comment == nil || *updated.Comment != "Updated" {
		t.Fatalf("updated = %+v, want rating/comment update", updated)
	}
	if err := repo.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := repo.GetByID(ctx, item.ID); !errors.Is(err, errs.ErrReviewNotFound) {
		t.Fatalf("GetByID deleted err = %v, want ErrReviewNotFound", err)
	}
}

type reviewFixture struct {
	UserID       string
	OtherUserID  string
	CategoryID   string
	ProductID    string
	OrderID      string
	OtherOrderID string
}

func newReviewFixture() reviewFixture {
	return reviewFixture{
		UserID:       uuid.NewString(),
		OtherUserID:  uuid.NewString(),
		CategoryID:   uuid.NewString(),
		ProductID:    uuid.NewString(),
		OrderID:      uuid.NewString(),
		OtherOrderID: uuid.NewString(),
	}
}

func openReviewIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
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

func insertReviewFixture(ctx context.Context, t *testing.T, db *pgxpool.Pool, f reviewFixture) {
	t.Helper()
	if _, err := db.Exec(ctx, `
		INSERT INTO users (id, email, full_name, password_hash, role)
		VALUES ($1, $2, 'Review User', 'hash', 'user'),
		       ($3, $4, 'Other Review User', 'hash', 'user')
	`, f.UserID, "review-"+f.UserID+"@example.com", f.OtherUserID, "review-"+f.OtherUserID+"@example.com"); err != nil {
		t.Fatalf("insert users error = %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO categories (id, name, slug) VALUES ($1, $2, $3)`, f.CategoryID, "Review Category "+f.CategoryID[:8], "review-"+f.CategoryID[:8]); err != nil {
		t.Fatalf("insert category error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO products (id, name, description, price, stock, category_id, style, color, image_url)
		VALUES ($1, 'Review Product', 'Review product description', 300000, 5, $2, 'formal', 'black', 'https://example.com/review.jpg')
	`, f.ProductID, f.CategoryID); err != nil {
		t.Fatalf("insert product error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO orders (id, user_id, status, total_amount)
		VALUES ($1, $2, 'completed', 300000),
		       ($3, $4, 'paid', 300000)
	`, f.OrderID, f.UserID, f.OtherOrderID, f.OtherUserID); err != nil {
		t.Fatalf("insert orders error = %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO order_items (id, order_id, product_id, quantity, unit_price, subtotal)
		VALUES ($1, $2, $3, 1, 300000, 300000),
		       ($4, $5, $3, 1, 300000, 300000)
	`, uuid.NewString(), f.OrderID, f.ProductID, uuid.NewString(), f.OtherOrderID); err != nil {
		t.Fatalf("insert order items error = %v", err)
	}
}

func cleanupReviewRows(ctx context.Context, t *testing.T, db *pgxpool.Pool, f reviewFixture) {
	t.Helper()
	_, _ = db.Exec(ctx, `DELETE FROM product_reviews WHERE product_id = $1`, f.ProductID)
	_, _ = db.Exec(ctx, `DELETE FROM order_items WHERE product_id = $1`, f.ProductID)
	_, _ = db.Exec(ctx, `DELETE FROM orders WHERE id IN ($1, $2)`, f.OrderID, f.OtherOrderID)
	_, _ = db.Exec(ctx, `DELETE FROM products WHERE id = $1`, f.ProductID)
	_, _ = db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, f.CategoryID)
	_, _ = db.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, f.UserID, f.OtherUserID)
}
