package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// AdminRepo implements domain.AdminRepository using PostgreSQL.
type AdminRepo struct {
	pool *pgxpool.Pool
}

// NewAdminRepo creates a new AdminRepo.
func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{pool: pool}
}

// last_login_ip is INET in PostgreSQL — cast to text so pgx can scan it into *string.
const adminColumns = `id, email, password_hash, name, role, permissions, is_active,
	last_login_at, last_login_ip::text, password_reset_token, password_reset_expires,
	created_by, updated_by, created_at, updated_at`

// scanAdmin scans a single row into a domain.Admin.
func scanAdmin(row pgx.Row) (*domain.Admin, error) {
	var a domain.Admin
	var permissions []string
	var lastLoginIP *string

	err := row.Scan(
		&a.ID, &a.Email, &a.PasswordHash, &a.Name, &a.Role, &permissions, &a.IsActive,
		&a.LastLoginAt, &lastLoginIP, &a.PasswordResetToken, &a.PasswordResetExpires,
		&a.CreatedBy, &a.UpdatedBy, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Convert []string to []AdminPermission.
	a.Permissions = make([]domain.AdminPermission, len(permissions))
	for i, p := range permissions {
		a.Permissions[i] = domain.AdminPermission(p)
	}
	a.LastLoginIP = lastLoginIP

	return &a, nil
}

// permissionsToStrings converts []AdminPermission to []string for database storage.
func permissionsToStrings(perms []domain.AdminPermission) []string {
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = string(p)
	}
	return result
}

func (r *AdminRepo) Create(ctx context.Context, admin *domain.Admin) error {
	if admin.ID == uuid.Nil {
		admin.ID = uuid.New()
	}

	query := `
		INSERT INTO admins (
			id, email, password_hash, name, role, permissions, is_active,
			last_login_at, last_login_ip, password_reset_token, password_reset_expires,
			created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		admin.ID, admin.Email, admin.PasswordHash, admin.Name, admin.Role,
		permissionsToStrings(admin.Permissions), admin.IsActive,
		admin.LastLoginAt, admin.LastLoginIP,
		admin.PasswordResetToken, admin.PasswordResetExpires,
		admin.CreatedBy, admin.UpdatedBy,
	).Scan(&admin.CreatedAt, &admin.UpdatedAt)
}

func (r *AdminRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Admin, error) {
	query := `SELECT ` + adminColumns + ` FROM admins WHERE id = $1`
	admin, err := scanAdmin(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		return nil, fmt.Errorf("finding admin by id: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) FindByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	query := `SELECT ` + adminColumns + ` FROM admins WHERE email = $1`
	admin, err := scanAdmin(r.pool.QueryRow(ctx, query, email))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		return nil, fmt.Errorf("finding admin by email: %w", err)
	}
	return admin, nil
}

func (r *AdminRepo) FindActive(ctx context.Context) ([]*domain.Admin, error) {
	query := `SELECT ` + adminColumns + ` FROM admins WHERE is_active = true ORDER BY name`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying active admins: %w", err)
	}
	defer rows.Close()

	var admins []*domain.Admin
	for rows.Next() {
		admin, err := scanAdmin(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning admin row: %w", err)
		}
		admins = append(admins, admin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin rows: %w", err)
	}
	return admins, nil
}

func (r *AdminRepo) FindByRole(ctx context.Context, role domain.AdminRole) ([]*domain.Admin, error) {
	query := `SELECT ` + adminColumns + ` FROM admins WHERE role = $1 ORDER BY name`
	rows, err := r.pool.Query(ctx, query, role)
	if err != nil {
		return nil, fmt.Errorf("querying admins by role: %w", err)
	}
	defer rows.Close()

	var admins []*domain.Admin
	for rows.Next() {
		admin, err := scanAdmin(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning admin row: %w", err)
		}
		admins = append(admins, admin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating admin rows: %w", err)
	}
	return admins, nil
}

func (r *AdminRepo) Update(ctx context.Context, admin *domain.Admin) error {
	query := `
		UPDATE admins SET
			email = $2, password_hash = $3, name = $4, role = $5, permissions = $6,
			is_active = $7, password_reset_token = $8, password_reset_expires = $9,
			updated_by = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		admin.ID, admin.Email, admin.PasswordHash, admin.Name, admin.Role,
		permissionsToStrings(admin.Permissions), admin.IsActive,
		admin.PasswordResetToken, admin.PasswordResetExpires,
		admin.UpdatedBy,
	).Scan(&admin.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("admin not found")
		}
		return fmt.Errorf("updating admin: %w", err)
	}
	return nil
}

func (r *AdminRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	query := `
		UPDATE admins SET
			last_login_at = NOW(),
			last_login_ip = $2
		WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id, ip)
	if err != nil {
		return fmt.Errorf("updating last login: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("admin not found")
	}
	return nil
}
