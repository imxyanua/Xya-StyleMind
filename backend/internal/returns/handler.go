package returns

import (
	"errors"
	"net/http"
	"strings"

	"stylemind/internal/audit"
	"stylemind/internal/errs"
	"stylemind/pkg/logger"
	"stylemind/pkg/pagination"
	"stylemind/pkg/response"
	"stylemind/pkg/validator"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
	audit   audit.Recorder
}

func RegisterRoutes(api *gin.RouterGroup, admin *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service, recorders ...audit.Recorder) {
	var recorder audit.Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}
	h := &Handler{service: service, audit: recorder}

	orders := api.Group("/orders")
	orders.Use(authMiddleware)
	orders.POST("/:id/return-requests", h.Create)

	me := api.Group("/me")
	me.Use(authMiddleware)
	me.GET("/return-requests", h.ListMine)

	admin.GET("/return-requests", h.ListAdmin)
	admin.GET("/return-requests/:id", h.GetAdmin)
	admin.PATCH("/return-requests/:id/status", h.UpdateStatus)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	item, err := h.service.Create(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req)
	if err != nil {
		handleCreateError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "return request created", item)
}

func (h *Handler) ListMine(c *gin.Context) {
	page := pagination.Parse(c)
	items, total, err := h.service.ListMine(c.Request.Context(), c.GetString("user_id"), page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch return requests")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) ListAdmin(c *gin.Context) {
	page := pagination.Parse(c)
	filter, err := parseAdminFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := h.service.ListAdmin(c.Request.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidReturnRequestStatus):
			response.Error(c, http.StatusBadRequest, "invalid status")
		case errors.Is(err, errs.ErrInvalidID):
			response.Error(c, http.StatusBadRequest, "invalid id")
		case errors.Is(err, errs.ErrInvalidSort):
			response.Error(c, http.StatusBadRequest, "invalid sort")
		case errors.Is(err, errs.ErrValidationFailed):
			response.Error(c, http.StatusBadRequest, "validation failed")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to fetch return requests")
		}
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) GetAdmin(c *gin.Context) {
	item, err := h.service.GetAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleAdminReadError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "ok", item)
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.auditFailed(c, c.Param("id"), "admin.return_request.update", gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		h.auditFailed(c, c.Param("id"), actionForStatus(req.Status), gin.H{"new_status": req.Status, "reason": "validation_error"})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	current, err := h.service.GetAdmin(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.auditFailed(c, c.Param("id"), actionForStatus(req.Status), gin.H{"new_status": req.Status, "reason": safeReason(err)})
		handleAdminReadError(c, err)
		return
	}

	item, err := h.service.UpdateStatus(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		metadata := gin.H{
			"order_id":           current.OrderID,
			"old_status":         current.Status,
			"new_status":         req.Status,
			"old_payment_status": current.Order.PaymentStatus,
			"reason":             safeReason(err),
		}
		h.auditFailed(c, current.ID, actionForStatus(req.Status), metadata)
		handleUpdateError(c, err)
		return
	}

	metadata := gin.H{
		"order_id":           item.OrderID,
		"old_status":         current.Status,
		"new_status":         item.Status,
		"old_payment_status": current.Order.PaymentStatus,
		"new_payment_status": item.Order.PaymentStatus,
	}
	h.auditSuccess(c, item.ID, actionForStatus(item.Status), metadata)
	response.Success(c, http.StatusOK, "return request updated", item)
}

func handleCreateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid order id")
	case errors.Is(err, errs.ErrValidationFailed):
		response.Error(c, http.StatusBadRequest, "validation failed")
	case errors.Is(err, errs.ErrOrderNotFound):
		response.Error(c, http.StatusNotFound, "order not found")
	case errors.Is(err, errs.ErrReturnRequestNotAllowed):
		response.Error(c, http.StatusBadRequest, "order is not eligible for return")
	case errors.Is(err, errs.ErrReturnRequestAlreadyExists):
		response.Error(c, http.StatusConflict, "active return request already exists")
	default:
		response.Error(c, http.StatusInternalServerError, "failed to create return request")
	}
}

func handleAdminReadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid return request id")
	case errors.Is(err, errs.ErrReturnRequestNotFound):
		response.Error(c, http.StatusNotFound, "return request not found")
	default:
		response.Error(c, http.StatusInternalServerError, "failed to fetch return request")
	}
}

func handleUpdateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid return request id")
	case errors.Is(err, errs.ErrReturnRequestNotFound):
		response.Error(c, http.StatusNotFound, "return request not found")
	case errors.Is(err, errs.ErrInvalidReturnRequestStatus):
		response.Error(c, http.StatusBadRequest, "invalid return request status")
	case errors.Is(err, errs.ErrInvalidPaymentStatusTransition):
		response.Error(c, http.StatusBadRequest, "invalid payment status transition")
	default:
		response.Error(c, http.StatusInternalServerError, "failed to update return request")
	}
}

func (h *Handler) auditSuccess(c *gin.Context, id, action string, metadata map[string]any) {
	logger.Audit(c, action, logger.AuditResultSuccess, metadata)
	h.recordAudit(c, action, id, audit.ResultSuccess, metadata)
}

func (h *Handler) auditFailed(c *gin.Context, id, action string, metadata map[string]any) {
	logger.Audit(c, action, logger.AuditResultFailed, metadata)
	h.recordAudit(c, action, id, audit.ResultFailed, metadata)
}

func (h *Handler) recordAudit(c *gin.Context, action, resourceID, result string, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	h.audit.RecordAdmin(c, action, "return_request", resourceID, result, metadata)
}

func actionForStatus(status string) string {
	switch strings.TrimSpace(status) {
	case StatusApproved:
		return "admin.return_request.approve"
	case StatusRejected:
		return "admin.return_request.reject"
	case StatusCancelled:
		return "admin.return_request.cancel"
	default:
		return "admin.return_request.update"
	}
}

func safeReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrInvalidID), errors.Is(err, errs.ErrInvalidReturnRequestStatus):
		return "validation_error"
	case errors.Is(err, errs.ErrReturnRequestNotFound), errors.Is(err, errs.ErrOrderNotFound):
		return "not_found"
	case errors.Is(err, errs.ErrInvalidPaymentStatusTransition):
		return "invalid_status_transition"
	default:
		return "internal_error"
	}
}

func parseAdminFilter(c *gin.Context) (AdminFilter, error) {
	from, err := ParseOptionalTime(c.Query("from"), false)
	if err != nil {
		return AdminFilter{}, errors.New("invalid from")
	}
	to, err := ParseOptionalTime(c.Query("to"), true)
	if err != nil {
		return AdminFilter{}, errors.New("invalid to")
	}
	return AdminFilter{
		Status:  strings.TrimSpace(c.Query("status")),
		UserID:  strings.TrimSpace(c.Query("user_id")),
		OrderID: strings.TrimSpace(c.Query("order_id")),
		From:    from,
		To:      to,
		Sort:    strings.TrimSpace(c.DefaultQuery("sort", SortNewest)),
	}, nil
}
