package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AdminLog represents an entry in the admin audit trail.
type AdminLog struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	AdminID       uuid.UUID       `json:"adminId" db:"admin_id"`
	Action        AdminAction     `json:"action" db:"action"`
	TargetType    string          `json:"targetType" db:"target_type"`
	TargetID      *uuid.UUID      `json:"targetId,omitempty" db:"target_id"`
	Details       json.RawMessage `json:"details,omitempty" db:"details"`
	PreviousState json.RawMessage `json:"previousState,omitempty" db:"previous_state"`
	NewState      json.RawMessage `json:"newState,omitempty" db:"new_state"`
	IPAddress     *string         `json:"ipAddress,omitempty" db:"ip_address"`
	UserAgent     *string         `json:"userAgent,omitempty" db:"user_agent"`
	Reason        *string         `json:"reason,omitempty" db:"reason"`
	Metadata      json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
}

// AdminLogListFilter provides filtering and pagination for admin log listing.
type AdminLogListFilter struct {
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	AdminID    *uuid.UUID   `json:"adminId,omitempty"`
	Action     *AdminAction `json:"action,omitempty"`
	TargetType *string      `json:"targetType,omitempty"`
	TargetID   *uuid.UUID   `json:"targetId,omitempty"`
	SortBy     string       `json:"sortBy,omitempty"`
	SortOrder  SortOrder    `json:"sortOrder,omitempty"`
}

// AdminLogRepository defines the interface for admin log data access.
type AdminLogRepository interface {
	Create(ctx context.Context, log *AdminLog) error
	FindByAdmin(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]*AdminLog, int64, error)
	FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, limit, offset int) ([]*AdminLog, int64, error)
	FindByAction(ctx context.Context, action AdminAction, limit, offset int) ([]*AdminLog, int64, error)
	GetRecentLogs(ctx context.Context, limit int) ([]*AdminLog, error)
	ListWithFilters(ctx context.Context, filter AdminLogListFilter) ([]*AdminLog, int64, error)
}
