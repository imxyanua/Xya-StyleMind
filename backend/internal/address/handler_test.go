package address

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"stylemind/internal/auth"
	"stylemind/internal/errs"
	"stylemind/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type fakeAddressRepo struct {
	items          []Address
	createErr      error
	updateErr      error
	deleteErr      error
	setDefaultErr  error
	lastUserID     string
	lastAddressID  string
	lastInput      AddressRequest
	deleted        bool
	defaultCleared bool
}

func (r *fakeAddressRepo) List(_ context.Context, userID string) ([]Address, error) {
	r.lastUserID = userID
	return r.items, nil
}

func (r *fakeAddressRepo) Create(_ context.Context, userID string, input AddressRequest) (*Address, error) {
	r.lastUserID = userID
	r.lastInput = input
	if r.createErr != nil {
		return nil, r.createErr
	}
	return sampleAddress(userID, uuid.NewString(), input.IsDefault), nil
}

func (r *fakeAddressRepo) Update(_ context.Context, userID, addressID string, input AddressRequest) (*Address, error) {
	r.lastUserID = userID
	r.lastAddressID = addressID
	r.lastInput = input
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	return sampleAddress(userID, addressID, input.IsDefault), nil
}

func (r *fakeAddressRepo) Delete(_ context.Context, userID, addressID string) error {
	r.lastUserID = userID
	r.lastAddressID = addressID
	r.deleted = true
	return r.deleteErr
}

func (r *fakeAddressRepo) SetDefault(_ context.Context, userID, addressID string) (*Address, error) {
	r.lastUserID = userID
	r.lastAddressID = addressID
	r.defaultCleared = true
	if r.setDefaultErr != nil {
		return nil, r.setDefaultErr
	}
	return sampleAddress(userID, addressID, true), nil
}

func sampleAddress(userID, id string, isDefault bool) *Address {
	return &Address{
		ID:            id,
		UserID:        userID,
		RecipientName: "E2E Buyer",
		Phone:         "0901234567",
		AddressLine:   "88 Style Street",
		City:          "Ho Chi Minh City",
		District:      "District 1",
		Note:          "Call first",
		IsDefault:     isDefault,
	}
}

func validAddressBody(isDefault bool) *bytes.Buffer {
	body := map[string]any{
		"recipient_name": "E2E Buyer",
		"phone":          "0901234567",
		"address_line":   "88 Style Street",
		"city":           "Ho Chi Minh City",
		"district":       "District 1",
		"note":           "Call first",
		"is_default":     isDefault,
	}
	payload, _ := json.Marshal(body)
	return bytes.NewBuffer(payload)
}

func newAddressTestRouter(repo *fakeAddressRepo) *gin.Engine {
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

func newProtectedAddressTestRouter(repo *fakeAddressRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterRoutes(api, middleware.JWTAuth(addressTestTokenConfig()), NewService(repo))
	return router
}

func addressTestTokenConfig() auth.TokenConfig {
	return auth.TokenConfig{Secret: "test-secret", Issuer: "stylemind-api", Audience: "stylemind-web"}
}

func TestProtectedAddressRoutesRequireToken(t *testing.T) {
	router := newProtectedAddressTestRouter(&fakeAddressRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me/addresses", nil)
	router.ServeHTTP(w, req)

	assertAddressError(t, w, http.StatusUnauthorized, "unauthorized")
}

func TestHandlerCreateAddressSuccess(t *testing.T) {
	repo := &fakeAddressRepo{}
	router := newAddressTestRouter(repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/addresses", validAddressBody(true))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", w.Code, w.Body.String())
	}
	if repo.lastUserID != "user-1" || !repo.lastInput.IsDefault {
		t.Fatalf("repo scope/input = %s/%+v", repo.lastUserID, repo.lastInput)
	}
}

func TestHandlerCreateAddressValidation(t *testing.T) {
	router := newAddressTestRouter(&fakeAddressRepo{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/addresses", bytes.NewBufferString(`{"recipient_name":"A"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertAddressError(t, w, http.StatusBadRequest, "validation failed")
}

func TestHandlerUpdateAddressOtherUserNotFound(t *testing.T) {
	repo := &fakeAddressRepo{updateErr: errs.ErrAddressNotFound}
	router := newAddressTestRouter(repo)
	addressID := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/addresses/"+addressID, validAddressBody(false))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assertAddressError(t, w, http.StatusNotFound, "address not found")
	if repo.lastUserID != "user-1" || repo.lastAddressID != addressID {
		t.Fatalf("repo scope = %s/%s", repo.lastUserID, repo.lastAddressID)
	}
}

func TestHandlerSetDefaultClearsPreviousDefault(t *testing.T) {
	repo := &fakeAddressRepo{}
	router := newAddressTestRouter(repo)
	addressID := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/me/addresses/"+addressID+"/default", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !repo.defaultCleared || repo.lastUserID != "user-1" || repo.lastAddressID != addressID {
		t.Fatalf("repo default call = cleared:%v user:%s address:%s", repo.defaultCleared, repo.lastUserID, repo.lastAddressID)
	}
}

func TestHandlerDeleteAddressSuccess(t *testing.T) {
	repo := &fakeAddressRepo{}
	router := newAddressTestRouter(repo)
	addressID := uuid.NewString()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/me/addresses/"+addressID, nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if !repo.deleted || repo.lastUserID != "user-1" || repo.lastAddressID != addressID {
		t.Fatalf("repo delete call = deleted:%v user:%s address:%s", repo.deleted, repo.lastUserID, repo.lastAddressID)
	}
}

func assertAddressError(t *testing.T, w *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if w.Code != wantStatus {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, wantStatus, w.Body.String())
	}
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json unmarshal error = %v", err)
	}
	if body.Success || body.Message != wantMessage {
		t.Fatalf("body = %+v, want success false message %q", body, wantMessage)
	}
}
