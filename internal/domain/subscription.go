package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Subscription represents a user's subscription to a premium plan.
type Subscription struct {
	ID                 uuid.UUID          `json:"id" db:"id"`
	UserID             uuid.UUID          `json:"userId" db:"user_id"`
	Status             SubscriptionStatus `json:"status" db:"status"`
	Plan               SubscriptionPlan   `json:"plan" db:"plan"`
	StartDate          *time.Time         `json:"startDate,omitempty" db:"start_date"`
	EndDate            *time.Time         `json:"endDate,omitempty" db:"end_date"`
	AutoRenew          bool               `json:"autoRenew" db:"auto_renew"`
	LastPaymentID      *uuid.UUID         `json:"lastPaymentId,omitempty" db:"last_payment_id"`
	GracePeriodStarted *time.Time         `json:"gracePeriodStarted,omitempty" db:"grace_period_started"`
	CancelledAt        *time.Time         `json:"cancelledAt,omitempty" db:"cancelled_at"`
	CancelReason       *string            `json:"cancelReason,omitempty" db:"cancel_reason"`
	CreatedAt          time.Time          `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time          `json:"updatedAt" db:"updated_at"`
}

// SubscriptionStats holds aggregate statistics about subscriptions.
type SubscriptionStats struct {
	TotalActive    int64 `json:"totalActive"`
	TotalExpired   int64 `json:"totalExpired"`
	TotalCancelled int64 `json:"totalCancelled"`
	TotalPending   int64 `json:"totalPending"`
	TotalGrace     int64 `json:"totalGrace"`
}

// SubscriptionRepository defines the interface for subscription data access.
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *Subscription) error
	FindByID(ctx context.Context, id uuid.UUID) (*Subscription, error)
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*Subscription, error)
	FindActive(ctx context.Context, userID uuid.UUID) (*Subscription, error)
	Update(ctx context.Context, subscription *Subscription) error
	FindExpired(ctx context.Context) ([]*Subscription, error)
	FindExpiredGracePeriod(ctx context.Context) ([]*Subscription, error)
	GetStats(ctx context.Context) (*SubscriptionStats, error)
}
