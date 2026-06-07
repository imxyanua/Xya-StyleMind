package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

func (r *Repository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]User, int64, error) {
	whereSQL, args := buildWhere(filter)

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(ctx, `
		SELECT id, email, full_name, role, COALESCE(status, 'active'), created_at, updated_at
		FROM users
	`+whereSQL+`
		ORDER BY `+sortClause(filter.Sort)+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]User, 0)
	for rows.Next() {
		var item User
		if err := rows.Scan(&item.ID, &item.Email, &item.FullName, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id string) (*User, error) {
	item := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, full_name, role, COALESCE(status, 'active'), created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&item.ID, &item.Email, &item.FullName, &item.Role, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errs.ErrUserNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *Repository) UpdateRole(ctx context.Context, actorUserID, targetUserID, newRole string) (*User, string, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var current User
	err = tx.QueryRow(ctx, `
		SELECT id, email, full_name, role, COALESCE(status, 'active'), created_at, updated_at
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, targetUserID).Scan(&current.ID, &current.Email, &current.FullName, &current.Role, &current.Status, &current.CreatedAt, &current.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", errs.ErrUserNotFound
		}
		return nil, "", err
	}

	if current.Role == RoleAdmin && newRole == RoleUser && current.ID == actorUserID {
		var adminCount int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role = $1`, RoleAdmin).Scan(&adminCount); err != nil {
			return nil, "", err
		}
		if adminCount <= 1 {
			return nil, current.Role, errs.ErrCannotDemoteLastAdmin
		}
	}

	updated := &User{}
	err = tx.QueryRow(ctx, `
		UPDATE users
		SET role = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, full_name, role, COALESCE(status, 'active'), created_at, updated_at
	`, targetUserID, newRole).Scan(&updated.ID, &updated.Email, &updated.FullName, &updated.Role, &updated.Status, &updated.CreatedAt, &updated.UpdatedAt)
	if err != nil {
		return nil, current.Role, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, current.Role, err
	}
	return updated, current.Role, nil
}

func buildWhere(filter ListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if filter.Query != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(LOWER(email) LIKE "+placeholder+" OR LOWER(full_name) LIKE "+placeholder+")")
	}
	if filter.Role != "" {
		args = append(args, filter.Role)
		clauses = append(clauses, fmt.Sprintf("role = $%d", len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, fmt.Sprintf("COALESCE(status, 'active') = $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func sortClause(sort string) string {
	switch sort {
	case SortOldest:
		return "created_at ASC, id ASC"
	case "", SortNewest:
		return "created_at DESC, id DESC"
	default:
		return "created_at DESC, id DESC"
	}
}

func IsUUID(value string) bool {
	_, err := uuid.Parse(value)
	return err == nil
}
