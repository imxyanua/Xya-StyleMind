package order

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"stylemind/internal/audit"
	"stylemind/internal/errs"
	"stylemind/internal/notification"
	"stylemind/pkg/logger"
	"time"

	"stylemind/pkg/pagination"
	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       *Service
	audit         audit.Recorder
	notifications notifier
}

type notifier interface {
	Create(ctx context.Context, input notification.CreateInput) (*notification.Notification, error)
}

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service, extras ...any) {
	var recorder audit.Recorder
	var notifications notifier
	for _, extra := range extras {
		if candidate, ok := extra.(audit.Recorder); ok {
			recorder = candidate
		}
		if candidate, ok := extra.(notifier); ok {
			notifications = candidate
		}
	}
	h := &Handler{service: service, audit: recorder, notifications: notifications}

	orders := api.Group("/orders")
	orders.Use(authMiddleware)
	orders.POST("", h.Checkout)
	orders.GET("", h.ListMine)
	orders.GET("/:id", h.GetMine)

	admin.GET("/orders", h.ListAdmin)
	admin.GET("/orders/:id", h.GetAdmin)
	admin.PUT("/orders/:id/status", h.UpdateStatus)
	admin.PATCH("/orders/:id/status", h.UpdateStatus)
	admin.PATCH("/orders/:id/payment-status", h.UpdatePaymentStatus)
}

func (h *Handler) Checkout(c *gin.Context) {
	var req CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	userID := c.GetString("user_id")
	order, err := h.service.Checkout(c.Request.Context(), userID, CheckoutDetails{
		RecipientName:  req.RecipientName,
		Phone:          req.Phone,
		AddressLine:    req.AddressLine,
		City:           req.City,
		District:       req.District,
		Note:           req.Note,
		ShippingMethod: req.ShippingMethod,
		PaymentMethod:  req.PaymentMethod,
		CouponCode:     req.CouponCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrValidationFailed):
			response.Error(c, http.StatusBadRequest, "validation failed")
		case errors.Is(err, errs.ErrCartEmpty):
			response.Error(c, http.StatusBadRequest, "cart is empty")
		case errors.Is(err, errs.ErrInsufficientStock):
			response.Error(c, http.StatusBadRequest, "insufficient stock")
		case errors.Is(err, errs.ErrCouponNotFound):
			response.Error(c, http.StatusNotFound, "coupon not found")
		case errors.Is(err, errs.ErrCouponInactive):
			response.Error(c, http.StatusBadRequest, "coupon is inactive")
		case errors.Is(err, errs.ErrCouponExpired):
			response.Error(c, http.StatusBadRequest, "coupon is expired")
		case errors.Is(err, errs.ErrCouponUsageLimitReached):
			response.Error(c, http.StatusBadRequest, "coupon usage limit reached")
		case errors.Is(err, errs.ErrCouponMinOrderNotMet):
			response.Error(c, http.StatusBadRequest, "minimum order amount not met")
		case errors.Is(err, errs.ErrInvalidCoupon):
			response.Error(c, http.StatusBadRequest, "invalid coupon")
		default:
			response.Error(c, http.StatusInternalServerError, "checkout failed")
		}
		return
	}
	h.notify(c, order.UserID, notification.TypeOrderCreated, "Order placed successfully", "Your order has been created and is waiting for processing.", gin.H{
		"order_id":        order.ID,
		"status":          order.Status,
		"payment_status":  order.PaymentStatus,
		"total_amount":    order.TotalAmount,
		"discount_amount": order.DiscountAmount,
		"coupon_code":     order.CouponCode,
	})
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

