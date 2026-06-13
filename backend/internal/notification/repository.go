package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"stylemind/internal/errs"
	"time"

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

func (r *Repository) Create(ctx context.Context, input CreateInput) (*Notification, error) {
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	_, err = r.db.Exec(ctx, `
		INSERT INTO notifications (id, user_id, type, title, message, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, input.UserID, input.Type, input.Title, input.Message, rawMetadata)
	if err != nil {
		return nil, err
	}
	return r.GetByIDForUser(ctx, id, input.UserID)
}

func (r *Repository) GetPreferences(ctx context.Context, userID string) (*Preferences, error) {
	row := r.db.QueryRow(ctx, `
		SELECT user_id, order_updates_enabled, payment_updates_enabled, return_updates_enabled,
		       promotion_enabled, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`, userID)
	prefs, err := scanPreferences(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultPreferences(userID), nil
		}
		return nil, err
	}
	return &prefs, nil
}

func (r *Repository) UpdatePreferences(ctx context.Context, userID string, input UpdatePreferencesInput) (*Preferences, error) {
	current, err := r.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if input.OrderUpdatesEnabled != nil {
		current.OrderUpdatesEnabled = *input.OrderUpdatesEnabled
	}
	if input.PaymentUpdatesEnabled != nil {
		current.PaymentUpdatesEnabled = *input.PaymentUpdatesEnabled
	}
	if input.ReturnUpdatesEnabled != nil {
		current.ReturnUpdatesEnabled = *input.ReturnUpdatesEnabled
	}
	if input.PromotionEnabled != nil {
		current.PromotionEnabled = *input.PromotionEnabled
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO notification_preferences (
			user_id, order_updates_enabled, payment_updates_enabled,
			return_updates_enabled, promotion_enabled
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			order_updates_enabled = EXCLUDED.order_updates_enabled,
			payment_updates_enabled = EXCLUDED.payment_updates_enabled,
			return_updates_enabled = EXCLUDED.return_updates_enabled,
			promotion_enabled = EXCLUDED.promotion_enabled,
			updated_at = NOW()
		RETURNING user_id, order_updates_enabled, payment_updates_enabled, return_updates_enabled,
		          promotion_enabled, created_at, updated_at
	`, userID, current.OrderUpdatesEnabled, current.PaymentUpdatesEnabled, current.ReturnUpdatesEnabled, current.PromotionEnabled)
	prefs, err := scanPreferences(row)
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string, filter ListFilter, limit, offset int) ([]Notification, int64, error) {
	whereSQL, args := buildWhere(userID, filter)
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM notifications`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, type, title, message, metadata, read_at, created_at
		FROM notifications
	`+whereSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByIDForUser(ctx context.Context, id, userID string) (*Notification, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_id, type, title, message, metadata, read_at, created_at
		FROM notifications
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	item, err := scanNotification(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrNotificationNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *Repository) MarkRead(ctx context.Context, userID, id string) (*Notification, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, NOW())
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, errs.ErrNotificationNotFound
	}
	return r.GetByIDForUser(ctx, id, userID)
}

func (r *Repository) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, NOW())
		WHERE user_id = $1 AND read_at IS NULL
	`, userID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanNotification(row scanner) (Notification, error) {
	var item Notification
	var rawMetadata []byte
	if err := row.Scan(
		&item.ID, &item.UserID, &item.Type, &item.Title, &item.Message,
		&rawMetadata, &item.ReadAt, &item.CreatedAt,
	); err != nil {
		return Notification{}, err
	}
	item.Metadata = map[string]any{}
	if len(rawMetadata) > 0 {
		if err := json.Unmarshal(rawMetadata, &item.Metadata); err != nil {
			return Notification{}, err
		}
	}
	return item, nil
}

func scanPreferences(row scanner) (Preferences, error) {
	var prefs Preferences
	err := row.Scan(
		&prefs.UserID,
		&prefs.OrderUpdatesEnabled,
		&prefs.PaymentUpdatesEnabled,
		&prefs.ReturnUpdatesEnabled,
		&prefs.PromotionEnabled,
		&prefs.CreatedAt,
		&prefs.UpdatedAt,
	)
	return prefs, err
}

func defaultPreferences(userID string) *Preferences {
	now := time.Now().UTC()
	return &Preferences{
		UserID:                userID,
		OrderUpdatesEnabled:   true,
		PaymentUpdatesEnabled: true,
		ReturnUpdatesEnabled:  true,
		PromotionEnabled:      true,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func buildWhere(userID string, filter ListFilter) (string, []any) {
	clauses := []string{"user_id = $1"}
	args := []any{userID}
	if filter.UnreadOnly {
		clauses = append(clauses, "read_at IS NULL")
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
