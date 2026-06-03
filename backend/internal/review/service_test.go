package review

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type fakeReviewRepository struct {
	existingProducts map[string]bool
	purchased        map[string]bool
	reviews          map[string]*Review
	addErr           error
	lastUserID       string
	lastProductID    string
}

func newFakeReviewRepository() *fakeReviewRepository {
	return &fakeReviewRepository{
		existingProducts: make(map[string]bool),
		purchased:        make(map[string]bool),
		reviews:          make(map[string]*Review),
	}
}

func (r *fakeReviewRepository) ProductExists(_ context.Context, productID string) (bool, error) {
	return r.existingProducts[productID], nil
}

func (r *fakeReviewRepository) HasPurchasedProduct(_ context.Context, userID, productID, orderID string) (bool, error) {
	return r.purchased[userID+"|"+productID+"|"+orderID], nil
}

func (r *fakeReviewRepository) Create(_ context.Context, userID, productID string, req CreateReviewRequest) (*Review, error) {
	r.lastUserID = userID
	r.lastProductID = productID
	if r.addErr != nil {
		return nil, r.addErr
	}
	for _, existing := range r.reviews {
		if existing.UserID == userID && existing.ProductID == productID {
			return nil, errs.ErrReviewAlreadyExists
		}
	}
	id := uuid.NewString()
	item := &Review{ID: id, UserID: userID, ProductID: productID, OrderID: req.OrderID, Rating: req.Rating, Comment: req.Comment}
	r.reviews[id] = item
	return item, nil
}

func (r *fakeReviewRepository) ListByProduct(_ context.Context, productID string, _, _ int) ([]Review, int64, error) {
	out := make([]Review, 0)
	for _, item := range r.reviews {
		if item.ProductID == productID {
			out = append(out, *item)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeReviewRepository) SummaryByProduct(_ context.Context, productID string) (*RatingSummary, error) {
	summary := &RatingSummary{RatingBreakdown: map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}}
	total := 0
	for _, item := range r.reviews {
		if item.ProductID == productID {
			summary.ReviewCount++
			summary.RatingBreakdown[item.Rating]++
			total += item.Rating
		}
	}
	if summary.ReviewCount > 0 {
		summary.AverageRating = float64(total) / float64(summary.ReviewCount)
	}
	return summary, nil
}

func (r *fakeReviewRepository) GetByID(_ context.Context, reviewID string) (*Review, error) {
	item, ok := r.reviews[reviewID]
	if !ok {
		return nil, errs.ErrReviewNotFound
	}
	return item, nil
}

func (r *fakeReviewRepository) Update(_ context.Context, reviewID string, req UpdateReviewRequest) (*Review, error) {
	item, ok := r.reviews[reviewID]
	if !ok {
		return nil, errs.ErrReviewNotFound
	}
	item.Rating = req.Rating
	item.Comment = req.Comment
	return item, nil
}

func (r *fakeReviewRepository) Delete(_ context.Context, reviewID string) error {
	if _, ok := r.reviews[reviewID]; !ok {
		return errs.ErrReviewNotFound
	}
	delete(r.reviews, reviewID)
	return nil
}

func TestServiceCreateReviewSuccess(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	orderID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.purchased["user-1|"+productID+"|"+orderID] = true
	service := NewService(repo)

	comment := "Great fit"
	item, err := service.Create(context.Background(), "user-1", productID, CreateReviewRequest{OrderID: orderID, Rating: 5, Comment: &comment})
	if err != nil {
		t.Fatalf("Create error = %v", err)
	}
	if item.UserID != "user-1" || item.ProductID != productID || item.Rating != 5 {
		t.Fatalf("review = %+v, want created review", item)
	}
}

func TestServiceCreateReviewRequiresPurchasedProduct(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	service := NewService(repo)

	_, err := service.Create(context.Background(), "user-1", productID, CreateReviewRequest{OrderID: uuid.NewString(), Rating: 5})
	if !errors.Is(err, errs.ErrProductNotPurchased) {
		t.Fatalf("err = %v, want ErrProductNotPurchased", err)
	}
}

func TestServiceCreateReviewProductNotFound(t *testing.T) {
	service := NewService(newFakeReviewRepository())

	_, err := service.Create(context.Background(), "user-1", uuid.NewString(), CreateReviewRequest{OrderID: uuid.NewString(), Rating: 5})
	if !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
}

func TestServiceCreateReviewValidation(t *testing.T) {
	service := NewService(newFakeReviewRepository())
	longComment := makeString(1001)

	tests := []CreateReviewRequest{
		{OrderID: uuid.NewString(), Rating: 0},
		{OrderID: uuid.NewString(), Rating: 6},
		{OrderID: uuid.NewString(), Rating: 5, Comment: &longComment},
	}
	for _, req := range tests {
		_, err := service.Create(context.Background(), "user-1", uuid.NewString(), req)
		if !errors.Is(err, errs.ErrValidationFailed) {
			t.Fatalf("err = %v, want ErrValidationFailed", err)
		}
	}
}

func TestServiceCreateReviewDuplicate(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	orderID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.purchased["user-1|"+productID+"|"+orderID] = true
	service := NewService(repo)

	if _, err := service.Create(context.Background(), "user-1", productID, CreateReviewRequest{OrderID: orderID, Rating: 5}); err != nil {
		t.Fatalf("Create first error = %v", err)
	}
	_, err := service.Create(context.Background(), "user-1", productID, CreateReviewRequest{OrderID: orderID, Rating: 4})
	if !errors.Is(err, errs.ErrReviewAlreadyExists) {
		t.Fatalf("err = %v, want ErrReviewAlreadyExists", err)
	}
}

func TestServiceUpdateAndDeleteOwnership(t *testing.T) {
	repo := newFakeReviewRepository()
	id := uuid.NewString()
	repo.reviews[id] = &Review{ID: id, UserID: "user-1", ProductID: uuid.NewString(), Rating: 3}
	service := NewService(repo)

	comment := "Updated"
	updated, err := service.Update(context.Background(), "user-1", id, UpdateReviewRequest{Rating: 4, Comment: &comment})
	if err != nil {
		t.Fatalf("Update own error = %v", err)
	}
	if updated.Rating != 4 {
		t.Fatalf("rating = %d, want 4", updated.Rating)
	}

	if _, err := service.Update(context.Background(), "user-2", id, UpdateReviewRequest{Rating: 5}); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("Update other err = %v, want ErrForbidden", err)
	}
	if err := service.Delete(context.Background(), "user-2", id); !errors.Is(err, errs.ErrForbidden) {
		t.Fatalf("Delete other err = %v, want ErrForbidden", err)
	}
	if err := service.Delete(context.Background(), "user-1", id); err != nil {
		t.Fatalf("Delete own error = %v", err)
	}
}

func TestServiceSummary(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.reviews[uuid.NewString()] = &Review{ID: uuid.NewString(), ProductID: productID, Rating: 5}
	repo.reviews[uuid.NewString()] = &Review{ID: uuid.NewString(), ProductID: productID, Rating: 3}
	service := NewService(repo)

	summary, err := service.SummaryByProduct(context.Background(), productID)
	if err != nil {
		t.Fatalf("SummaryByProduct error = %v", err)
	}
	if summary.ReviewCount != 2 || summary.AverageRating != 4 || summary.RatingBreakdown[5] != 1 || summary.RatingBreakdown[3] != 1 {
		t.Fatalf("summary = %+v, want average 4 count 2 breakdown", summary)
	}
}

func makeString(length int) string {
	out := make([]byte, length)
	for i := range out {
		out[i] = 'a'
	}
	return string(out)
}