func (h *Handler) ListAdmin(c *gin.Context) {
	page := pagination.Parse(c)
	filter, err := parseAdminOrderFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	orders, total, err := h.service.ListOrders(c.Request.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidOrderStatus) {
			response.Error(c, http.StatusBadRequest, "invalid status")
			return
		}
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid user_id")
			return
		}
		if errors.Is(err, errs.ErrInvalidSort) {
			response.Error(c, http.StatusBadRequest, "invalid sort")
			return
		}
		if errors.Is(err, errs.ErrValidationFailed) {
			response.Error(c, http.StatusBadRequest, "validation failed")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch orders")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", orders, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) GetAdmin(c *gin.Context) {
	order, err := h.service.GetOrder(c.Request.Context(), c.Param("id"))
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
		logger.Audit(c, "admin.order_status.update", logger.AuditResultFailed, map[string]any{
			"order_id": c.Param("id"),
			"reason":   "invalid_payload",
		})
		h.recordAudit(c, "admin.order_status.update", c.Param("id"), audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		logger.Audit(c, "admin.order_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":   c.Param("id"),
			"new_status": req.Status,
			"reason":     "validation_error",
		})
		h.recordAudit(c, "admin.order_status.update", c.Param("id"), audit.ResultFailed, gin.H{"new_status": req.Status, "reason": "validation_error"})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	currentOrder, err := h.service.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		logger.Audit(c, "admin.order_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":   c.Param("id"),
			"new_status": req.Status,
			"reason":     safeOrderAuditReason(err),
		})
		h.recordAudit(c, "admin.order_status.update", c.Param("id"), audit.ResultFailed, gin.H{"new_status": req.Status, "reason": safeOrderAuditReason(err)})
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid order id")
			return
		}
		if errors.Is(err, errs.ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update order status")
		return
	}

	order, err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		logger.Audit(c, "admin.order_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":   c.Param("id"),
			"old_status": currentOrder.Status,
			"new_status": req.Status,
			"reason":     safeOrderAuditReason(err),
		})
		h.recordAudit(c, "admin.order_status.update", c.Param("id"), audit.ResultFailed, gin.H{"old_status": currentOrder.Status, "new_status": req.Status, "reason": safeOrderAuditReason(err)})
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
	logger.Audit(c, "admin.order_status.update", logger.AuditResultSuccess, map[string]any{
		"order_id":   order.ID,
		"old_status": currentOrder.Status,
		"new_status": order.Status,
	})
	h.recordAudit(c, "admin.order_status.update", order.ID, audit.ResultSuccess, gin.H{"old_status": currentOrder.Status, "new_status": order.Status})
	h.notify(c, order.UserID, notification.TypeOrderStatusUpdated, "Order status updated", "Your order status changed from "+currentOrder.Status+" to "+order.Status+".", gin.H{
		"order_id":   order.ID,
		"old_status": currentOrder.Status,
		"new_status": order.Status,
	})
	response.Success(c, http.StatusOK, "order status updated", order)
}

