package coupon

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"stylemind/internal/audit"
	"stylemind/internal/errs"
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

	cart := api.Group("/cart")
	cart.Use(authMiddleware)
	cart.POST("/apply-coupon", h.Apply)

	admin.GET("/coupons", h.List)
	admin.GET("/coupons/:id", h.Get)
	admin.POST("/coupons", h.Create)
	admin.PATCH("/coupons/:id", h.Update)
	admin.DELETE("/coupons/:id", h.Delete)
}

func (h *Handler) Apply(c *gin.Context) {
	var req ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	result, err := h.service.ApplyToCart(c.Request.Context(), c.GetString("user_id"), req.Code)
	if err != nil {
		writeCouponApplyError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "coupon applied", result)
}

func (h *Handler) List(c *gin.Context) {
	page := pagination.Parse(c)
	filter, err := parseListFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	items, total, err := h.service.List(c.Request.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		if errors.Is(err, errs.ErrInvalidCoupon) {
			response.Error(c, http.StatusBadRequest, "invalid coupon filter")
			return
		}
		if errors.Is(err, errs.ErrInvalidSort) {
			response.Error(c, http.StatusBadRequest, "invalid sort")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch coupons")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) Get(c *gin.Context) {
	item, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeCouponAdminError(c, err, "failed to fetch coupon")
		return
	}
	response.Success(c, http.StatusOK, "ok", item)
}

func (h *Handler) Create(c *gin.Context) {
	var req MutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordAudit(c, "admin.coupon.create", "", audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		h.recordAudit(c, "admin.coupon.create", "", audit.ResultFailed, gin.H{"reason": "validation_error", "coupon_code": NormalizeCode(req.Code)})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		h.recordAudit(c, "admin.coupon.create", "", audit.ResultFailed, gin.H{"reason": safeAuditReason(err), "coupon_code": NormalizeCode(req.Code)})
		writeCouponAdminError(c, err, "failed to create coupon")
		return
	}
	h.recordAudit(c, "admin.coupon.create", item.ID, audit.ResultSuccess, gin.H{"coupon_code": item.Code, "type": item.Type, "value": item.Value})
	response.Success(c, http.StatusCreated, "coupon created", item)
}

func (h *Handler) Update(c *gin.Context) {
	var req MutationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordAudit(c, "admin.coupon.update", c.Param("id"), audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		h.recordAudit(c, "admin.coupon.update", c.Param("id"), audit.ResultFailed, gin.H{"reason": "validation_error", "coupon_code": NormalizeCode(req.Code)})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}
	item, err := h.service.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.recordAudit(c, "admin.coupon.update", c.Param("id"), audit.ResultFailed, gin.H{"reason": safeAuditReason(err), "coupon_code": NormalizeCode(req.Code)})
		writeCouponAdminError(c, err, "failed to update coupon")
		return
	}
	h.recordAudit(c, "admin.coupon.update", item.ID, audit.ResultSuccess, gin.H{"coupon_code": item.Code, "type": item.Type, "value": item.Value, "is_active": item.IsActive})
	response.Success(c, http.StatusOK, "coupon updated", item)
}

func (h *Handler) Delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.recordAudit(c, "admin.coupon.delete", c.Param("id"), audit.ResultFailed, gin.H{"reason": safeAuditReason(err)})
		writeCouponAdminError(c, err, "failed to delete coupon")
		return
	}
	h.recordAudit(c, "admin.coupon.delete", c.Param("id"), audit.ResultSuccess, nil)
	response.Success(c, http.StatusOK, "coupon deleted", gin.H{"id": c.Param("id")})
}

func parseListFilter(c *gin.Context) (ListFilter, error) {
	isActive, err := parseOptionalBool(c.Query("is_active"))
	if err != nil {
		return ListFilter{}, errors.New("invalid is_active")
	}
	return ListFilter{
		Query:    strings.TrimSpace(c.Query("q")),
		Type:     strings.TrimSpace(c.Query("type")),
		IsActive: isActive,
		Sort:     strings.TrimSpace(c.DefaultQuery("sort", SortNewest)),
	}, nil
}

func parseOptionalBool(raw string) (*bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func writeCouponApplyError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrCartEmpty):
		response.Error(c, http.StatusBadRequest, "cart is empty")
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
		response.Error(c, http.StatusInternalServerError, "failed to apply coupon")
	}
}

func writeCouponAdminError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid coupon id")
	case errors.Is(err, errs.ErrValidationFailed):
		response.Error(c, http.StatusBadRequest, "validation failed")
	case errors.Is(err, errs.ErrInvalidCoupon):
		response.Error(c, http.StatusBadRequest, "invalid coupon")
	case errors.Is(err, errs.ErrCouponAlreadyExists):
		response.Error(c, http.StatusConflict, "coupon already exists")
	case errors.Is(err, errs.ErrCouponNotFound):
		response.Error(c, http.StatusNotFound, "coupon not found")
	default:
		response.Error(c, http.StatusInternalServerError, fallback)
	}
}

func (h *Handler) recordAudit(c *gin.Context, action, resourceID, result string, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	h.audit.RecordAdmin(c, action, "coupon", resourceID, result, metadata)
}

func safeAuditReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrInvalidID), errors.Is(err, errs.ErrValidationFailed), errors.Is(err, errs.ErrInvalidCoupon):
		return "validation_error"
	case errors.Is(err, errs.ErrCouponAlreadyExists):
		return "conflict"
	case errors.Is(err, errs.ErrCouponNotFound):
		return "not_found"
	default:
		return "internal_error"
	}
}
