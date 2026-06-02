package middleware

import (
	"net/http"
	"sync"
	"time"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
)

type rateLimitBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	var mu sync.Mutex
	buckets := make(map[string]rateLimitBucket)

	return func(c *gin.Context) {
		now := time.Now()
		key := c.ClientIP()

		mu.Lock()
		bucket := buckets[key]
		if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
			bucket = rateLimitBucket{
				count:    0,
				resetAt:  now.Add(window),
				lastSeen: now,
			}
		}
		bucket.count++
		bucket.lastSeen = now
		buckets[key] = bucket

		if bucket.count%100 == 0 {
			for ip, item := range buckets {
				if now.Sub(item.lastSeen) > window*2 {
					delete(buckets, ip)
				}
			}
		}
		allowed := bucket.count <= limit
		mu.Unlock()

		if !allowed {
			response.Error(c, http.StatusTooManyRequests, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