func (h *Handler) UpdatePaymentStatus(c *gin.Context) {
	var req UpdatePaymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Audit(c, "admin.order_payment_status.update", logger.AuditResultFailed, map[string]any{
			"order_id": c.Param("id"),
			"reason":   "invalid_payload",
		})
		h.recordAudit(c, "admin.order_payment_status.update", c.Param("id"), audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		logger.Audit(c, "admin.order_payment_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":           c.Param("id"),
			"new_payment_status": req.PaymentStatus,
			"reason":             "validation_error",
		})
		h.recordAudit(c, "admin.order_payment_status.update", c.Param("id"), audit.ResultFailed, gin.H{"new_payment_status": req.PaymentStatus, "reason": "validation_error"})
		response.Error(c, http.StatusBadRequest, "invalid payment status")
		return
	}

	currentOrder, err := h.service.GetOrder(c.Request.Context(), c.Param("id"))
	if err != nil {
		logger.Audit(c, "admin.order_payment_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":           c.Param("id"),
			"new_payment_status": req.PaymentStatus,
			"reason":             safePaymentAuditReason(err),
		})
		h.recordAudit(c, "admin.order_payment_status.update", c.Param("id"), audit.ResultFailed, gin.H{"new_payment_status": req.PaymentStatus, "reason": safePaymentAuditReason(err)})
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid order id")
			return
		}
		if errors.Is(err, errs.ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update payment status")
		return
	}

	order, err := h.service.UpdatePaymentStatus(c.Request.Context(), c.Param("id"), req.PaymentStatus)
	if err != nil {
		logger.Audit(c, "admin.order_payment_status.update", logger.AuditResultFailed, map[string]any{
			"order_id":           c.Param("id"),
			"old_payment_status": currentOrder.PaymentStatus,
			"new_payment_status": req.PaymentStatus,
			"reason":             safePaymentAuditReason(err),
		})
		h.recordAudit(c, "admin.order_payment_status.update", c.Param("id"), audit.ResultFailed, gin.H{"old_payment_status": currentOrder.PaymentStatus, "new_payment_status": req.PaymentStatus, "reason": safePaymentAuditReason(err)})
		if errors.Is(err, errs.ErrInvalidID) {
			response.Error(c, http.StatusBadRequest, "invalid order id")
			return
		}
		if errors.Is(err, errs.ErrOrderNotFound) {
			response.Error(c, http.StatusNotFound, "order not found")
			return
		}
		if errors.Is(err, errs.ErrInvalidPaymentStatus) {
			response.Error(c, http.StatusBadRequest, "invalid payment status")
			return
		}
		if errors.Is(err, errs.ErrInvalidPaymentStatusTransition) {
			response.Error(c, http.StatusBadRequest, "invalid payment status transition")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to update payment status")
		return
	}
	logger.Audit(c, "admin.order_payment_status.update", logger.AuditResultSuccess, map[string]any{
		"order_id":           order.ID,
		"old_payment_status": currentOrder.PaymentStatus,
		"new_payment_status": order.PaymentStatus,
	})
	h.recordAudit(c, "admin.order_payment_status.update", order.ID, audit.ResultSuccess, gin.H{"old_payment_status": currentOrder.PaymentStatus, "new_payment_status": order.PaymentStatus})
	h.notify(c, order.UserID, notification.TypePaymentStatusUpdated, "Payment status updated", "Your payment status changed from "+currentOrder.PaymentStatus+" to "+order.PaymentStatus+".", gin.H{
		"order_id":           order.ID,
		"old_payment_status": currentOrder.PaymentStatus,
		"new_payment_status": order.PaymentStatus,
	})
	response.Success(c, http.StatusOK, "payment status updated", order)
}

func (h *Handler) notify(c *gin.Context, userID, notificationType, title, message string, metadata map[string]any) {
	if h.notifications == nil || userID == "" {
		return
	}
	_, _ = h.notifications.Create(c.Request.Context(), notification.CreateInput{
		UserID:   userID,
		Type:     notificationType,
		Title:    title,
		Message:  message,
		Metadata: metadata,
	})
}

func (h *Handler) recordAudit(c *gin.Context, action, resourceID, result string, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	h.audit.RecordAdmin(c, action, "order", resourceID, result, metadata)
}

func safeOrderAuditReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		return "validation_error"
	case errors.Is(err, errs.ErrOrderNotFound):
		return "not_found"
	case errors.Is(err, errs.ErrInvalidOrderStatus):
		return "validation_error"
	case errors.Is(err, errs.ErrInvalidOrderStatusTransition):
		return "invalid_status_transition"
	default:
		return "internal_error"
	}
}

func safePaymentAuditReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		return "validation_error"
	case errors.Is(err, errs.ErrOrderNotFound):
		return "not_found"
	case errors.Is(err, errs.ErrInvalidPaymentStatus):
		return "validation_error"
	case errors.Is(err, errs.ErrInvalidPaymentStatusTransition):
		return "invalid_status_transition"
	default:
		return "internal_error"
	}
}

func parseAdminOrderFilter(c *gin.Context) (AdminOrderFilter, error) {
	from, err := parseOptionalTime(c.Query("from"), false)
	if err != nil {
		return AdminOrderFilter{}, errors.New("invalid from")
	}
	to, err := parseOptionalTime(c.Query("to"), true)
	if err != nil {
		return AdminOrderFilter{}, errors.New("invalid to")
	}
	return AdminOrderFilter{
		Query:  strings.TrimSpace(c.Query("q")),
		Status: strings.TrimSpace(c.Query("status")),
		UserID: strings.TrimSpace(c.Query("user_id")),
		From:   from,
		To:     to,
		Sort:   strings.TrimSpace(c.DefaultQuery("sort", AdminOrderSortNewest)),
	}, nil
}

func parseOptionalTime(raw string, endOfDay bool) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
