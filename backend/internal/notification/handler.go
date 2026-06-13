package notification

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"stylemind/internal/errs"
	"stylemind/pkg/pagination"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc, service *Service) {
	h := &Handler{service: service}
	me := api.Group("/me")
	me.Use(authMiddleware)
	me.GET("/notifications", h.ListMine)
	me.PATCH("/notifications/:id/read", h.MarkRead)
	me.PATCH("/notifications/read-all", h.MarkAllRead)
}

func (h *Handler) ListMine(c *gin.Context) {
	page := pagination.Parse(c)
	filter, err := parseFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid unread")
		return
	}
	items, total, err := h.service.ListMine(c.Request.Context(), c.GetString("user_id"), filter, page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch notifications")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func (h *Handler) MarkRead(c *gin.Context) {
	item, err := h.service.MarkRead(c.Request.Context(), c.GetString("user_id"), c.Param("id"))
	if err != nil {
		writeReadError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "notification marked read", item)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	updated, err := h.service.MarkAllRead(c.Request.Context(), c.GetString("user_id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to mark notifications read")
		return
	}
	response.Success(c, http.StatusOK, "notifications marked read", MarkAllReadResponse{Updated: updated})
}

func parseFilter(c *gin.Context) (ListFilter, error) {
	rawUnread := strings.TrimSpace(c.Query("unread"))
	if rawUnread == "" {
		return ListFilter{}, nil
	}
	value, err := strconv.ParseBool(rawUnread)
	if err != nil {
		return ListFilter{}, err
	}
	return ListFilter{UnreadOnly: value}, nil
}

func writeReadError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errs.ErrInvalidID):
		response.Error(c, http.StatusBadRequest, "invalid notification id")
	case errors.Is(err, errs.ErrNotificationNotFound):
		response.Error(c, http.StatusNotFound, "notification not found")
	default:
		response.Error(c, http.StatusInternalServerError, "failed to update notification")
	}
}
