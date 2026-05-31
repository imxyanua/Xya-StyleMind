package cart

import (
	"errors"
	"net/http"
	"stylemind/internal/errs"

	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}

	cart := api.Group("/cart")
	cart.Use(authMiddleware)
	cart.GET("", h.GetCart)
	cart.POST("/items", h.AddItem)
	cart.PUT("/items/:id", h.UpdateItem)
	cart.DELETE("/items/:id", h.DeleteItem)
}

func (h *Handler) GetCart(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.service.GetCart(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch cart")
		return
	}
	response.Success(c, http.StatusOK, "ok", result)
}

func (h *Handler) AddItem(c *gin.Context) {
	userID := c.GetString("user_id")
	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	result, err := h.service.AddItem(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidQuantity):
			response.Error(c, http.StatusBadRequest, "quantity must be greater than 0")
		case errors.Is(err, errs.ErrProductNotFound):
			response.Error(c, http.StatusNotFound, "product not found")
		case errors.Is(err, errs.ErrInsufficientStock):
			response.Error(c, http.StatusBadRequest, "insufficient stock")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to add cart item")
		}
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	userID := c.GetString("user_id")
	var req UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	result, err := h.service.UpdateItem(c.Request.Context(), userID, c.Param("id"), req.Quantity)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidQuantity):
			response.Error(c, http.StatusBadRequest, "quantity must be greater than 0")
		case errors.Is(err, errs.ErrCartItemNotFound):
			response.Error(c, http.StatusNotFound, "cart item not found")
		case errors.Is(err, errs.ErrInsufficientStock):
			response.Error(c, http.StatusBadRequest, "insufficient stock")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update cart item")
		}
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.service.DeleteItem(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, errs.ErrCartItemNotFound) {
			response.Error(c, http.StatusNotFound, "cart item not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete cart item")
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}
