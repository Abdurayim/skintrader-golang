package postgres

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// ReportRepo implements domain.ReportRepository using PostgreSQL.
type ReportRepo struct {
	pool *pgxpool.Pool
}

// NewReportRepo creates a new ReportRepo.
func NewReportRepo(pool *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{pool: pool}
}

// ip_address is INET in PostgreSQL — cast to text so pgx can scan it into *string.
const reportColumns = `id, reporter_id, report_type, target_id, target_model, category,
	description, status, priority, reviewed_by, reviewed_at,
	resolution_action, resolution_notes, resolution_admin_notes, resolved_at,
	report_hash, ip_address::text, user_agent, report_count, created_at, updated_at`

// scanReport scans a single row into a domain.Report.
func scanReport(row pgx.Row) (*domain.Report, error) {
	var r domain.Report
	var ipAddress *string

	err := row.Scan(
		&r.ID, &r.ReporterID, &r.ReportType, &r.TargetID, &r.TargetModel, &r.Category,
		&r.Description, &r.Status, &r.Priority, &r.ReviewedBy, &r.ReviewedAt,
		&r.ResolutionAction, &r.ResolutionNotes, &r.ResolutionAdminNotes, &r.ResolvedAt,
		&r.ReportHash, &ipAddress, &r.UserAgent, &r.ReportCount, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	r.IPAddress = ipAddress
	return &r, nil
}

func (r *ReportRepo) Create(ctx context.Context, report *domain.Report) error {
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}

	query := `
		INSERT INTO reports (
			id, reporter_id, report_type, target_id, target_model, category,
			description, status, priority, reviewed_by, reviewed_at,
			resolution_action, resolution_notes, resolution_admin_notes, resolved_at,
			report_hash, ip_address, user_agent, report_count
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19
		)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		report.ID, report.ReporterID, report.ReportType, report.TargetID, report.TargetModel, report.Category,
		report.Description, report.Status, report.Priority, report.ReviewedBy, report.ReviewedAt,
		report.ResolutionAction, report.ResolutionNotes, report.ResolutionAdminNotes, report.ResolvedAt,
		report.ReportHash, report.IPAddress, report.UserAgent, report.ReportCount,
	).Scan(&report.CreatedAt, &report.UpdatedAt)
}

func (r *ReportRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	query := `SELECT ` + reportColumns + ` FROM reports WHERE id = $1`
	report, err := scanReport(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("report not found")
		}
		return nil, fmt.Errorf("finding report by id: %w", err)
	}
	return report, nil
}

func (r *ReportRepo) FindByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]*domain.Report, int64, error) {
	// Count total reports by reporter.
	countQuery := `SELECT COUNT(*) FROM reports WHERE reporter_id = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, reporterID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting reporter reports: %w", err)
	}

	query := `SELECT ` + reportColumns + `
		FROM reports
		WHERE reporter_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, reporterID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying reporter reports: %w", err)
	}
	defer rows.Close()

	var reports []*domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating report rows: %w", err)
	}
	return reports, total, nil
}

func (r *ReportRepo) FindByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.Report, error) {
	query := `SELECT ` + reportColumns + `
		FROM reports
		WHERE target_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, targetID)
	if err != nil {
		return nil, fmt.Errorf("querying target reports: %w", err)
	}
	defer rows.Close()

	var reports []*domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating report rows: %w", err)
	}
	return reports, nil
}

func (r *ReportRepo) FindPending(ctx context.Context, limit, offset int) ([]*domain.Report, int64, error) {
	countQuery := `SELECT COUNT(*) FROM reports WHERE status = 'pending'`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting pending reports: %w", err)
	}

	query := `SELECT ` + reportColumns + `
		FROM reports
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying pending reports: %w", err)
	}
	defer rows.Close()

	var reports []*domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating report rows: %w", err)
	}
	return reports, total, nil
}

func (r *ReportRepo) Update(ctx context.Context, report *domain.Report) error {
	query := `
		UPDATE reports SET
			status = $2, priority = $3, reviewed_by = $4, reviewed_at = $5,
			resolution_action = $6, resolution_notes = $7, resolution_admin_notes = $8, resolved_at = $9,
			report_count = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		report.ID, report.Status, report.Priority, report.ReviewedBy, report.ReviewedAt,
		report.ResolutionAction, report.ResolutionNotes, report.ResolutionAdminNotes, report.ResolvedAt,
		report.ReportCount,
	).Scan(&report.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("report not found")
		}
		return fmt.Errorf("updating report: %w", err)
	}
	return nil
}

func (r *ReportRepo) CheckDuplicate(ctx context.Context, reportHash string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM reports WHERE report_hash = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, reportHash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking duplicate report: %w", err)
	}
	return exists, nil
}

func (r *ReportRepo) CountByTarget(ctx context.Context, targetID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM reports WHERE target_id = $1`
	var count int64
	err := r.pool.QueryRow(ctx, query, targetID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting reports by target: %w", err)
	}
	return count, nil
}

func (r *ReportRepo) GetStats(ctx context.Context) (*domain.ReportStats, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'under_review' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'dismissed' THEN 1 ELSE 0 END), 0)
		FROM reports`

	var stats domain.ReportStats
	err := r.pool.QueryRow(ctx, query).Scan(
		&stats.TotalPending,
		&stats.TotalUnderReview,
		&stats.TotalResolved,
		&stats.TotalDismissed,
	)
	if err != nil {
		return nil, fmt.Errorf("getting report stats: %w", err)
	}
	return &stats, nil
}

func (r *ReportRepo) ListWithFilters(ctx context.Context, filter domain.ReportListFilter) ([]*domain.Report, int64, error) {
	// Build WHERE clause using squirrel.
	countBuilder := psql.Select("COUNT(*)").From("reports")
	countBuilder = r.applyReportFilters(countBuilder, filter)

	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building count query: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting filtered reports: %w", err)
	}

	if total == 0 {
		return []*domain.Report{}, 0, nil
	}

	// Data query.
	dataBuilder := psql.Select(
		"id", "reporter_id", "report_type", "target_id", "target_model", "category",
		"description", "status", "priority", "reviewed_by", "reviewed_at",
		"resolution_action", "resolution_notes", "resolution_admin_notes", "resolved_at",
		"report_hash", "ip_address::text", "user_agent", "report_count", "created_at", "updated_at",
	).From("reports")
	dataBuilder = r.applyReportFilters(dataBuilder, filter)

	// Sorting.
	sortColumn := "created_at"
	switch filter.SortBy {
	case "created_at", "updated_at", "priority", "status", "report_type", "category":
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
		return nil, 0, fmt.Errorf("querying filtered reports: %w", err)
	}
	defer rows.Close()

	var reports []*domain.Report
	for rows.Next() {
		report, err := scanReport(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning report row: %w", err)
		}
		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating report rows: %w", err)
	}
	return reports, total, nil
}

// applyReportFilters applies common ReportListFilter conditions to a squirrel SelectBuilder.
func (r *ReportRepo) applyReportFilters(builder sq.SelectBuilder, filter domain.ReportListFilter) sq.SelectBuilder {
	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		builder = builder.Where(
			sq.Expr("LOWER(description) LIKE ?", searchPattern),
		)
	}
	if filter.Status != nil {
		builder = builder.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.ReportType != nil {
		builder = builder.Where(sq.Eq{"report_type": *filter.ReportType})
	}
	if filter.Category != nil {
		builder = builder.Where(sq.Eq{"category": *filter.Category})
	}
	if filter.Priority != nil {
		builder = builder.Where(sq.Eq{"priority": *filter.Priority})
	}
	return builder
}
