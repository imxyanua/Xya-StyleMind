package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Params struct {
	Page   int
	Limit  int
	Offset int
}

type Meta struct {
	Page      int   `json:"page"`
	Limit     int   `json:"limit"`
	Total     int64 `json:"total"`
	TotalPage int64 `json:"total_page"`
}

func Parse(c *gin.Context) Params {
	page := parseInt(c.DefaultQuery("page", "1"), 1)
	limit := parseInt(c.DefaultQuery("limit", "20"), 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	return Params{
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
}

func BuildMeta(page, limit int, total int64) Meta {
	totalPage := total / int64(limit)
	if total%int64(limit) != 0 {
		totalPage++
	}
	return Meta{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: totalPage,
	}
}

func parseInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}
