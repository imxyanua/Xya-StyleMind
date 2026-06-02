package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"stylemind/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimitStore interface {
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
	Close() error
}

type RateLimiter struct {
	store      RateLimitStore
	limit      int
	window     time.Duration
	failClosed bool
}

type RateLimitOption func(*RateLimiter)

func WithFailClosed(failClosed bool) RateLimitOption {
	return func(l *RateLimiter) {
		l.failClosed = failClosed
	}
}

func NewRateLimiter(store RateLimitStore, limit int, window time.Duration, opts ...RateLimitOption) *RateLimiter {
	if store == nil {
		store = NewMemoryRateLimitStore()
	}
	if limit <= 0 {
		limit = 10
	}
	if window <= 0 {
		window = time.Minute
	}

	limiter := &RateLimiter{
		store:  store,
		limit:  limit,
		window: window,
	}
	for _, opt := range opts {
		opt(limiter)
	}
	return limiter
}

func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	count, err := l.store.Increment(ctx, key, l.window)
	if err != nil {
		return !l.failClosed, err
	}
	return count <= l.limit, nil
}

func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return NewRateLimiter(NewMemoryRateLimitStore(), limit, window).Middleware("generic", IPKeyExtractor)
}

func (l *RateLimiter) Middleware(action string, extractors ...RateLimitKeyExtractor) gin.HandlerFunc {
	if len(extractors) == 0 {
		extractors = []RateLimitKeyExtractor{IPKeyExtractor}
	}

	return func(c *gin.Context) {
		for _, extractor := range extractors {
			keyPart, err := extractor(c)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, "rate limit failed")
				c.Abort()
				return
			}
			if keyPart == "" {
				continue
			}

			key := buildRateLimitKey(action, keyPart)
			allowed, err := l.Allow(c.Request.Context(), key)
			if err != nil && !allowed {
				response.Error(c, http.StatusTooManyRequests, "too many requests")
				c.Abort()
				return
			}
			if !allowed {
				response.Error(c, http.StatusTooManyRequests, "too many requests")
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

type RateLimitKeyExtractor func(c *gin.Context) (string, error)

func IPKeyExtractor(c *gin.Context) (string, error) {
	return "ip:" + c.ClientIP(), nil
}

func EmailKeyExtractor(c *gin.Context) (string, error) {
	if c.Request.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return "", nil
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", nil
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" {
		return "", nil
	}
	return "email:" + email, nil
}

func buildRateLimitKey(action, keyPart string) string {
	return "rl:" + sanitizeRateLimitKey(action) + ":" + sanitizeRateLimitKey(keyPart)
}

func sanitizeRateLimitKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

type MemoryRateLimitStore struct {
	mu      sync.Mutex
	buckets map[string]rateLimitBucket
}

type rateLimitBucket struct {
	count    int
	resetAt  time.Time
	lastSeen time.Time
}

func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{buckets: make(map[string]rateLimitBucket)}
}

func (s *MemoryRateLimitStore) Increment(_ context.Context, key string, window time.Duration) (int, error) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	bucket := s.buckets[key]
	if bucket.resetAt.IsZero() || now.After(bucket.resetAt) {
		bucket = rateLimitBucket{
			count:    0,
			resetAt:  now.Add(window),
			lastSeen: now,
		}
	}
	bucket.count++
	bucket.lastSeen = now
	s.buckets[key] = bucket

	if bucket.count%100 == 0 {
		for itemKey, item := range s.buckets {
			if now.Sub(item.lastSeen) > window*2 {
				delete(s.buckets, itemKey)
			}
		}
	}

	return bucket.count, nil
}

func (s *MemoryRateLimitStore) Close() error {
	return nil
}

type RedisRateLimitStore struct {
	client *redis.Client
}

func NewRedisRateLimitStore(addr, password string, db int) *RedisRateLimitStore {
	return &RedisRateLimitStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (s *RedisRateLimitStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := s.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	count := incr.Val()
	if count > int64(^uint(0)>>1) {
		return 0, errors.New("rate limit counter overflow")
	}
	return int(count), nil
}

func (s *RedisRateLimitStore) Close() error {
	return s.client.Close()
}

func NewRateLimitStoreFromConfig(addr, password, dbValue string) (RateLimitStore, bool, error) {
	if strings.TrimSpace(addr) == "" {
		// Local dev fallback: when Redis is not configured we keep the app usable with
		// per-process memory limits. Production should set REDIS_ADDR for multi-instance safety.
		return NewMemoryRateLimitStore(), false, nil
	}

	db, err := strconv.Atoi(dbValue)
	if err != nil {
		return nil, true, fmt.Errorf("invalid redis db: %w", err)
	}
	return NewRedisRateLimitStore(addr, password, db), true, nil
}
