package address

import (
	"context"
	"errors"
	"stylemind/internal/errs"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context, userID string) ([]Address, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, recipient_name, phone, address_line, city, district, COALESCE(note, ''), is_default, created_at, updated_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Address, 0)
	for rows.Next() {
		var item Address
		if err := scanAddress(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, userID string, input AddressRequest) (*Address, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if input.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE user_addresses SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	item := &Address{}
	err = tx.QueryRow(ctx, `
		INSERT INTO user_addresses (id, user_id, recipient_name, phone, address_line, city, district, note, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)
		RETURNING id, user_id, recipient_name, phone, address_line, city, district, COALESCE(note, ''), is_default, created_at, updated_at
	`, uuid.NewString(), userID, input.RecipientName, input.Phone, input.AddressLine, input.City, input.District, input.Note, input.IsDefault).Scan(
		&item.ID, &item.UserID, &item.RecipientName, &item.Phone, &item.AddressLine, &item.City, &item.District, &item.Note, &item.IsDefault, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) Update(ctx context.Context, userID, addressID string, input AddressRequest) (*Address, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if input.IsDefault {
		if _, err := tx.Exec(ctx, `UPDATE user_addresses SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
			return nil, err
		}
	}

	item := &Address{}
	err = tx.QueryRow(ctx, `
		UPDATE user_addresses
		SET recipient_name = $3,
		    phone = $4,
		    address_line = $5,
		    city = $6,
		    district = $7,
		    note = NULLIF($8, ''),
		    is_default = $9,
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, recipient_name, phone, address_line, city, district, COALESCE(note, ''), is_default, created_at, updated_at
	`, addressID, userID, input.RecipientName, input.Phone, input.AddressLine, input.City, input.District, input.Note, input.IsDefault).Scan(
		&item.ID, &item.UserID, &item.RecipientName, &item.Phone, &item.AddressLine, &item.City, &item.District, &item.Note, &item.IsDefault, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrAddressNotFound
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

func (r *Repository) Delete(ctx context.Context, userID, addressID string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM user_addresses WHERE id = $1 AND user_id = $2`, addressID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrAddressNotFound
	}
	return nil
}

func (r *Repository) SetDefault(ctx context.Context, userID, addressID string) (*Address, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM user_addresses WHERE id = $1 AND user_id = $2)`, addressID, userID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, errs.ErrAddressNotFound
	}
	if _, err := tx.Exec(ctx, `UPDATE user_addresses SET is_default = FALSE, updated_at = NOW() WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}

	item := &Address{}
	err = tx.QueryRow(ctx, `
		UPDATE user_addresses
		SET is_default = TRUE, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, recipient_name, phone, address_line, city, district, COALESCE(note, ''), is_default, created_at, updated_at
	`, addressID, userID).Scan(
		&item.ID, &item.UserID, &item.RecipientName, &item.Phone, &item.AddressLine, &item.City, &item.District, &item.Note, &item.IsDefault, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrAddressNotFound
		}
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAddress(row rowScanner, item *Address) error {
	return row.Scan(
		&item.ID, &item.UserID, &item.RecipientName, &item.Phone, &item.AddressLine,
		&item.City, &item.District, &item.Note, &item.IsDefault, &item.CreatedAt, &item.UpdatedAt,
	)
}
