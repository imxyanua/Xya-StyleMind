package product

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
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
	filter, err := parseListFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := h.service.List(c.Request.Context(), filter, filter.Limit, filter.Offset)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidID):
			response.Error(c, http.StatusBadRequest, "invalid category_id")
		case errors.Is(err, errs.ErrInvalidSort):
			response.Error(c, http.StatusBadRequest, "invalid sort")
		case errors.Is(err, errs.ErrValidationFailed):
			response.Error(c, http.StatusBadRequest, "validation failed")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to fetch products")
		}
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(filter.Page, filter.Limit, total))
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

func parseListFilter(c *gin.Context) (ListFilter, error) {
	page, err := parsePositiveIntQuery(c, "page", 1, 0)
	if err != nil {
		return ListFilter{}, err
	}
	limit, err := parsePositiveIntQuery(c, "limit", 20, 100)
	if err != nil {
		return ListFilter{}, err
	}

	minPrice, err := parseOptionalNonNegativeFloat(c, "min_price")
	if err != nil {
		return ListFilter{}, err
	}
	maxPrice, err := parseOptionalNonNegativeFloat(c, "max_price")
	if err != nil {
		return ListFilter{}, err
	}
	minRating, err := parseOptionalNonNegativeFloat(c, "min_rating")
	if err != nil {
		return ListFilter{}, err
	}
	inStock, err := parseOptionalBool(c, "in_stock")
	if err != nil {
		return ListFilter{}, err
	}

	sort := strings.TrimSpace(c.DefaultQuery("sort", SortNewest))
	return ListFilter{
		Query:      strings.TrimSpace(c.Query("q")),
		CategoryID: strings.TrimSpace(c.Query("category_id")),
		MinPrice:   minPrice,
		MaxPrice:   maxPrice,
		Style:      strings.TrimSpace(c.Query("style")),
		Color:      strings.TrimSpace(c.Query("color")),
		MinRating:  minRating,
		InStock:    inStock,
		Sort:       sort,
		Page:       page,
		Limit:      limit,
		Offset:     (page - 1) * limit,
	}, nil
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int, max int) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("invalid " + key)
	}
	if max > 0 && value > max {
		return 0, errors.New("invalid " + key)
	}
	return value, nil
}

func parseOptionalNonNegativeFloat(c *gin.Context, key string) (*float64, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		return nil, errors.New("invalid " + key)
	}
	return &value, nil
}

func parseOptionalBool(c *gin.Context, key string) (*bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, errors.New("invalid " + key)
	}
	return &value, nil
}
