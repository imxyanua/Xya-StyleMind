package pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/products", nil)

	params := Parse(c)
	if params.Page != 1 || params.Limit != 20 || params.Offset != 0 {
		t.Fatalf("Parse defaults = %+v, want page=1 limit=20 offset=0", params)
	}
}

func TestParseBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/products?page=-1&limit=500", nil)

	params := Parse(c)
	if params.Page != 1 {
		t.Fatalf("Page = %d, want 1", params.Page)
	}
	if params.Limit != 100 {
		t.Fatalf("Limit = %d, want 100", params.Limit)
	}
	if params.Offset != 0 {
		t.Fatalf("Offset = %d, want 0", params.Offset)
	}
}

func TestBuildMeta(t *testing.T) {
	meta := BuildMeta(2, 20, 45)
	if meta.Page != 2 || meta.Limit != 20 || meta.Total != 45 || meta.TotalPage != 3 {
		t.Fatalf("BuildMeta = %+v, want page=2 limit=20 total=45 total_page=3", meta)
	}
}
