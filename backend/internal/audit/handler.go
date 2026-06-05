package audit

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"stylemind/internal/errs"
	"stylemind/pkg/pagination"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type Recorder interface {
	RecordAdmin(c *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any)
}

type Handler struct {
	service *Service
}

func RegisterRoutes(admin *gin.RouterGroup, service *Service) {
	h := &Handler{service: service}
	admin.GET("/audit-logs", h.List)
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
		switch {
		case errors.Is(err, errs.ErrInvalidID):
			response.Error(c, http.StatusBadRequest, "invalid actor_user_id")
		case errors.Is(err, errs.ErrInvalidSort):
			response.Error(c, http.StatusBadRequest, "invalid sort")
		case errors.Is(err, errs.ErrValidationFailed):
			response.Error(c, http.StatusBadRequest, "validation failed")
		default:
			response.Error(c, http.StatusInternalServerError, "failed to fetch audit logs")
		}
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}

func parseListFilter(c *gin.Context) (ListFilter, error) {
	from, err := parseOptionalTime(c.Query("from"), false)
	if err != nil {
		return ListFilter{}, errors.New("invalid from")
	}
	to, err := parseOptionalTime(c.Query("to"), true)
	if err != nil {
		return ListFilter{}, errors.New("invalid to")
	}
	return ListFilter{
		Action:       strings.TrimSpace(c.Query("action")),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		ActorUserID:  strings.TrimSpace(c.Query("actor_user_id")),
		Result:       strings.TrimSpace(c.Query("result")),
		From:         from,
		To:           to,
		Sort:         strings.TrimSpace(c.DefaultQuery("sort", SortNewest)),
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
