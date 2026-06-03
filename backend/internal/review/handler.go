package review

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

func RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}

	api.GET("/products/:product_id/reviews", h.ListByProduct)
	api.GET("/products/:product_id/rating-summary", h.SummaryByProduct)
	api.POST("/products/:product_id/reviews", authMiddleware, h.Create)
	api.PATCH("/reviews/:id", authMiddleware, h.Update)
	api.DELETE("/reviews/:id", authMiddleware, h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	review, err := h.service.Create(c.Request.Context(), c.GetString("user_id"), c.Param("product_id"), req)
	if err != nil {
		h.writeError(c, err, "failed to create review")
		return
	}
	response.Success(c, http.StatusCreated, "review created", review)
}

func (h *Handler) ListByProduct(c *gin.Context) {
	page := pagination.Parse(c)
	reviews, total, err := h.service.ListByProduct(c.Request.Context(), c.Param("product_id"), page.Limit, page.Offset)
	if err != nil {
		h.writeError(c, err, "failed to fetch reviews")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", reviews, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) SummaryByProduct(c *gin.Context) {
	summary, err := h.service.SummaryByProduct(c.Request.Context(), c.Param("product_id"))
	if err != nil {
		h.writeError(c, err, "failed to fetch rating summary")
		return
	}
	response.Success(c, http.StatusOK, "ok", summary)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	review, err := h.service.Update(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req)
	if err != nil {
		h.writeError(c, err, "failed to update review")
		return
	}
	response.Success(c, http.StatusOK, "review updated", review)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), c.GetString("user_id"), c.Param("id")); err != nil {
		h.writeError(c, err, "failed to delete review")
		return
	}
	response.Success(c, http.StatusOK, "review deleted", gin.H{"id": c.Param("id")})
}

func (h *Handler) writeError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid id")
	case errors.Is(err, errs.ErrValidationFailed):
		response.Error(c, http.StatusBadRequest, "validation failed")
	case errors.Is(err, errs.ErrProductNotFound):
		response.Error(c, http.StatusNotFound, "product not found")
	case errors.Is(err, errs.ErrProductNotPurchased):
		response.Error(c, http.StatusForbidden, "product not purchased")
	case errors.Is(err, errs.ErrReviewAlreadyExists):
		response.Error(c, http.StatusConflict, "review already exists")
	case errors.Is(err, errs.ErrReviewNotFound):
		response.Error(c, http.StatusNotFound, "review not found")
	case errors.Is(err, errs.ErrForbidden):
		response.Error(c, http.StatusForbidden, "forbidden")
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}
