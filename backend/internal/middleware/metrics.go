package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const unmatchedRoute = "unmatched"

type HTTPMetrics struct {
	registry        *prometheus.Registry
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	errorsTotal     *prometheus.CounterVec
}

func NewHTTPMetrics() *HTTPMetrics {
	registry := prometheus.NewRegistry()
	metrics := &HTTPMetrics{
		registry: registry,
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests processed.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route", "status"},
		),
		errorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP requests with 4xx or 5xx status codes.",
			},
			[]string{"method", "route", "status"},
		),
	}
	registry.MustRegister(metrics.requestsTotal, metrics.requestDuration, metrics.errorsTotal)
	return metrics
}

func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		labels := prometheus.Labels{
			"method": c.Request.Method,
			"route":  routePattern(c),
			"status": strconv.Itoa(status),
		}
		m.requestsTotal.With(labels).Inc()
		m.requestDuration.With(labels).Observe(time.Since(start).Seconds())
		if status >= http.StatusBadRequest {
			m.errorsTotal.With(labels).Inc()
		}
	}
}

func (m *HTTPMetrics) RegisterRoutes(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})))
}

func routePattern(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		return unmatchedRoute
	}
	return route
}
