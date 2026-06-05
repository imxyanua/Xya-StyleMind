package audit

import (
	"context"
	"log"
	"strings"

	"stylemind/internal/errs"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Store interface {
	Create(ctx context.Context, params CreateLogParams) (*Log, error)
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]Log, int64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, filter ListFilter, limit, offset int) ([]Log, int64, error) {
	filter.Action = strings.TrimSpace(filter.Action)
	filter.ResourceType = strings.TrimSpace(filter.ResourceType)
	filter.ActorUserID = strings.TrimSpace(filter.ActorUserID)
	filter.Result = strings.TrimSpace(filter.Result)
	filter.Sort = strings.TrimSpace(filter.Sort)
	if filter.ActorUserID != "" {
		if _, err := uuid.Parse(filter.ActorUserID); err != nil {
			return nil, 0, errs.ErrInvalidID
		}
	}
	if filter.Result != "" && filter.Result != ResultSuccess && filter.Result != ResultFailed {
		return nil, 0, errs.ErrValidationFailed
	}
	if filter.Sort != "" && filter.Sort != SortNewest && filter.Sort != SortOldest {
		return nil, 0, errs.ErrInvalidSort
	}
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, 0, errs.ErrValidationFailed
	}
	return s.store.List(ctx, filter, limit, offset)
}

func (s *Service) RecordAdmin(c *gin.Context, action, resourceType, resourceID, result string, metadata map[string]any) {
	if s == nil || s.store == nil || c == nil {
		return
	}
	if result != ResultSuccess && result != ResultFailed {
		result = ResultFailed
	}
	actorRole := c.GetString("user_role")
	if actorRole == "" {
		actorRole = "admin"
	}
	_, err := s.store.Create(c.Request.Context(), CreateLogParams{
		ActorUserID:  c.GetString("user_id"),
		ActorRole:    actorRole,
		Action:       strings.TrimSpace(action),
		ResourceType: strings.TrimSpace(resourceType),
		ResourceID:   strings.TrimSpace(resourceID),
		Result:       result,
		Metadata:     sanitizeMetadata(metadata),
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    c.GetString("request_id"),
	})
	if err != nil {
		log.Printf("audit persistence failed: %v", err)
	}
}

func sanitizeMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" || isSensitiveMetadataKey(cleanKey) {
			continue
		}
		switch v := value.(type) {
		case string:
			if v != "" {
				out[cleanKey] = v
			}
		case bool, int, int64, float64, float32:
			out[cleanKey] = v
		case nil:
			continue
		default:
			out[cleanKey] = v
		}
	}
	return out
}

func isSensitiveMetadataKey(key string) bool {
	lower := strings.ToLower(key)
	for _, needle := range []string{"password", "token", "secret", "authorization", "jwt", "credential"} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}
