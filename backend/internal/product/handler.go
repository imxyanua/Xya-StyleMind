package product

import (
	"errors"
	"net/http"
	"stylemind/internal/errs"

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

	api.GET("/products", h.List)
	api.GET("/products/:id", h.GetDetail)

	admin.POST("/products", h.Create)
	admin.PUT("/products/:id", h.Update)
	admin.DELETE("/products/:id", h.Delete)
}

func (h *Handler) List(c *gin.Context) {
	page := pagination.Parse(c)
	filter := ListFilter{
		Style:      c.Query("style"),
		Color:      c.Query("color"),
		CategoryID: c.Query("category_id"),
	}
	items, total, err := h.service.List(c.Request.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid category_id")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch products")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) GetDetail(c *gin.Context) {
	item, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid product id")
			return
		}
		if errors.Is(err, errs.ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch product")
		return
	}
	response.Success(c, http.StatusOK, "ok", item)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateProductRequest
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
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid category_id")
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to create product")
		return
	}
	response.Success(c, http.StatusCreated, "product created", item)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	item, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid product or category id")
			return
		}
		if errors.Is(err, errs.ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to update product")
		return
	}
	response.Success(c, http.StatusOK, "product updated", item)
}

func (h *Handler) Delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid product id")
			return
		}
		if errors.Is(err, errs.ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found")
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to delete product")
		return
	}
	response.Success(c, http.StatusOK, "product deleted", gin.H{"id": c.Param("id")})
}
