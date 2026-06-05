package review

import (
	"net/http"
	"testing"

	"stylemind/internal/product"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesDoesNotConflictWithProductRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api/v1")
	admin := api.Group("/admin")
	auth := func(c *gin.Context) { c.Next() }

	product.RegisterRoutes(api, admin, (*product.Service)(nil))
	RegisterRoutes(api, auth, (*Service)(nil))

	routes := map[string]bool{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}

	expected := []string{
		http.MethodGet + " /api/v1/products/:id",
		http.MethodGet + " /api/v1/products/:id/reviews",
		http.MethodGet + " /api/v1/products/:id/rating-summary",
		http.MethodPost + " /api/v1/products/:id/reviews",
	}
	for _, route := range expected {
		if !routes[route] {
			t.Fatalf("expected route %s to be registered", route)
		}
	}
}
