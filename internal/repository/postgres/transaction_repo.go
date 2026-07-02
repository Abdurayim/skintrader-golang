package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// TransactionRepo implements domain.TransactionRepository using PostgreSQL.
type TransactionRepo struct {
	pool *pgxpool.Pool
}

// NewTransactionRepo creates a new TransactionRepo.
func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

const transactionColumns = `id, user_id, subscription_id, external_transaction_id,
	amount, currency, status, payment_method, payment_response,
	webhook_received, webhook_received_at,
	ip_address, user_agent, error_message, error_code,
	refunded_at, refund_reason, refunded_by,
	created_at, updated_at`

// scanTransaction scans a single row into a domain.Transaction.
func scanTransaction(row pgx.Row) (*domain.Transaction, error) {
	var t domain.Transaction
	var paymentResponseBytes []byte
	var ipAddress *string

	err := row.Scan(
		&t.ID, &t.UserID, &t.SubscriptionID, &t.ExternalTransactionID,
		&t.Amount, &t.Currency, &t.Status, &t.PaymentMethod, &paymentResponseBytes,
		&t.WebhookReceived, &t.WebhookReceivedAt,
		&ipAddress, &t.UserAgent, &t.ErrorMessage, &t.ErrorCode,
		&t.RefundedAt, &t.RefundReason, &t.RefundedBy,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if paymentResponseBytes != nil {
		t.PaymentResponse = paymentResponseBytes
	}
	t.IPAddress = ipAddress

	return &t, nil
}

func (r *TransactionRepo) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.ID == uuid.Nil {
		transaction.ID = uuid.New()
	}

	query := `
		INSERT INTO transactions (
			id, user_id, subscription_id, external_transaction_id,
			amount, currency, status, payment_method, payment_response,
			webhook_received, webhook_received_at,
			ip_address, user_agent, error_message, error_code,
			refunded_at, refund_reason, refunded_by
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11,
			$12, $13, $14, $15,
			$16, $17, $18
		)
		RETURNING created_at, updated_at`

	var paymentResponse []byte
	if transaction.PaymentResponse != nil {
		paymentResponse = []byte(transaction.PaymentResponse)
	}

	return r.pool.QueryRow(ctx, query,
		transaction.ID, transaction.UserID, transaction.SubscriptionID, transaction.ExternalTransactionID,
		transaction.Amount, transaction.Currency, transaction.Status, transaction.PaymentMethod, paymentResponse,
		transaction.WebhookReceived, transaction.WebhookReceivedAt,
		transaction.IPAddress, transaction.UserAgent, transaction.ErrorMessage, transaction.ErrorCode,
		transaction.RefundedAt, transaction.RefundReason, transaction.RefundedBy,
	).Scan(&transaction.CreatedAt, &transaction.UpdatedAt)
}

func (r *TransactionRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `SELECT ` + transactionColumns + ` FROM transactions WHERE id = $1`
	tx, err := scanTransaction(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("finding transaction by id: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepo) FindByExternalID(ctx context.Context, externalID string) (*domain.Transaction, error) {
	query := `SELECT ` + transactionColumns + ` FROM transactions WHERE external_transaction_id = $1`
	tx, err := scanTransaction(r.pool.QueryRow(ctx, query, externalID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("finding transaction by external id: %w", err)
	}
	return tx, nil
}

func (r *TransactionRepo) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, int64, error) {
	// Count total transactions for the user.
	countQuery := `SELECT COUNT(*) FROM transactions WHERE user_id = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting user transactions: %w", err)
	}

	query := `SELECT ` + transactionColumns + `
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying user transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		tx, err := scanTransaction(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning transaction row: %w", err)
		}
		transactions = append(transactions, tx)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating transaction rows: %w", err)
	}
	return transactions, total, nil
}

func (r *TransactionRepo) Update(ctx context.Context, transaction *domain.Transaction) error {
	var paymentResponse []byte
	if transaction.PaymentResponse != nil {
		paymentResponse = []byte(transaction.PaymentResponse)
	}

	query := `
		UPDATE transactions SET
			subscription_id = $2, external_transaction_id = $3,
			amount = $4, currency = $5, status = $6, payment_method = $7, payment_response = $8,
			webhook_received = $9, webhook_received_at = $10,
			ip_address = $11, user_agent = $12, error_message = $13, error_code = $14,
			refunded_at = $15, refund_reason = $16, refunded_by = $17
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		transaction.ID, transaction.SubscriptionID, transaction.ExternalTransactionID,
		transaction.Amount, transaction.Currency, transaction.Status, transaction.PaymentMethod, paymentResponse,
		transaction.WebhookReceived, transaction.WebhookReceivedAt,
		transaction.IPAddress, transaction.UserAgent, transaction.ErrorMessage, transaction.ErrorCode,
		transaction.RefundedAt, transaction.RefundReason, transaction.RefundedBy,
	).Scan(&transaction.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("transaction not found")
		}
		return fmt.Errorf("updating transaction: %w", err)
	}
	return nil
}

func (r *TransactionRepo) GetRevenueStats(ctx context.Context) (*domain.RevenueStats, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'completed' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' AND created_at >= date_trunc('month', NOW()) THEN amount ELSE 0 END), 0),
			COUNT(*),
			CASE WHEN COUNT(*) > 0
				THEN COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END)::float / COUNT(*)::float * 100, 0)
				ELSE 0
			END
		FROM transactions`

	var stats domain.RevenueStats
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalRevenue,
		&stats.MonthlyRevenue,
		&stats.TotalTransactions,
		&stats.SuccessRate,
	)
	if err != nil {
		return nil, fmt.Errorf("getting revenue stats: %w", err)
	}
	return &stats, nil
}
