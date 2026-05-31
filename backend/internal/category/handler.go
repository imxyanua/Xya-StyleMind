package category

import (
	"net/http"

	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, repo *Repository) {
	h := &Handler{repo: repo}
	api.GET("/categories", h.List)
	admin.POST("/categories", h.Create)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.repo.List(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch categories", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", items)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	item, err := h.repo.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create category", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "category created", item)
}
