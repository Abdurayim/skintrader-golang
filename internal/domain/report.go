package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Report represents a user-submitted report against a post or another user.
type Report struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	ReporterID           uuid.UUID      `json:"reporterId" db:"reporter_id"`
	ReportType           ReportType     `json:"reportType" db:"report_type"`
	TargetID             uuid.UUID      `json:"targetId" db:"target_id"`
	TargetModel          string         `json:"targetModel" db:"target_model"`
	Category             ReportCategory `json:"category" db:"category"`
	Description          string         `json:"description,omitempty" db:"description"`
	Status               ReportStatus   `json:"status" db:"status"`
	Priority             ReportPriority `json:"priority" db:"priority"`
	ReviewedBy           *uuid.UUID     `json:"reviewedBy,omitempty" db:"reviewed_by"`
	ReviewedAt           *time.Time     `json:"reviewedAt,omitempty" db:"reviewed_at"`
	ResolutionAction     *ReportAction  `json:"resolutionAction,omitempty" db:"resolution_action"`
	ResolutionNotes      *string        `json:"resolutionNotes,omitempty" db:"resolution_notes"`
	ResolutionAdminNotes *string        `json:"resolutionAdminNotes,omitempty" db:"resolution_admin_notes"`
	ResolvedAt           *time.Time     `json:"resolvedAt,omitempty" db:"resolved_at"`
	ReportHash           string         `json:"reportHash" db:"report_hash"`
	IPAddress            *string        `json:"ipAddress,omitempty" db:"ip_address"`
	UserAgent            *string        `json:"userAgent,omitempty" db:"user_agent"`
	ReportCount          int            `json:"reportCount" db:"report_count"`
	CreatedAt            time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time      `json:"updatedAt" db:"updated_at"`
}

// ReportStats holds aggregate statistics about reports.
type ReportStats struct {
	TotalPending     int64 `json:"totalPending"`
	TotalUnderReview int64 `json:"totalUnderReview"`
	TotalResolved    int64 `json:"totalResolved"`
	TotalDismissed   int64 `json:"totalDismissed"`
}

// ReportListFilter provides filtering and pagination for admin report listing.
type ReportListFilter struct {
	Page       int             `json:"page"`
	Limit      int             `json:"limit"`
	Search     string          `json:"search,omitempty"`
	Status     *ReportStatus   `json:"status,omitempty"`
	ReportType *ReportType     `json:"reportType,omitempty"`
	Category   *ReportCategory `json:"category,omitempty"`
	Priority   *ReportPriority `json:"priority,omitempty"`
	SortBy     string          `json:"sortBy,omitempty"`
	SortOrder  SortOrder       `json:"sortOrder,omitempty"`
}

// ReportRepository defines the interface for report data access.
type ReportRepository interface {
	Create(ctx context.Context, report *Report) error
	FindByID(ctx context.Context, id uuid.UUID) (*Report, error)
	FindByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]*Report, int64, error)
	FindByTarget(ctx context.Context, targetID uuid.UUID) ([]*Report, error)
	FindPending(ctx context.Context, limit, offset int) ([]*Report, int64, error)
	Update(ctx context.Context, report *Report) error
	CheckDuplicate(ctx context.Context, reportHash string) (bool, error)
	CountByTarget(ctx context.Context, targetID uuid.UUID) (int64, error)
	GetStats(ctx context.Context) (*ReportStats, error)
	ListWithFilters(ctx context.Context, filter ReportListFilter) ([]*Report, int64, error)
}
