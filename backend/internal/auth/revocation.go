package auth

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const RevokedTokenKeyPrefix = "auth:revoked:jti:"

type TokenRevocationStore interface {
	RevokeToken(ctx context.Context, jti string, ttl time.Duration) error
	IsTokenRevoked(ctx context.Context, jti string) (bool, error)
	Close() error
}

type MemoryTokenRevocationStore struct {
	mu      sync.Mutex
	revoked map[string]time.Time
}

func NewMemoryTokenRevocationStore() *MemoryTokenRevocationStore {
	return &MemoryTokenRevocationStore{revoked: make(map[string]time.Time)}
}

func (s *MemoryTokenRevocationStore) RevokeToken(_ context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[jti] = time.Now().Add(ttl)
	return nil
}

func (s *MemoryTokenRevocationStore) IsTokenRevoked(_ context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	expiresAt, ok := s.revoked[jti]
	if !ok {
		return false, nil
	}
	if time.Now().After(expiresAt) {
		delete(s.revoked, jti)
		return false, nil
	}
	return true, nil
}

func (s *MemoryTokenRevocationStore) Close() error {
	return nil
}

type RedisTokenRevocationStore struct {
	client redis.UniversalClient
}

func NewRedisTokenRevocationStore(addr, password string, db int) *RedisTokenRevocationStore {
	return &RedisTokenRevocationStore{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
	}
}

func (s *RedisTokenRevocationStore) RevokeToken(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	return s.client.Set(ctx, RevokedTokenKeyPrefix+jti, "1", ttl).Err()
}

func (s *RedisTokenRevocationStore) IsTokenRevoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}

	value, err := s.client.Exists(ctx, RevokedTokenKeyPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return value > 0, nil
}

func (s *RedisTokenRevocationStore) Close() error {
	return s.client.Close()
}

func NewTokenRevocationStoreFromConfig(addr, password, dbValue string) (TokenRevocationStore, bool, error) {
	if addr == "" {
		// Local dev fallback: revocation works within a single backend process.
		// Production/multi-instance deployments should set REDIS_ADDR.
		return NewMemoryTokenRevocationStore(), false, nil
	}

	db, err := strconv.Atoi(dbValue)
	if err != nil {
		return nil, false, err
	}
	return NewRedisTokenRevocationStore(addr, password, db), true, nil
}
