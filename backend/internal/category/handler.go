package category

import (
	"context"
	"net/http"

	"stylemind/internal/audit"
	"stylemind/pkg/pagination"
	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service categoryService
	audit   audit.Recorder
}

type categoryService interface {
	List(ctx context.Context, limit, offset int) ([]Category, int64, error)
	Create(ctx context.Context, req CreateCategoryRequest) (*Category, error)
}

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, service categoryService, recorders ...audit.Recorder) {
	var recorder audit.Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}
	h := &Handler{service: service, audit: recorder}
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
		h.recordAudit(c, "admin.category.create", "", audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		h.recordAudit(c, "admin.category.create", "", audit.ResultFailed, gin.H{"reason": "validation_error", "category_name": req.Name, "slug": req.Slug})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.recordAudit(c, "admin.category.create", "", audit.ResultFailed, gin.H{"reason": "internal_error", "category_name": req.Name, "slug": req.Slug})
		response.Error(c, http.StatusBadRequest, "failed to create category")
		return
	}
	h.recordAudit(c, "admin.category.create", item.ID, audit.ResultSuccess, gin.H{"category_name": item.Name, "slug": item.Slug})
	response.Success(c, http.StatusCreated, "category created", item)
}

func (h *Handler) recordAudit(c *gin.Context, action, resourceID, result string, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	h.audit.RecordAdmin(c, action, "category", resourceID, result, metadata)
}
