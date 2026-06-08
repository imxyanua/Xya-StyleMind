package auth

import (
	"context"
	"errors"
	"strings"
	"stylemind/internal/errs"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo            UserRepository
	tokenConfig     TokenConfig
	revocationStore TokenRevocationStore
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

type ServiceOption func(*Service)

func WithTokenRevocationStore(store TokenRevocationStore) ServiceOption {
	return func(s *Service) {
		s.revocationStore = store
	}
}

func NewService(repo UserRepository, tokenConfig TokenConfig, opts ...ServiceOption) *Service {
	service := &Service{repo: repo, tokenConfig: tokenConfig}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (map[string]interface{}, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	_, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil {
		return nil, errs.ErrEmailAlreadyExists
	}
	if !errors.Is(err, errs.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.NewString(),
		Email:        email,
		FullName:     strings.TrimSpace(req.FullName),
		PasswordHash: string(hash),
		Role:         "user",
		Status:       "active",
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := GenerateToken(s.tokenConfig, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
			"status":    user.Status,
		},
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (map[string]interface{}, error) {
	user, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		if errors.Is(err, errs.ErrUserNotFound) {
			return nil, errs.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errs.ErrInvalidCredentials
	}
	if normalizedUserStatus(user.Status) == "disabled" {
		return nil, errs.ErrUserDisabled
	}

	token, err := GenerateToken(s.tokenConfig, user.ID, user.Role)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":        user.ID,
			"email":     user.Email,
			"full_name": user.FullName,
			"role":      user.Role,
			"status":    normalizedUserStatus(user.Status),
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" {
		return errs.ErrUnauthorized
	}
	if s.revocationStore == nil {
		return nil
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return s.revocationStore.RevokeToken(ctx, jti, ttl)
}

func normalizedUserStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "active"
	}
	return strings.TrimSpace(status)
}
