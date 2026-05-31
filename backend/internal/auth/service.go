package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type Service struct {
	repo      *Repository
	jwtSecret string
}

func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (map[string]interface{}, error) {
	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, errors.New("email already exists")
	}
	if !errors.Is(err, ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           uuid.NewString(),
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		FullName:     strings.TrimSpace(req.FullName),
		PasswordHash: string(hash),
		Role:         "user",
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	token, err := GenerateToken(s.jwtSecret, user.ID, user.Role)
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
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := GenerateToken(s.jwtSecret, user.ID, user.Role)
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
