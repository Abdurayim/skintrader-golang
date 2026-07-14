package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// AdminLogRepo implements domain.AdminLogRepository using PostgreSQL.
type AdminLogRepo struct {
	pool *pgxpool.Pool
}

// NewAdminLogRepo creates a new AdminLogRepo.
func NewAdminLogRepo(pool *pgxpool.Pool) *AdminLogRepo {
	return &AdminLogRepo{pool: pool}
}

// ip_address is INET in PostgreSQL — cast to text so pgx can scan it into *string.
const adminLogColumns = `id, admin_id, action, target_type, target_id,
	details, previous_state, new_state,
	ip_address::text, user_agent, reason, metadata, created_at`

// scanAdminLog scans a single row into a domain.AdminLog.
func scanAdminLog(row pgx.Row) (*domain.AdminLog, error) {
	var l domain.AdminLog
	var detailsBytes, previousStateBytes, newStateBytes, metadataBytes []byte
	var ipAddress *string

	err := row.Scan(
		&l.ID, &l.AdminID, &l.Action, &l.TargetType, &l.TargetID,
		&detailsBytes, &previousStateBytes, &newStateBytes,
		&ipAddress, &l.UserAgent, &l.Reason, &metadataBytes, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if detailsBytes != nil {
		l.Details = detailsBytes
	}
	if previousStateBytes != nil {
		l.PreviousState = previousStateBytes
	}
	if newStateBytes != nil {
		l.NewState = newStateBytes
	}
	if metadataBytes != nil {
		l.Metadata = metadataBytes
	}
	l.IPAddress = ipAddress

	return &l, nil
}

func (r *AdminLogRepo) Create(ctx context.Context, log *domain.AdminLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	var details, previousState, newState, metadata []byte
	if log.Details != nil {
		details = []byte(log.Details)
	}
	if log.PreviousState != nil {
		previousState = []byte(log.PreviousState)
	}
	if log.NewState != nil {
		newState = []byte(log.NewState)
	}
	if log.Metadata != nil {
		metadata = []byte(log.Metadata)
	}

	query := `
		INSERT INTO admin_logs (
			id, admin_id, action, target_type, target_id,
			details, previous_state, new_state,
			ip_address, user_agent, reason, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		log.ID, log.AdminID, log.Action, log.TargetType, log.TargetID,
		details, previousState, newState,
		log.IPAddress, log.UserAgent, log.Reason, metadata,
	).Scan(&log.CreatedAt)
}

func (r *AdminLogRepo) FindByAdmin(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error) {
	// Count total logs for the admin.
	countQuery := `SELECT COUNT(*) FROM admin_logs WHERE admin_id = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, adminID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting admin logs: %w", err)
	}

	query := `SELECT ` + adminLogColumns + `
		FROM admin_logs
		WHERE admin_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, adminID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying admin logs by admin: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AdminLog
	for rows.Next() {
		l, err := scanAdminLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning admin log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating admin log rows: %w", err)
	}
	return logs, total, nil
}

func (r *AdminLogRepo) FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error) {
	// Count total logs for the target.
	countQuery := `SELECT COUNT(*) FROM admin_logs WHERE target_type = $1 AND target_id = $2`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, targetType, targetID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting target logs: %w", err)
	}

	query := `SELECT ` + adminLogColumns + `
		FROM admin_logs
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying admin logs by target: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AdminLog
	for rows.Next() {
		l, err := scanAdminLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning admin log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating admin log rows: %w", err)
	}
	return logs, total, nil
}

func (r *AdminLogRepo) FindByAction(ctx context.Context, action domain.AdminAction, limit, offset int) ([]*domain.AdminLog, int64, error) {
	// Count total logs for the action.
	countQuery := `SELECT COUNT(*) FROM admin_logs WHERE action = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, action).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting action logs: %w", err)
	}

	query := `SELECT ` + adminLogColumns + `
		FROM admin_logs
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, action, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying admin logs by action: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AdminLog
	for rows.Next() {
		l, err := scanAdminLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning admin log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating admin log rows: %w", err)
	}
	return logs, total, nil
}

func (r *AdminLogRepo) GetRecentLogs(ctx context.Context, limit int) ([]*domain.AdminLog, error) {
	query := `SELECT ` + adminLogColumns + `
		FROM admin_logs
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying recent admin logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AdminLog
	for rows.Next() {
		l, err := scanAdminLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning admin log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin log rows: %w", err)
	}
	return logs, nil
}

func (r *AdminLogRepo) ListWithFilters(ctx context.Context, filter domain.AdminLogListFilter) ([]*domain.AdminLog, int64, error) {
	// Build count query.
	countBuilder := psql.Select("COUNT(*)").From("admin_logs")
	countBuilder = r.applyAdminLogFilters(countBuilder, filter)

	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building count query: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting filtered admin logs: %w", err)
	}

	if total == 0 {
		return []*domain.AdminLog{}, 0, nil
	}

	// Data query.
	dataBuilder := psql.Select(
		"id", "admin_id", "action", "target_type", "target_id",
		"details", "previous_state", "new_state",
		"ip_address::text", "user_agent", "reason", "metadata", "created_at",
	).From("admin_logs")
	dataBuilder = r.applyAdminLogFilters(dataBuilder, filter)

	// Sorting.
	sortColumn := "created_at"
	switch filter.SortBy {
	case "created_at", "action", "target_type":
		sortColumn = filter.SortBy
	}

	sortOrder := "DESC"
	if filter.SortOrder == domain.SortOrderASC {
		sortOrder = "ASC"
	}
	dataBuilder = dataBuilder.OrderBy(sortColumn + " " + sortOrder)

	// Pagination.
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	dataBuilder = dataBuilder.Limit(uint64(limit)).Offset(uint64(offset))

	dataSQL, dataArgs, err := dataBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building data query: %w", err)
	}

	rows, err := r.pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying filtered admin logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AdminLog
	for rows.Next() {
		l, err := scanAdminLog(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning admin log row: %w", err)
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating admin log rows: %w", err)
	}
	return logs, total, nil
}

// applyAdminLogFilters applies common AdminLogListFilter conditions to a squirrel SelectBuilder.
func (r *AdminLogRepo) applyAdminLogFilters(builder sq.SelectBuilder, filter domain.AdminLogListFilter) sq.SelectBuilder {
	if filter.AdminID != nil {
		builder = builder.Where(sq.Eq{"admin_id": *filter.AdminID})
	}
	if filter.Action != nil {
		builder = builder.Where(sq.Eq{"action": *filter.Action})
	}
	if filter.TargetType != nil {
		builder = builder.Where(sq.Eq{"target_type": *filter.TargetType})
	}
	if filter.TargetID != nil {
		builder = builder.Where(sq.Eq{"target_id": *filter.TargetID})
	}
	return builder
}
