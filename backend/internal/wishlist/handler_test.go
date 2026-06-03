package wishlist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stylemind/internal/auth"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newWishlistTestRouter(repo *fakeWishlistRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-a")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, authMiddleware, NewService(repo))
	return router
}

func newProtectedWishlistTestRouter(repo *fakeWishlistRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterRoutes(api, middleware.JWTAuth(wishlistTestTokenConfig()), NewService(repo))
	return router
}

func TestHandlerAddWishlistProductSuccessAndDuplicate(t *testing.T) {
	repo := newFakeWishlistRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	router := newWishlistTestRouter(repo)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlist/products/"+productID, nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d, body=%s", i, w.Code, http.StatusOK, w.Body.String())
		}
	}
	if len(repo.itemsByUser["user-a"]) != 1 {
		t.Fatalf("wishlist item count = %d, want 1", len(repo.itemsByUser["user-a"]))
	}
}

func TestHandlerAddWishlistProductNotFound(t *testing.T) {
	router := newWishlistTestRouter(newFakeWishlistRepository())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/wishlist/products/"+uuid.NewString(), nil)
	router.ServeHTTP(w, req)

	assertWishlistErrorResponse(t, w, http.StatusNotFound, "product not found")
}

func TestHandlerRemoveWishlistProductSuccessAndMissing(t *testing.T) {
	repo := newFakeWishlistRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.itemsByUser["user-a"] = map[string]WishlistItem{
		productID: {ID: uuid.NewString(), UserID: "user-a", ProductID: productID},
	}
	router := newWishlistTestRouter(repo)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/api/v1/wishlist/products/"+productID, nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d, body=%s", i, w.Code, http.StatusOK, w.Body.String())
		}
	}
	if len(repo.itemsByUser["user-a"]) != 0 {
		t.Fatalf("wishlist item count = %d, want 0", len(repo.itemsByUser["user-a"]))
	}
}

func TestHandlerRemoveWishlistProductNotFound(t *testing.T) {
	router := newWishlistTestRouter(newFakeWishlistRepository())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/wishlist/products/"+uuid.NewString(), nil)
	router.ServeHTTP(w, req)

	assertWishlistErrorResponse(t, w, http.StatusNotFound, "product not found")
}

func TestHandlerWishlistUnauthorized(t *testing.T) {
	router := newProtectedWishlistTestRouter(newFakeWishlistRepository())

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/wishlist"},
		{method: http.MethodPost, path: "/api/v1/wishlist/products/" + uuid.NewString()},
		{method: http.MethodDelete, path: "/api/v1/wishlist/products/" + uuid.NewString()},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			router.ServeHTTP(w, req)
			assertWishlistErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
		})
	}
}

func TestHandlerListWishlistEmpty(t *testing.T) {
	router := newWishlistTestRouter(newFakeWishlistRepository())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlist", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	body := wishlistResponseBody(t, w)
	if len(body.Data) != 0 || body.Meta.Total != 0 {
		t.Fatalf("data/meta = %+v/%+v, want empty list", body.Data, body.Meta)
	}
}

func TestHandlerListWishlistUsesUserScopeAndPagination(t *testing.T) {
	repo := newFakeWishlistRepository()
	productA := uuid.NewString()
	productB := uuid.NewString()
	otherProduct := uuid.NewString()
	repo.itemsByUser["user-a"] = map[string]WishlistItem{
		productA: {ID: uuid.NewString(), UserID: "user-a", ProductID: productA, Product: ProductSnapshot{ID: productA, Name: "A"}},
		productB: {ID: uuid.NewString(), UserID: "user-a", ProductID: productB, Product: ProductSnapshot{ID: productB, Name: "B"}},
	}
	repo.itemsByUser["user-b"] = map[string]WishlistItem{
		otherProduct: {ID: uuid.NewString(), UserID: "user-b", ProductID: otherProduct},
	}
	router := newWishlistTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/wishlist?page=2&limit=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	body := wishlistResponseBody(t, w)
	if len(body.Data) != 2 || body.Meta.Total != 2 {
		t.Fatalf("data/meta = %+v/%+v, want two user-a items", body.Data, body.Meta)
	}
	if repo.lastUserID != "user-a" || repo.listLimit != 10 || repo.listOffset != 10 {
		t.Fatalf("repo scope = user:%s limit:%d offset:%d, want user-a/10/10", repo.lastUserID, repo.listLimit, repo.listOffset)
	}
	for _, item := range body.Data {
		if item.UserID != "user-a" {
			t.Fatalf("item.UserID = %s, want user-a", item.UserID)
		}
	}
}

func wishlistResponseBody(t *testing.T, w *httptest.ResponseRecorder) struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    []WishlistItem `json:"data"`
	Meta    struct {
		Page      int   `json:"page"`
		Limit     int   `json:"limit"`
		Total     int64 `json:"total"`
		TotalPage int64 `json:"total_page"`
	} `json:"meta"`
} {
	t.Helper()
	var body struct {
		Success bool           `json:"success"`
		Message string         `json:"message"`
		Data    []WishlistItem `json:"data"`
		Meta    struct {
			Page      int   `json:"page"`
			Limit     int   `json:"limit"`
			Total     int64 `json:"total"`
			TotalPage int64 `json:"total_page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	return body
}

func assertWishlistErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, status, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %v, want false", body["success"])
	}
	if body["message"] != message {
		t.Fatalf("message = %v, want %s", body["message"], message)
	}
}

func wishlistTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		Secret:   "test-secret",
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
	}
}
