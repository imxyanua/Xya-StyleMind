package product

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandlerListInvalidQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	RegisterRoutes(api, admin, NewService(nil))

	tests := []struct {
		name    string
		query   string
		message string
	}{
		{name: "invalid page", query: "page=0", message: "invalid page"},
		{name: "invalid limit", query: "limit=101", message: "invalid limit"},
		{name: "invalid min price", query: "min_price=-1", message: "invalid min_price"},
		{name: "invalid max price", query: "max_price=bad", message: "invalid max_price"},
		{name: "invalid min rating", query: "min_rating=bad", message: "invalid min_rating"},
		{name: "invalid bool", query: "in_stock=maybe", message: "invalid in_stock"},
		{name: "invalid sort", query: "sort=bad", message: "invalid sort"},
		{name: "invalid price range", query: "min_price=20&max_price=10", message: "validation failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/products?"+tt.query, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusBadRequest, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.message) {
				t.Fatalf("body = %s, want message %q", w.Body.String(), tt.message)
			}
		})
	}
}

func TestParseListFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodGet, "/products?q=hoodie&category_id=cat&min_price=10&max_price=20&style=streetwear&color=black&min_rating=4&in_stock=true&sort=price_asc&page=2&limit=10", nil)
	c.Request = req

	filter, err := parseListFilter(c)
	if err != nil {
		t.Fatalf("parseListFilter error = %v", err)
	}
	if filter.Query != "hoodie" || filter.CategoryID != "cat" || filter.Style != "streetwear" || filter.Color != "black" || filter.Sort != SortPriceAsc {
		t.Fatalf("filter string fields = %+v", filter)
	}
	if filter.MinPrice == nil || *filter.MinPrice != 10 || filter.MaxPrice == nil || *filter.MaxPrice != 20 {
		t.Fatalf("price filters = %+v/%+v", filter.MinPrice, filter.MaxPrice)
	}
	if filter.MinRating == nil || *filter.MinRating != 4 {
		t.Fatalf("min rating = %+v", filter.MinRating)
	}
	if filter.InStock == nil || *filter.InStock != true {
		t.Fatalf("in stock = %+v", filter.InStock)
	}
	if filter.Page != 2 || filter.Limit != 10 || filter.Offset != 10 {
		t.Fatalf("pagination = page:%d limit:%d offset:%d", filter.Page, filter.Limit, filter.Offset)
	}
}
