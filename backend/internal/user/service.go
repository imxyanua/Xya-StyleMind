package user

import (
	"context"
	"strings"

	"stylemind/internal/errs"
)

type RepositoryPort interface {
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]User, int64, error)
	GetByID(ctx context.Context, id string) (*User, error)
	UpdateRole(ctx context.Context, actorUserID, targetUserID, newRole string) (*User, string, error)
}

type Service struct {
	repo RepositoryPort
}

func NewService(repo RepositoryPort) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]User, int64, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Role = strings.TrimSpace(filter.Role)
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Sort = strings.TrimSpace(filter.Sort)
	if filter.Role != "" && !IsValidRole(filter.Role) {
		return nil, 0, errs.ErrInvalidUserRole
	}
	if filter.Status != "" && !IsValidStatus(filter.Status) {
		return nil, 0, errs.ErrInvalidUserStatus
	}
	if filter.Sort != "" && filter.Sort != SortNewest && filter.Sort != SortOldest {
		return nil, 0, errs.ErrInvalidSort
	}
	return s.repo.List(ctx, filter, limit, offset)
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	if !IsUUID(id) {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateRole(ctx context.Context, actorUserID, targetUserID, role string) (*User, string, error) {
	role = strings.TrimSpace(role)
	if !IsUUID(targetUserID) {
		return nil, "", errs.ErrInvalidID
	}
	if actorUserID == "" || !IsUUID(actorUserID) {
		return nil, "", errs.ErrUnauthorized
	}
	if !IsValidRole(role) {
		return nil, "", errs.ErrInvalidUserRole
	}
	return s.repo.UpdateRole(ctx, actorUserID, targetUserID, role)
}

func IsValidRole(role string) bool {
	return role == RoleUser || role == RoleAdmin
}

func IsValidStatus(status string) bool {
	return status == StatusActive || status == StatusDisabled
}
