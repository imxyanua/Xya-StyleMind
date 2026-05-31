package product

import (
	"errors"
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

	api.GET("/products", h.List)
	api.GET("/products/:id", h.GetDetail)

	admin.POST("/products", h.Create)
	admin.PUT("/products/:id", h.Update)
	admin.DELETE("/products/:id", h.Delete)
}

func (h *Handler) List(c *gin.Context) {
	filter := ListFilter{
		Style:      c.Query("style"),
		Color:      c.Query("color"),
		CategoryID: c.Query("category_id"),
	}
	items, err := h.repo.List(c.Request.Context(), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch products", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", items)
}

func (h *Handler) GetDetail(c *gin.Context) {
	item, err := h.repo.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch product", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", item)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateProductRequest
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
		response.Error(c, http.StatusBadRequest, "failed to create product", err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "product created", item)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	item, err := h.repo.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found", err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to update product", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "product updated", item)
}

func (h *Handler) Delete(c *gin.Context) {
	err := h.repo.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, "product not found", err.Error())
			return
		}
		response.Error(c, http.StatusBadRequest, "failed to delete product", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "product deleted", gin.H{"id": c.Param("id")})
}
