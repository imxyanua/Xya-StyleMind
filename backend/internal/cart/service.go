package cart

import (
	"context"
	"errors"
	"stylemind/internal/errs"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetCart(ctx context.Context, userID string) (*CartResponse, error) {
	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildCartResponse(ctx, cart)
}

func (s *Service) AddItem(ctx context.Context, userID string, req AddCartItemRequest) (*CartResponse, error) {
	if req.Quantity <= 0 {
		return nil, errs.ErrInvalidQuantity
	}

	product, err := s.repo.GetProductSnapshot(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}

	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetCartItemByProduct(ctx, cart.ID, req.ProductID)
	if err != nil && !errors.Is(err, errs.ErrCartItemNotFound) {
		return nil, err
	}

	if errors.Is(err, errs.ErrCartItemNotFound) {
		if req.Quantity > product.Stock {
			return nil, errs.ErrInsufficientStock
		}
		if err := s.repo.CreateCartItem(ctx, cart.ID, req.ProductID, req.Quantity); err != nil {
			return nil, err
		}
	} else {
		nextQty := existing.Quantity + req.Quantity
		if nextQty > product.Stock {
			return nil, errs.ErrInsufficientStock
		}
		if err := s.repo.UpdateCartItemQuantity(ctx, existing.ID, nextQty); err != nil {
			return nil, err
		}
	}

	return s.buildCartResponse(ctx, cart)
}

func (s *Service) UpdateItem(ctx context.Context, userID, itemID string, quantity int) (*CartResponse, error) {
	if quantity <= 0 {
		return nil, errs.ErrInvalidQuantity
	}

	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	item, err := s.repo.GetCartItemByID(ctx, cart.ID, itemID)
	if err != nil {
		return nil, err
	}
	if quantity > item.Product.Stock {
		return nil, errs.ErrInsufficientStock
	}

	if err := s.repo.UpdateCartItemQuantity(ctx, item.ID, quantity); err != nil {
		return nil, err
	}
	return s.buildCartResponse(ctx, cart)
}

func (s *Service) DeleteItem(ctx context.Context, userID, itemID string) (*CartResponse, error) {
	cart, err := s.repo.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.DeleteCartItem(ctx, cart.ID, itemID); err != nil {
		return nil, err
	}
	return s.buildCartResponse(ctx, cart)
}

func (s *Service) buildCartResponse(ctx context.Context, cart *Cart) (*CartResponse, error) {
	items, err := s.repo.ListCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	resp := &CartResponse{
		CartID: cart.ID,
		UserID: cart.UserID,
		Items:  make([]CartItemResponse, 0, len(items)),
		Total:  0,
	}

	for _, item := range items {
		subtotal := item.Product.Price * float64(item.Quantity)
		resp.Items = append(resp.Items, CartItemResponse{
			ID:       item.ID,
			Product:  item.Product,
			Quantity: item.Quantity,
			Subtotal: subtotal,
		})
		resp.Total += subtotal
	}

	return resp, nil
}
