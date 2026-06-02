package order

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
		case errors.Is(err, errs.ErrCartEmpty):
			response.Error(c, http.StatusBadRequest, "cart is empty")
		case errors.Is(err, errs.ErrInsufficientStock):
			response.Error(c, http.StatusBadRequest, "insufficient stock")
		default:
			response.Error(c, http.StatusInternalServerError, "checkout failed")
		}
		return
	}
	response.Success(c, http.StatusCreated, "order created", order)
}

func (h *Handler) ListMine(c *gin.Context) {
	userID := c.GetString("user_id")
	page := pagination.Parse(c)
	orders, total, err := h.service.ListMyOrders(c.Request.Context(), userID, page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", orders, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) GetMine(c *gin.Context) {
	userID := c.GetString("user_id")
	order, err := h.service.GetMyOrder(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid order id")
			return
		}
		if errors.Is(err, errs.ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch order")
		return
	}
	response.Success(c, http.StatusOK, "ok", order)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	order, err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid order id")
			return
		}
		if errors.Is(err, errs.ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, errs.ErrInvalidOrderStatus) {
			response.Error(c, http.StatusBadRequest, "invalid order status")
			return
		}
		if errors.Is(err, errs.ErrInvalidOrderStatusTransition) {
			response.Error(c, http.StatusBadRequest, "invalid order status transition")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update order status")
		return
	}
	response.Success(c, http.StatusOK, "order status updated", order)
}
