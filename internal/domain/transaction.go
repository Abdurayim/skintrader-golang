package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Transaction represents a payment transaction for a subscription.
type Transaction struct {
	ID                    uuid.UUID         `json:"id" db:"id"`
	UserID                uuid.UUID         `json:"userId" db:"user_id"`
	SubscriptionID        *uuid.UUID        `json:"subscriptionId,omitempty" db:"subscription_id"`
	ExternalTransactionID *string           `json:"externalTransactionId,omitempty" db:"external_transaction_id"`
	Amount                float64           `json:"amount" db:"amount"`
	Currency              Currency          `json:"currency" db:"currency"`
	Status                TransactionStatus `json:"status" db:"status"`
	PaymentMethod         PaymentMethod     `json:"paymentMethod" db:"payment_method"`
	PaymentResponse       json.RawMessage   `json:"paymentResponse,omitempty" db:"payment_response"`
	WebhookReceived       bool              `json:"webhookReceived" db:"webhook_received"`
	WebhookReceivedAt     *time.Time        `json:"webhookReceivedAt,omitempty" db:"webhook_received_at"`
	IPAddress             *string           `json:"ipAddress,omitempty" db:"ip_address"`
	UserAgent             *string           `json:"userAgent,omitempty" db:"user_agent"`
	ErrorMessage          *string           `json:"errorMessage,omitempty" db:"error_message"`
	ErrorCode             *string           `json:"errorCode,omitempty" db:"error_code"`
	RefundedAt            *time.Time        `json:"refundedAt,omitempty" db:"refunded_at"`
	RefundReason          *string           `json:"refundReason,omitempty" db:"refund_reason"`
	RefundedBy            *uuid.UUID        `json:"refundedBy,omitempty" db:"refunded_by"`
	CreatedAt             time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt             time.Time         `json:"updatedAt" db:"updated_at"`
}

// RevenueStats holds aggregate revenue statistics.
type RevenueStats struct {
	TotalRevenue     float64 `json:"totalRevenue"`
	MonthlyRevenue   float64 `json:"monthlyRevenue"`
	TotalTransactions int64  `json:"totalTransactions"`
	SuccessRate      float64 `json:"successRate"`
}

// TransactionRepository defines the interface for transaction data access.
type TransactionRepository interface {
	Create(ctx context.Context, transaction *Transaction) error
	FindByID(ctx context.Context, id uuid.UUID) (*Transaction, error)
	FindByExternalID(ctx context.Context, externalID string) (*Transaction, error)
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Transaction, int64, error)
	Update(ctx context.Context, transaction *Transaction) error
	GetRevenueStats(ctx context.Context) (*RevenueStats, error)
}
