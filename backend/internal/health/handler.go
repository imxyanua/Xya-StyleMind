package health

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"stylemind/internal/config"
	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	StatusUp      = "up"
	StatusDown    = "down"
	StatusSkipped = "skipped"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	postgres        Pinger
	redis           Pinger
	redisConfigured bool
}

type DependencyStatus struct {
	Status string `json:"status"`
}

type ReadinessResponse struct {
	Status       string                      `json:"status"`
	Dependencies map[string]DependencyStatus `json:"dependencies"`
}

func RegisterRoutes(router *gin.Engine, api *gin.RouterGroup, postgres Pinger, redis Pinger, redisConfigured bool) {
	h := &Handler{
		postgres:        postgres,
		redis:           redis,
		redisConfigured: redisConfigured,
	}

	router.GET("/health", h.Healthz)
	router.GET("/healthz", h.Healthz)
	router.GET("/livez", h.Livez)
	router.GET("/readyz", h.Readyz)

	// Backward-compatible lightweight health endpoint for older local checks.
	api.GET("/health", h.Healthz)
}

func (h *Handler) Healthz(c *gin.Context) {
	response.Success(c, http.StatusOK, "ok", gin.H{
		"status":  StatusUp,
		"service": "stylemind-backend",
	})
}

func (h *Handler) Livez(c *gin.Context) {
	response.Success(c, http.StatusOK, "ok", gin.H{
		"status": StatusUp,
	})
}

func (h *Handler) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	ready := true
	dependencies := map[string]DependencyStatus{
		"postgres": {Status: StatusUp},
		"redis":    {Status: StatusSkipped},
	}

	if h.postgres == nil || h.postgres.Ping(ctx) != nil {
		ready = false
		dependencies["postgres"] = DependencyStatus{Status: StatusDown}
	}

	if h.redisConfigured {
		dependencies["redis"] = DependencyStatus{Status: StatusUp}
		if h.redis == nil || h.redis.Ping(ctx) != nil {
			ready = false
			dependencies["redis"] = DependencyStatus{Status: StatusDown}
		}
	}

	status := "ready"
	code := http.StatusOK
	message := "ok"
	if !ready {
		status = "not_ready"
		code = http.StatusServiceUnavailable
		message = "not ready"
	}

	c.JSON(code, response.APIResponse{
		Success: ready,
		Message: message,
		Data: ReadinessResponse{
			Status:       status,
			Dependencies: dependencies,
		},
	})
}

func NewPostgresChecker(db *pgxpool.Pool) Pinger {
	return db
}

type RedisChecker struct {
	client *redis.Client
}

func NewRedisChecker(redisConfig config.RedisConfig) (*RedisChecker, error) {
	db, err := parseRedisDB(redisConfig.DB)
	if err != nil {
		return nil, err
	}
	return &RedisChecker{
		client: redis.NewClient(&redis.Options{
			Addr:     redisConfig.Addr,
			Password: redisConfig.Password,
			DB:       db,
		}),
	}, nil
}

func (c *RedisChecker) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisChecker) Close() error {
	return c.client.Close()
}

func parseRedisDB(value string) (int, error) {
	return strconv.Atoi(value)
}
