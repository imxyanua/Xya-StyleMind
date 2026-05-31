package category

import (
	"net/http"

	"stylemind/pkg/pagination"
	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, service *Service) {
	h := &Handler{service: service}
	api.GET("/categories", h.List)
	admin.POST("/categories", h.Create)
}

func (h *Handler) List(c *gin.Context) {
	page := pagination.Parse(c)
	items, total, err := h.service.List(c.Request.Context(), page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch categories")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "failed to create category")
		return
	}
	response.Success(c, http.StatusCreated, "category created", item)
}
