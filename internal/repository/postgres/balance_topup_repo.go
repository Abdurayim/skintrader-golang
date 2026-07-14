package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// BalanceTopupRepo implements domain.BalanceTopupRepository using PostgreSQL.
type BalanceTopupRepo struct {
	pool *pgxpool.Pool
}

// NewBalanceTopupRepo creates a new BalanceTopupRepo.
func NewBalanceTopupRepo(pool *pgxpool.Pool) *BalanceTopupRepo {
	return &BalanceTopupRepo{pool: pool}
}

const topupColumns = `id, user_id, amount, cheque_path, status, review_note, reviewed_by, reviewed_at, created_at`

func scanTopup(row pgx.Row) (*domain.BalanceTopup, error) {
	var t domain.BalanceTopup
	err := row.Scan(
		&t.ID, &t.UserID, &t.Amount, &t.ChequePath, &t.Status,
		&t.ReviewNote, &t.ReviewedBy, &t.ReviewedAt, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Create inserts a new pending top-up request.
func (r *BalanceTopupRepo) Create(ctx context.Context, topup *domain.BalanceTopup) error {
	if topup.ID == uuid.Nil {
		topup.ID = uuid.New()
	}
	if topup.Status == "" {
		topup.Status = domain.TopupStatusPending
	}
	return r.pool.QueryRow(ctx,
		`INSERT INTO balance_topups (id, user_id, amount, cheque_path, status)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		topup.ID, topup.UserID, topup.Amount, topup.ChequePath, topup.Status,
	).Scan(&topup.CreatedAt)
}

// FindByID retrieves a top-up by ID.
func (r *BalanceTopupRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.BalanceTopup, error) {
	query := `SELECT ` + topupColumns + ` FROM balance_topups WHERE id = $1`
	topup, err := scanTopup(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("top-up not found")
		}
		return nil, fmt.Errorf("finding top-up: %w", err)
	}
	return topup, nil
}

// FindByUser returns a user's top-up requests, newest first.
func (r *BalanceTopupRepo) FindByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.BalanceTopup, error) {
	query := `SELECT ` + topupColumns + ` FROM balance_topups
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying top-ups: %w", err)
	}
	defer rows.Close()

	topups := []*domain.BalanceTopup{}
	for rows.Next() {
		t, err := scanTopup(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning top-up: %w", err)
		}
		topups = append(topups, t)
	}
	return topups, rows.Err()
}

// ListWithFilters returns top-ups for admin review with pagination.
// Pending requests are listed oldest-first so admins process them in order;
// other listings are newest-first.
func (r *BalanceTopupRepo) ListWithFilters(ctx context.Context, filter domain.TopupListFilter) ([]*domain.BalanceTopup, int64, error) {
	where := "TRUE"
	args := []any{}
	argN := 1

	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *filter.Status)
		argN++
	}
	if filter.UserID != nil {
		where += fmt.Sprintf(" AND user_id = $%d", argN)
		args = append(args, *filter.UserID)
		argN++
	}

	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM balance_topups WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting top-ups: %w", err)
	}

	order := "created_at DESC"
	if filter.Status != nil && *filter.Status == domain.TopupStatusPending {
		order = "created_at ASC"
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`SELECT %s FROM balance_topups WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`,
		topupColumns, where, order, argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying top-ups: %w", err)
	}
	defer rows.Close()

	topups := []*domain.BalanceTopup{}
	for rows.Next() {
		t, err := scanTopup(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning top-up: %w", err)
		}
		topups = append(topups, t)
	}
	return topups, total, rows.Err()
}

// Approve marks a pending top-up approved and credits the user's balance
// in a single transaction.
func (r *BalanceTopupRepo) Approve(ctx context.Context, id uuid.UUID, adminID uuid.UUID, note string) (*domain.BalanceTopup, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin approve tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var noteArg *string
	if note != "" {
		noteArg = &note
	}

	topup, err := scanTopup(tx.QueryRow(ctx,
		`UPDATE balance_topups
		 SET status = 'approved', reviewed_by = $2, review_note = $3, reviewed_at = NOW()
		 WHERE id = $1 AND status = 'pending'
		 RETURNING `+topupColumns,
		id, adminID, noteArg,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("top-up not found or already reviewed")
		}
		return nil, fmt.Errorf("approving top-up: %w", err)
	}

	cmdTag, err := tx.Exec(ctx,
		`UPDATE users SET balance = balance + $2, updated_at = NOW() WHERE id = $1`,
		topup.UserID, topup.Amount,
	)
	if err != nil {
		return nil, fmt.Errorf("crediting balance: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return nil, fmt.Errorf("user not found for top-up")
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit approve tx: %w", err)
	}
	return topup, nil
}

// Reject marks a pending top-up rejected. The balance is not changed.
func (r *BalanceTopupRepo) Reject(ctx context.Context, id uuid.UUID, adminID uuid.UUID, note string) (*domain.BalanceTopup, error) {
	var noteArg *string
	if note != "" {
		noteArg = &note
	}

	topup, err := scanTopup(r.pool.QueryRow(ctx,
		`UPDATE balance_topups
		 SET status = 'rejected', reviewed_by = $2, review_note = $3, reviewed_at = NOW()
		 WHERE id = $1 AND status = 'pending'
		 RETURNING `+topupColumns,
		id, adminID, noteArg,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("top-up not found or already reviewed")
		}
		return nil, fmt.Errorf("rejecting top-up: %w", err)
	}
	return topup, nil
}

// GetRevenueStats aggregates revenue figures from verified top-ups.
func (r *BalanceTopupRepo) GetRevenueStats(ctx context.Context) (*domain.TopupRevenueStats, error) {
	stats := &domain.TopupRevenueStats{}

	err := r.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE status = 'approved'), 0),
			COUNT(*) FILTER (WHERE status = 'approved'),
			COALESCE(SUM(amount) FILTER (WHERE status = 'approved' AND reviewed_at >= CURRENT_DATE), 0),
			COALESCE(SUM(amount) FILTER (WHERE status = 'approved' AND reviewed_at >= date_trunc('month', CURRENT_DATE)), 0),
			COALESCE(SUM(amount) FILTER (WHERE status = 'pending'), 0),
			COUNT(*) FILTER (WHERE status = 'pending')
		FROM balance_topups`,
	).Scan(
		&stats.TotalApproved, &stats.ApprovedCount,
		&stats.TodayApproved, &stats.MonthApproved,
		&stats.PendingAmount, &stats.PendingCount,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregating top-up revenue: %w", err)
	}

	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(balance), 0) FROM users`,
	).Scan(&stats.OutstandingBalance); err != nil {
		return nil, fmt.Errorf("summing user balances: %w", err)
	}

	stats.ConsumedRevenue = stats.TotalApproved - stats.OutstandingBalance
	if stats.ConsumedRevenue < 0 {
		stats.ConsumedRevenue = 0
	}
	return stats, nil
}
