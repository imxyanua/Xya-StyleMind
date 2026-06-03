package wishlist

import (
	"context"
	"errors"
	"testing"

	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type fakeWishlistRepository struct {
	existingProducts map[string]bool
	itemsByUser      map[string]map[string]WishlistItem
	addErr           error
	removeErr        error
	listErr          error
	lastUserID       string
	lastProductID    string
	listLimit        int
	listOffset       int
}

func newFakeWishlistRepository() *fakeWishlistRepository {
	return &fakeWishlistRepository{
		existingProducts: make(map[string]bool),
		itemsByUser:      make(map[string]map[string]WishlistItem),
	}
}

func (r *fakeWishlistRepository) ProductExists(_ context.Context, productID string) (bool, error) {
	return r.existingProducts[productID], nil
}

func (r *fakeWishlistRepository) AddProduct(_ context.Context, userID, productID string) error {
	r.lastUserID = userID
	r.lastProductID = productID
	if r.addErr != nil {
		return r.addErr
	}
	if r.itemsByUser[userID] == nil {
		r.itemsByUser[userID] = make(map[string]WishlistItem)
	}
	r.itemsByUser[userID][productID] = WishlistItem{
		ID:        uuid.NewString(),
		UserID:    userID,
		ProductID: productID,
		Product:   ProductSnapshot{ID: productID, Name: "Wishlist Product"},
	}
	return nil
}

func (r *fakeWishlistRepository) RemoveProduct(_ context.Context, userID, productID string) error {
	r.lastUserID = userID
	r.lastProductID = productID
	if r.removeErr != nil {
		return r.removeErr
	}
	delete(r.itemsByUser[userID], productID)
	return nil
}

func (r *fakeWishlistRepository) ListByUser(_ context.Context, userID string, limit, offset int) ([]WishlistItem, int64, error) {
	r.lastUserID = userID
	r.listLimit = limit
	r.listOffset = offset
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	items := make([]WishlistItem, 0, len(r.itemsByUser[userID]))
	for _, item := range r.itemsByUser[userID] {
		items = append(items, item)
	}
	return items, int64(len(items)), nil
}

func TestServiceAddProductSuccessAndDuplicateIdempotent(t *testing.T) {
	repo := newFakeWishlistRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	service := NewService(repo)

	if err := service.AddProduct(context.Background(), "user-1", productID); err != nil {
		t.Fatalf("AddProduct first error = %v", err)
	}
	if err := service.AddProduct(context.Background(), "user-1", productID); err != nil {
		t.Fatalf("AddProduct duplicate error = %v", err)
	}
	if len(repo.itemsByUser["user-1"]) != 1 {
		t.Fatalf("wishlist item count = %d, want 1", len(repo.itemsByUser["user-1"]))
	}
}

func TestServiceAddProductInvalidID(t *testing.T) {
	service := NewService(newFakeWishlistRepository())

	err := service.AddProduct(context.Background(), "user-1", "bad-id")
	if !errors.Is(err, errs.ErrInvalidID) {
		t.Fatalf("err = %v, want ErrInvalidID", err)
	}
}

func TestServiceAddProductNotFound(t *testing.T) {
	service := NewService(newFakeWishlistRepository())

	err := service.AddProduct(context.Background(), "user-1", uuid.NewString())
	if !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
}

func TestServiceRemoveProductSuccessAndMissingIdempotent(t *testing.T) {
	repo := newFakeWishlistRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	service := NewService(repo)

	if err := service.AddProduct(context.Background(), "user-1", productID); err != nil {
		t.Fatalf("AddProduct error = %v", err)
	}
	if err := service.RemoveProduct(context.Background(), "user-1", productID); err != nil {
		t.Fatalf("RemoveProduct existing error = %v", err)
	}
	if err := service.RemoveProduct(context.Background(), "user-1", productID); err != nil {
		t.Fatalf("RemoveProduct missing error = %v", err)
	}
	if len(repo.itemsByUser["user-1"]) != 0 {
		t.Fatalf("wishlist item count = %d, want 0", len(repo.itemsByUser["user-1"]))
	}
}

func TestServiceRemoveProductNotFound(t *testing.T) {
	service := NewService(newFakeWishlistRepository())

	err := service.RemoveProduct(context.Background(), "user-1", uuid.NewString())
	if !errors.Is(err, errs.ErrProductNotFound) {
		t.Fatalf("err = %v, want ErrProductNotFound", err)
	}
}

func TestServiceListUsesAuthenticatedUserScope(t *testing.T) {
	repo := newFakeWishlistRepository()
	userProductID := uuid.NewString()
	otherProductID := uuid.NewString()
	repo.itemsByUser["user-a"] = map[string]WishlistItem{
		userProductID: {ID: uuid.NewString(), UserID: "user-a", ProductID: userProductID},
	}
	repo.itemsByUser["user-b"] = map[string]WishlistItem{
		otherProductID: {ID: uuid.NewString(), UserID: "user-b", ProductID: otherProductID},
	}
	service := NewService(repo)

	items, total, err := service.List(context.Background(), "user-a", 20, 0)
	if err != nil {
		t.Fatalf("List error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != "user-a" {
		t.Fatalf("items/total = %+v/%d, want only user-a items", items, total)
	}
	if repo.lastUserID != "user-a" || repo.listLimit != 20 || repo.listOffset != 0 {
		t.Fatalf("repo scope = user:%s limit:%d offset:%d", repo.lastUserID, repo.listLimit, repo.listOffset)
	}
}
