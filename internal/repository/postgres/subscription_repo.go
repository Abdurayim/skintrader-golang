package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// SubscriptionRepo implements domain.SubscriptionRepository using PostgreSQL.
type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

// NewSubscriptionRepo creates a new SubscriptionRepo.
func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

const subscriptionColumns = `id, user_id, status, plan, start_date, end_date, auto_renew,
	last_payment_id, grace_period_started, cancelled_at, cancel_reason, created_at, updated_at`

// scanSubscription scans a single row into a domain.Subscription.
func scanSubscription(row pgx.Row) (*domain.Subscription, error) {
	var s domain.Subscription
	err := row.Scan(
		&s.ID, &s.UserID, &s.Status, &s.Plan, &s.StartDate, &s.EndDate, &s.AutoRenew,
		&s.LastPaymentID, &s.GracePeriodStarted, &s.CancelledAt, &s.CancelReason,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepo) Create(ctx context.Context, subscription *domain.Subscription) error {
	if subscription.ID == uuid.Nil {
		subscription.ID = uuid.New()
	}

	query := `
		INSERT INTO subscriptions (
			id, user_id, status, plan, start_date, end_date, auto_renew,
			last_payment_id, grace_period_started, cancelled_at, cancel_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		subscription.ID, subscription.UserID, subscription.Status, subscription.Plan,
		subscription.StartDate, subscription.EndDate, subscription.AutoRenew,
		subscription.LastPaymentID, subscription.GracePeriodStarted,
		subscription.CancelledAt, subscription.CancelReason,
	).Scan(&subscription.CreatedAt, &subscription.UpdatedAt)
}

func (r *SubscriptionRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions WHERE id = $1`
	sub, err := scanSubscription(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		return nil, fmt.Errorf("finding subscription by id: %w", err)
	}
	return sub, nil
}

func (r *SubscriptionRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying subscriptions by user: %w", err)
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning subscription row: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating subscription rows: %w", err)
	}
	return subs, nil
}

func (r *SubscriptionRepo) FindActive(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + ` FROM subscriptions WHERE user_id = $1 AND status = 'active' LIMIT 1`
	sub, err := scanSubscription(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("active subscription not found")
		}
		return nil, fmt.Errorf("finding active subscription: %w", err)
	}
	return sub, nil
}

func (r *SubscriptionRepo) Update(ctx context.Context, subscription *domain.Subscription) error {
	query := `
		UPDATE subscriptions SET
			status = $2, plan = $3, start_date = $4, end_date = $5, auto_renew = $6,
			last_payment_id = $7, grace_period_started = $8, cancelled_at = $9, cancel_reason = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		subscription.ID, subscription.Status, subscription.Plan,
		subscription.StartDate, subscription.EndDate, subscription.AutoRenew,
		subscription.LastPaymentID, subscription.GracePeriodStarted,
		subscription.CancelledAt, subscription.CancelReason,
	).Scan(&subscription.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("subscription not found")
		}
		return fmt.Errorf("updating subscription: %w", err)
	}
	return nil
}

func (r *SubscriptionRepo) FindExpired(ctx context.Context) ([]*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + `
		FROM subscriptions
		WHERE status = 'active' AND end_date < NOW()
		ORDER BY end_date ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying expired subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning expired subscription row: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expired subscription rows: %w", err)
	}
	return subs, nil
}

func (r *SubscriptionRepo) FindExpiredGracePeriod(ctx context.Context) ([]*domain.Subscription, error) {
	query := `SELECT ` + subscriptionColumns + `
		FROM subscriptions
		WHERE status = 'grace_period' AND grace_period_started < NOW() - INTERVAL '7 days'
		ORDER BY grace_period_started ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying expired grace period subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*domain.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning expired grace period subscription row: %w", err)
		}
		subs = append(subs, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating expired grace period subscription rows: %w", err)
	}
	return subs, nil
}

func (r *SubscriptionRepo) GetStats(ctx context.Context) (*domain.SubscriptionStats, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'active' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'expired' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'grace_period' THEN 1 ELSE 0 END), 0)
		FROM subscriptions`

	var stats domain.SubscriptionStats
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalActive,
		&stats.TotalExpired,
		&stats.TotalCancelled,
		&stats.TotalPending,
		&stats.TotalGrace,
	)
	if err != nil {
		return nil, fmt.Errorf("getting subscription stats: %w", err)
	}
	return &stats, nil
}
