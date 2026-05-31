package health

import (
	"context"
	"net/http"
	"time"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db *pgxpool.Pool
}

func RegisterRoutes(group *gin.RouterGroup, db *pgxpool.Pool) {
	h := &Handler{db: db}
	group.GET("/health", h.Check)
}

func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		response.Error(c, http.StatusServiceUnavailable, "database unavailable", gin.H{"database": "down"})
		return
	}

	response.Success(c, http.StatusOK, "ok", gin.H{
		"service":  "stylemind-backend",
		"database": "up",
	})
}
