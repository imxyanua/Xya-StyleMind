package user

import (
	"errors"
	"net/http"
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

func RegisterRoutes(admin *gin.RouterGroup, service *Service, recorders ...audit.Recorder) {
	var recorder audit.Recorder
	if len(recorders) > 0 {
		recorder = recorders[0]
	}
	h := &Handler{service: service, audit: recorder}
	admin.GET("/users", h.List)
	admin.GET("/users/:id", h.Get)
	admin.PATCH("/users/:id/role", h.UpdateRole)
}

func (h *Handler) List(c *gin.Context) {
	page := pagination.Parse(c)
	filter := ListFilter{
		Query:  strings.TrimSpace(c.Query("q")),
		Role:   strings.TrimSpace(c.Query("role")),
		Status: strings.TrimSpace(c.Query("status")),
		Sort:   strings.TrimSpace(c.DefaultQuery("sort", SortNewest)),
	}
	items, total, err := h.service.List(c.Request.Context(), filter, page.Limit, page.Offset)
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidUserRole):
			response.Error(c, http.StatusBadRequest, "invalid role")
		case errors.Is(err, errs.ErrInvalidUserStatus):
			response.Error(c, http.StatusBadRequest, "invalid status")
		case errors.Is(err, errs.ErrInvalidSort):
			response.Error(c, http.StatusBadRequest, "invalid sort")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to fetch users")
		}
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) Get(c *gin.Context) {
	item, err := h.service.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		switch {
		case errors.Is(err, errs.ErrInvalidID):
			response.Error(c, http.StatusBadRequest, "invalid user id")
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(c, http.StatusNotFound, "user not found")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to fetch user")
		}
		return
	}
	response.Success(c, http.StatusOK, "ok", item)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.recordAudit(c, c.Param("id"), audit.ResultFailed, gin.H{"reason": "invalid_payload"})
		response.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := validator.Validate.Struct(req); err != nil {
		h.recordAudit(c, c.Param("id"), audit.ResultFailed, gin.H{"new_role": req.Role, "reason": "validation_error"})
		response.Error(c, http.StatusBadRequest, "validation failed")
		return
	}

	item, oldRole, err := h.service.UpdateRole(c.Request.Context(), c.GetString("user_id"), c.Param("id"), req.Role)
	if err != nil {
		metadata := gin.H{"new_role": req.Role, "reason": safeUserAuditReason(err)}
		if oldRole != "" {
			metadata["old_role"] = oldRole
		}
		h.recordAudit(c, c.Param("id"), audit.ResultFailed, metadata)
		switch {
		case errors.Is(err, errs.ErrInvalidID):
			response.Error(c, http.StatusBadRequest, "invalid user id")
		case errors.Is(err, errs.ErrInvalidUserRole):
			response.Error(c, http.StatusBadRequest, "invalid role")
		case errors.Is(err, errs.ErrUnauthorized):
			response.Error(c, http.StatusUnauthorized, "unauthorized")
		case errors.Is(err, errs.ErrUserNotFound):
			response.Error(c, http.StatusNotFound, "user not found")
		case errors.Is(err, errs.ErrCannotDemoteLastAdmin):
			response.Error(c, http.StatusConflict, "cannot demote the last admin")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to update user role")
		}
		return
	}

	h.recordAudit(c, item.ID, audit.ResultSuccess, gin.H{"old_role": oldRole, "new_role": item.Role, "email": item.Email})
	response.Success(c, http.StatusOK, "user role updated", item)
}

func (h *Handler) recordAudit(c *gin.Context, resourceID, result string, metadata map[string]any) {
	if h.audit == nil {
		return
	}
	h.audit.RecordAdmin(c, "admin.user_role.update", "user", resourceID, result, metadata)
}

func safeUserAuditReason(err error) string {
	switch {
	case errors.Is(err, errs.ErrInvalidID), errors.Is(err, errs.ErrInvalidUserRole), errors.Is(err, errs.ErrValidationFailed):
		return "validation_error"
	case errors.Is(err, errs.ErrUnauthorized):
		return "unauthorized"
	case errors.Is(err, errs.ErrUserNotFound):
		return "not_found"
	case errors.Is(err, errs.ErrCannotDemoteLastAdmin):
		return "last_admin"
	default:
		return "internal_error"
	}
}
