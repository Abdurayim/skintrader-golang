package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// NotificationType identifies the event a notification describes.
// The frontend maps each type to a translated message template.
type NotificationType string

const (
	NotificationTopupApproved NotificationType = "topup_approved"
	NotificationTopupRejected NotificationType = "topup_rejected"
	NotificationKYCApproved   NotificationType = "kyc_approved"
	NotificationKYCRejected   NotificationType = "kyc_rejected"
	NotificationPostPublished NotificationType = "post_published"
	NotificationPostRemoved   NotificationType = "post_removed"
)

// Notification is an on-site notification for a user.
type Notification struct {
	ID        uuid.UUID        `json:"id" db:"id"`
	UserID    uuid.UUID        `json:"userId" db:"user_id"`
	Type      NotificationType `json:"type" db:"type"`
	Data      json.RawMessage  `json:"data,omitempty" db:"data"`
	ReadAt    *time.Time       `json:"readAt,omitempty" db:"read_at"`
	CreatedAt time.Time        `json:"createdAt" db:"created_at"`

	// Computed (not persisted)
	Read bool `json:"read" db:"-"`
}

// NotificationRepository defines the interface for notification data access.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Notification, int64, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
}

// TopupRevenueStats aggregates platform revenue from verified balance top-ups.
// All amounts are in UZS.
type TopupRevenueStats struct {
	TotalApproved      int64 `json:"totalApproved"`      // all-time verified top-ups
	ApprovedCount      int64 `json:"approvedCount"`      // number of approved requests
	TodayApproved      int64 `json:"todayApproved"`      // approved today
	MonthApproved      int64 `json:"monthApproved"`      // approved in the current month
	PendingAmount      int64 `json:"pendingAmount"`      // awaiting review
	PendingCount       int64 `json:"pendingCount"`       //
	OutstandingBalance int64 `json:"outstandingBalance"` // sum of all user balances (unspent)
	ConsumedRevenue    int64 `json:"consumedRevenue"`    // TotalApproved - OutstandingBalance (spent on posts)
}
