package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// PostCostUZS is the price of publishing one post, in Uzbek so'm.
const PostCostUZS int64 = 2000

// TopupStatus represents the review state of a balance top-up request.
type TopupStatus string

const (
	TopupStatusPending  TopupStatus = "pending"
	TopupStatusApproved TopupStatus = "approved"
	TopupStatusRejected TopupStatus = "rejected"
)

// BalanceTopup is a manual balance top-up request: the user transfers money
// to the platform card and uploads a cheque/receipt screenshot as proof.
type BalanceTopup struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	UserID     uuid.UUID   `json:"userId" db:"user_id"`
	Amount     int64       `json:"amount" db:"amount"`
	ChequePath string      `json:"chequePath" db:"cheque_path"`
	Status     TopupStatus `json:"status" db:"status"`
	ReviewNote *string     `json:"reviewNote,omitempty" db:"review_note"`
	ReviewedBy *uuid.UUID  `json:"reviewedBy,omitempty" db:"reviewed_by"`
	ReviewedAt *time.Time  `json:"reviewedAt,omitempty" db:"reviewed_at"`
	CreatedAt  time.Time   `json:"createdAt" db:"created_at"`

	// Computed (not persisted): user info for the admin review UI
	User *TopupUserInfo `json:"user,omitempty" db:"-"`
}

// TopupUserInfo is the requester's public info shown in the admin review UI.
type TopupUserInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Email       string    `json:"email"`
	Balance     int64     `json:"balance"`
}

// TopupListFilter provides filtering and pagination for admin top-up listing.
type TopupListFilter struct {
	Page   int
	Limit  int
	Status *TopupStatus
	UserID *uuid.UUID
}

// BalanceTopupRepository defines the interface for balance top-up data access.
type BalanceTopupRepository interface {
	Create(ctx context.Context, topup *BalanceTopup) error
	FindByID(ctx context.Context, id uuid.UUID) (*BalanceTopup, error)
	FindByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*BalanceTopup, error)
	ListWithFilters(ctx context.Context, filter TopupListFilter) ([]*BalanceTopup, int64, error)
	// Approve marks a pending top-up approved and credits the user's balance
	// atomically. Returns the updated top-up.
	Approve(ctx context.Context, id uuid.UUID, adminID uuid.UUID, note string) (*BalanceTopup, error)
	// Reject marks a pending top-up rejected without touching the balance.
	Reject(ctx context.Context, id uuid.UUID, adminID uuid.UUID, note string) (*BalanceTopup, error)
}
