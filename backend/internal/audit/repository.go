package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, params CreateLogParams) (*Log, error) {
	id := uuid.NewString()
	metadata, err := json.Marshal(params.Metadata)
	if err != nil {
		metadata = []byte(`{}`)
	}
	var actorUserID any
	if strings.TrimSpace(params.ActorUserID) != "" {
		actorUserID = params.ActorUserID
	}

	log := &Log{}
	var rawMetadata []byte
	err = r.db.QueryRow(ctx, `
		INSERT INTO audit_logs (id, actor_user_id, actor_role, action, resource_type, resource_id, result, metadata, ip, user_agent, request_id)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8::jsonb, NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''))
		RETURNING id, COALESCE(actor_user_id::text, ''), actor_role, action, resource_type, COALESCE(resource_id, ''), result, metadata, COALESCE(ip, ''), COALESCE(user_agent, ''), COALESCE(request_id, ''), created_at
	`, id, actorUserID, params.ActorRole, params.Action, params.ResourceType, params.ResourceID, params.Result, string(metadata), params.IP, params.UserAgent, params.RequestID).Scan(
		&log.ID, &log.ActorUserID, &log.ActorRole, &log.Action, &log.ResourceType, &log.ResourceID, &log.Result, &rawMetadata, &log.IP, &log.UserAgent, &log.RequestID, &log.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	log.Metadata = decodeMetadata(rawMetadata)
	return log, nil
}

func (r *Repository) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Log, int64, error) {
	whereSQL, args := buildAuditWhere(filter)
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs`+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitPlaceholder := fmt.Sprintf("$%d", len(args)+1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args)+2)
	queryArgs := append(append([]any{}, args...), limit, offset)
	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(actor_user_id::text, ''), actor_role, action, resource_type, COALESCE(resource_id, ''), result, metadata, COALESCE(ip, ''), COALESCE(user_agent, ''), COALESCE(request_id, ''), created_at
		FROM audit_logs
	`+whereSQL+`
		ORDER BY `+auditSortClause(filter.Sort)+`
		LIMIT `+limitPlaceholder+` OFFSET `+offsetPlaceholder, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]Log, 0)
	for rows.Next() {
		var item Log
		var rawMetadata []byte
		if err := rows.Scan(&item.ID, &item.ActorUserID, &item.ActorRole, &item.Action, &item.ResourceType, &item.ResourceID, &item.Result, &rawMetadata, &item.IP, &item.UserAgent, &item.RequestID, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		item.Metadata = decodeMetadata(rawMetadata)
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func decodeMetadata(raw []byte) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func buildAuditWhere(filter ListFilter) (string, []any) {
	clauses := make([]string, 0)
	args := make([]any, 0)
	if filter.Action != "" {
		args = append(args, filter.Action)
		clauses = append(clauses, fmt.Sprintf("action = $%d", len(args)))
	}
	if filter.ResourceType != "" {
		args = append(args, filter.ResourceType)
		clauses = append(clauses, fmt.Sprintf("resource_type = $%d", len(args)))
	}
	if filter.ActorUserID != "" {
		args = append(args, filter.ActorUserID)
		clauses = append(clauses, fmt.Sprintf("actor_user_id = $%d::uuid", len(args)))
	}
	if filter.Result != "" {
		args = append(args, filter.Result)
		clauses = append(clauses, fmt.Sprintf("result = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func auditSortClause(sort string) string {
	switch sort {
	case SortOldest:
		return "created_at ASC, id ASC"
	case "", SortNewest:
		return "created_at DESC, id DESC"
	default:
		return "created_at DESC, id DESC"
	}
}
