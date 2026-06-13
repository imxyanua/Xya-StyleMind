package inventory

import (
	"net/http"

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
	me.GET("/reservations", h.ListMine)
}

func (h *Handler) ListMine(c *gin.Context) {
	page := pagination.Parse(c)
	items, total, err := h.service.ListMine(c.Request.Context(), c.GetString("user_id"), page.Limit, page.Offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to fetch reservations")
		return
	}
	response.SuccessWithMeta(c, http.StatusOK, "ok", items, pagination.BuildMeta(page.Page, page.Limit, total))
}
