package dashboard

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"stylemind/internal/errs"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func RegisterRoutes(admin *gin.RouterGroup, service *Service) {
	h := &Handler{service: service}
	admin.GET("/dashboard/stats", h.GetStats)
}

func (h *Handler) GetStats(c *gin.Context) {
	filter, err := parseStatsFilter(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	stats, err := h.service.GetStats(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, errs.ErrValidationFailed) {
			response.Error(c, http.StatusBadRequest, "validation failed")
			return
		}
		response.Error(c, http.StatusInternalServerError, "failed to fetch dashboard stats")
		return
	}
	response.Success(c, http.StatusOK, "ok", stats)
}

func parseStatsFilter(c *gin.Context) (StatsFilter, error) {
	from, err := parseOptionalTime(c.Query("from"), false)
	if err != nil {
		return StatsFilter{}, errors.New("invalid from")
	}
	to, err := parseOptionalTime(c.Query("to"), true)
	if err != nil {
		return StatsFilter{}, errors.New("invalid to")
	}
	return StatsFilter{From: from, To: to}, nil
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
