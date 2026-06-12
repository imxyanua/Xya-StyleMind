package address

import (
	"context"
	"strings"
	"stylemind/internal/errs"

	"github.com/google/uuid"
)

type AddressRepository interface {
	List(ctx context.Context, userID string) ([]Address, error)
	Create(ctx context.Context, userID string, input AddressRequest) (*Address, error)
	Update(ctx context.Context, userID, addressID string, input AddressRequest) (*Address, error)
	Delete(ctx context.Context, userID, addressID string) error
	SetDefault(ctx context.Context, userID, addressID string) (*Address, error)
}

type Service struct {
	repo AddressRepository
}

func NewService(repo AddressRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string) ([]Address, error) {
	return s.repo.List(ctx, userID)
}

func (s *Service) Create(ctx context.Context, userID string, input AddressRequest) (*Address, error) {
	input, err := normalizeAddressInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, userID, input)
}

func (s *Service) Update(ctx context.Context, userID, addressID string, input AddressRequest) (*Address, error) {
	if _, err := uuid.Parse(addressID); err != nil {
		return nil, errs.ErrInvalidID
	}
	input, err := normalizeAddressInput(input)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, userID, addressID, input)
}

func (s *Service) Delete(ctx context.Context, userID, addressID string) error {
	if _, err := uuid.Parse(addressID); err != nil {
		return errs.ErrInvalidID
	}
	return s.repo.Delete(ctx, userID, addressID)
}

func (s *Service) SetDefault(ctx context.Context, userID, addressID string) (*Address, error) {
	if _, err := uuid.Parse(addressID); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.SetDefault(ctx, userID, addressID)
}

func normalizeAddressInput(input AddressRequest) (AddressRequest, error) {
	input.RecipientName = strings.TrimSpace(input.RecipientName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.AddressLine = strings.TrimSpace(input.AddressLine)
	input.City = strings.TrimSpace(input.City)
	input.District = strings.TrimSpace(input.District)
	input.Note = strings.TrimSpace(input.Note)

	if input.RecipientName == "" || input.Phone == "" || input.AddressLine == "" || input.City == "" || input.District == "" {
		return AddressRequest{}, errs.ErrValidationFailed
	}
	if len(input.RecipientName) < 2 || len(input.RecipientName) > 120 ||
		len(input.Phone) < 8 || len(input.Phone) > 32 ||
		len(input.AddressLine) < 5 || len(input.AddressLine) > 255 ||
		len(input.City) < 2 || len(input.City) > 120 ||
		len(input.District) < 2 || len(input.District) > 120 ||
		len(input.Note) > 1000 {
		return AddressRequest{}, errs.ErrValidationFailed
	}
	return input, nil
}
