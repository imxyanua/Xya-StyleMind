package audit

import "time"

const (
	ResultSuccess = "success"
	ResultFailed  = "failed"

	SortNewest = "newest"
	SortOldest = "oldest"
)

type Log struct {
	ID           string         `json:"id"`
	ActorUserID  string         `json:"actor_user_id,omitempty"`
	ActorRole    string         `json:"actor_role"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id,omitempty"`
	Result       string         `json:"result"`
	Metadata     map[string]any `json:"metadata"`
	IP           string         `json:"ip,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	RequestID    string         `json:"request_id,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type CreateLogParams struct {
	ActorUserID  string
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	Metadata     map[string]any
	IP           string
	UserAgent    string
	RequestID    string
}

type ListFilter struct {
	Action       string
	ResourceType string
	ActorUserID  string
	Result       string
	From         *time.Time
	To           *time.Time
	Sort         string
}
