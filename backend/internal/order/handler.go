package order

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

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}

	orders := api.Group("/orders")
	orders.Use(authMiddleware)
	orders.POST("", h.Checkout)
	orders.GET("", h.ListMine)
	orders.GET("/:id", h.GetMine)

	admin.PUT("/orders/:id/status", h.UpdateStatus)
}

func (h *Handler) Checkout(c *gin.Context) {
	userID := c.GetString("user_id")
	order, err := h.service.Checkout(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrCartEmpty):
			response.Error(c, http.StatusBadRequest, "checkout failed", err.Error())
		case errors.Is(err, ErrOutOfStock):
			response.Error(c, http.StatusBadRequest, "checkout failed", err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, "checkout failed", err.Error())
		}
		return
	}
	response.Success(c, http.StatusCreated, "order created", order)
}

func (h *Handler) ListMine(c *gin.Context) {
	userID := c.GetString("user_id")
	orders, err := h.service.ListMyOrders(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch orders", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", orders)
}

func (h *Handler) GetMine(c *gin.Context) {
	userID := c.GetString("user_id")
	order, err := h.service.GetMyOrder(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch order", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "ok", order)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload", err.Error())
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed", err.Error())
		return
	}

	order, err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found", err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update order status", err.Error())
		return
	}
	response.Success(c, http.StatusOK, "order status updated", order)
}
