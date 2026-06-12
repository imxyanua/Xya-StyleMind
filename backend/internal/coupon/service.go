package coupon

import (
	"context"
	"strings"
	"stylemind/internal/errs"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo RepositoryPort
	now  func() time.Time
}

type RepositoryPort interface {
	GetOrCreateCartID(ctx context.Context, userID string) (string, error)
	GetCartSubtotal(ctx context.Context, cartID string) (float64, error)
	GetByCode(ctx context.Context, code string) (*Coupon, error)
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]Coupon, int64, error)
	GetByID(ctx context.Context, id string) (*Coupon, error)
	Create(ctx context.Context, req MutationRequest) (*Coupon, error)
	Update(ctx context.Context, id string, req MutationRequest) (*Coupon, error)
	Delete(ctx context.Context, id string) error
}

func NewService(repo RepositoryPort) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) ApplyToCart(ctx context.Context, userID, code string) (*ApplyResult, error) {
	code = NormalizeCode(code)
	if code == "" {
		return nil, errs.ErrInvalidCoupon
	}
	cartID, err := s.repo.GetOrCreateCartID(ctx, userID)
	if err != nil {
		return nil, err
	}
	subtotal, err := s.repo.GetCartSubtotal(ctx, cartID)
	if err != nil {
		return nil, err
	}
	c, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if err := ValidateForSubtotal(*c, subtotal, s.now()); err != nil {
		return nil, err
	}
	discount := CalculateDiscount(subtotal, *c)
	return &ApplyResult{
		CouponID:       c.ID,
		CouponCode:     c.Code,
		Type:           c.Type,
		Value:          c.Value,
		SubtotalAmount: subtotal,
		DiscountAmount: discount,
		TotalAmount:    roundMoney(subtotal - discount),
	}, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Coupon, int64, error) {
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Type = strings.TrimSpace(filter.Type)
	filter.Sort = strings.TrimSpace(filter.Sort)
	if filter.Type != "" && filter.Type != TypePercent && filter.Type != TypeFixed {
		return nil, 0, errs.ErrInvalidCoupon
	}
	if filter.Sort != "" && filter.Sort != SortNewest && filter.Sort != SortOldest {
		return nil, 0, errs.ErrInvalidSort
	}
	return s.repo.List(ctx, filter, limit, offset)
}

func (s *Service) Get(ctx context.Context, id string) (*Coupon, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req MutationRequest) (*Coupon, error) {
	req, err := normalizeMutation(req)
	if err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, req)
}

func (s *Service) Update(ctx context.Context, id string, req MutationRequest) (*Coupon, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, errs.ErrInvalidID
	}
	req, err := normalizeMutation(req)
	if err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, id, req)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return errs.ErrInvalidID
	}
	return s.repo.Delete(ctx, id)
}

func normalizeMutation(req MutationRequest) (MutationRequest, error) {
	req.Code = NormalizeCode(req.Code)
	req.Type = strings.TrimSpace(req.Type)
	if req.Code == "" || len(req.Code) > 64 {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	if req.Type != TypePercent && req.Type != TypeFixed {
		return MutationRequest{}, errs.ErrInvalidCoupon
	}
	if req.Value <= 0 || (req.Type == TypePercent && req.Value > 100) {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	if req.MinOrderAmount < 0 {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	if req.MaxDiscountAmount != nil && *req.MaxDiscountAmount < 0 {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	if req.UsageLimit != nil && *req.UsageLimit <= 0 {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	if req.StartsAt != nil && req.ExpiresAt != nil && !req.ExpiresAt.After(*req.StartsAt) {
		return MutationRequest{}, errs.ErrValidationFailed
	}
	return req, nil
}

func ValidateForSubtotal(c Coupon, subtotal float64, now time.Time) error {
	if !c.IsActive {
		return errs.ErrCouponInactive
	}
	if c.StartsAt != nil && now.Before(*c.StartsAt) {
		return errs.ErrCouponExpired
	}
	if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
		return errs.ErrCouponExpired
	}
	if c.UsageLimit != nil && c.UsedCount >= *c.UsageLimit {
		return errs.ErrCouponUsageLimitReached
	}
	if subtotal < c.MinOrderAmount {
		return errs.ErrCouponMinOrderNotMet
	}
	return nil
}

func IsCouponValidationError(err error) bool {
	return err == errs.ErrCouponNotFound ||
		err == errs.ErrCouponInactive ||
		err == errs.ErrCouponExpired ||
		err == errs.ErrCouponUsageLimitReached ||
		err == errs.ErrCouponMinOrderNotMet ||
		err == errs.ErrInvalidCoupon
}
