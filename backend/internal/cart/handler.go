package cart

import (
	"errors"
	"net/http"

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
		response.Error(c, http.StatusInternalServerError, "failed to fetch cart", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", result)
}

func (h *Handler) AddItem(c *gin.Context) {
	userID := c.GetString("user_id")
	var req AddCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	result, err := h.service.AddItem(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidQuantity):
			response.Error(c, http.StatusBadRequest, "failed to add cart item", err.Error())
		case errors.Is(err, ErrProductNotFound):
			response.Error(c, http.StatusNotFound, "failed to add cart item", err.Error())
		case errors.Is(err, ErrOutOfStock):
			response.Error(c, http.StatusBadRequest, "failed to add cart item", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to add cart item", err.Error())
		}
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}

func (h *Handler) UpdateItem(c *gin.Context) {
	userID := c.GetString("user_id")
	var req UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	result, err := h.service.UpdateItem(c.Request.Context(), userID, c.Param("id"), req.Quantity)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidQuantity):
			response.Error(c, http.StatusBadRequest, "failed to update cart item", err.Error())
		case errors.Is(err, ErrCartItemNotFound):
			response.Error(c, http.StatusNotFound, "failed to update cart item", err.Error())
		case errors.Is(err, ErrOutOfStock):
			response.Error(c, http.StatusBadRequest, "failed to update cart item", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update cart item", err.Error())
		}
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}

func (h *Handler) DeleteItem(c *gin.Context) {
	userID := c.GetString("user_id")
	result, err := h.service.DeleteItem(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrCartItemNotFound) {
			response.Error(c, http.StatusNotFound, "failed to delete cart item", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to delete cart item", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "cart updated", result)
}
