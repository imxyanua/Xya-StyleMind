package auth

import (
	"context"
	"errors"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo        UserRepository
	tokenConfig TokenConfig
}

type UserRepository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
}

func NewService(repo UserRepository, tokenConfig TokenConfig) *Service {
	return &Service{repo: repo, tokenConfig: tokenConfig}
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
		},
	}, nil
}
