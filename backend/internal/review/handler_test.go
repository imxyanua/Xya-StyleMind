package review

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stylemind/internal/auth"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func newReviewTestRouter(repo *fakeReviewRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Set("user_role", "user")
		c.Next()
	}
	RegisterRoutes(api, authMiddleware, NewService(repo))
	return router
}

func newProtectedReviewTestRouter(repo *fakeReviewRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterRoutes(api, middleware.JWTAuth(reviewTestTokenConfig()), NewService(repo))
	return router
}

func TestHandlerCreateReviewSuccess(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	orderID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.purchased["user-1|"+productID+"|"+orderID] = true
	router := newReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+productID+"/reviews", bytes.NewBufferString(`{"order_id":"`+orderID+`","rating":5,"comment":"Great"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusCreated, w.Body.String())
	}
}

func TestHandlerCreateReviewUnauthorized(t *testing.T) {
	router := newProtectedReviewTestRouter(newFakeReviewRepository())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+uuid.NewString()+"/reviews", bytes.NewBufferString(`{"order_id":"`+uuid.NewString()+`","rating":5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReviewErrorResponse(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestHandlerCreateReviewForbiddenWhenNotPurchased(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	router := newReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+productID+"/reviews", bytes.NewBufferString(`{"order_id":"`+uuid.NewString()+`","rating":5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReviewErrorResponse(t, w, http.StatusForbidden, "product not purchased")
}

func TestHandlerCreateReviewProductNotFound(t *testing.T) {
	router := newReviewTestRouter(newFakeReviewRepository())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+uuid.NewString()+"/reviews", bytes.NewBufferString(`{"order_id":"`+uuid.NewString()+`","rating":5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReviewErrorResponse(t, w, http.StatusNotFound, "product not found")
}

func TestHandlerCreateReviewValidation(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	router := newReviewTestRouter(repo)

	for _, body := range []string{
		`{"order_id":"` + uuid.NewString() + `","rating":0}`,
		`{"order_id":"` + uuid.NewString() + `","rating":6}`,
		`{"order_id":"` + uuid.NewString() + `","rating":5,"comment":"` + makeString(1001) + `"}`,
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+productID+"/reviews", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		assertReviewErrorResponse(t, w, http.StatusBadRequest, "validation failed")
	}
}

func TestHandlerCreateReviewDuplicate(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	orderID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.purchased["user-1|"+productID+"|"+orderID] = true
	repo.reviews[uuid.NewString()] = &Review{ID: uuid.NewString(), UserID: "user-1", ProductID: productID, Rating: 4}
	router := newReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/products/"+productID+"/reviews", bytes.NewBufferString(`{"order_id":"`+orderID+`","rating":5}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertReviewErrorResponse(t, w, http.StatusConflict, "review already exists")
}

func TestHandlerListReviewsAndSummary(t *testing.T) {
	repo := newFakeReviewRepository()
	productID := uuid.NewString()
	repo.existingProducts[productID] = true
	repo.reviews[uuid.NewString()] = &Review{ID: uuid.NewString(), UserID: "user-1", ProductID: productID, Rating: 5}
	repo.reviews[uuid.NewString()] = &Review{ID: uuid.NewString(), UserID: "user-2", ProductID: productID, Rating: 3}
	router := newReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/products/"+productID+"/reviews?page=1&limit=20", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/products/"+productID+"/rating-summary", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("summary status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
	var body struct {
		Data RatingSummary `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body.Data.ReviewCount != 2 || body.Data.AverageRating != 4 || body.Data.RatingBreakdown[5] != 1 || body.Data.RatingBreakdown[3] != 1 {
		t.Fatalf("summary = %+v, want average 4 count 2", body.Data)
	}
}

func TestHandlerUpdateAndDeleteReviewOwnership(t *testing.T) {
	repo := newFakeReviewRepository()
	ownID := uuid.NewString()
	otherID := uuid.NewString()
	repo.reviews[ownID] = &Review{ID: ownID, UserID: "user-1", ProductID: uuid.NewString(), Rating: 3}
	repo.reviews[otherID] = &Review{ID: otherID, UserID: "user-2", ProductID: uuid.NewString(), Rating: 5}
	router := newReviewTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/reviews/"+ownID, bytes.NewBufferString(`{"rating":4,"comment":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update own status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/reviews/"+otherID, bytes.NewBufferString(`{"rating":4}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assertReviewErrorResponse(t, w, http.StatusForbidden, "forbidden")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/reviews/"+otherID, nil)
	router.ServeHTTP(w, req)
	assertReviewErrorResponse(t, w, http.StatusForbidden, "forbidden")

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/reviews/"+ownID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("delete own status = %d, want %d, body=%s", w.Code, http.StatusOK, w.Body.String())
	}
}

func assertReviewErrorResponse(t *testing.T, w *httptest.ResponseRecorder, status int, message string) {
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

func reviewTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{
		Secret:   "test-secret",
		Issuer:   "stylemind-api",
		Audience: "stylemind-web",
	}
}
