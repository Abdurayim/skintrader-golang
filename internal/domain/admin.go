package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Admin represents an administrator account.
type Admin struct {
	ID                   uuid.UUID         `json:"id" db:"id"`
	Email                string            `json:"email" db:"email"`
	PasswordHash         string            `json:"-" db:"password_hash"`
	Name                 string            `json:"name" db:"name"`
	Role                 AdminRole         `json:"role" db:"role"`
	Permissions          []AdminPermission `json:"permissions" db:"permissions"`
	IsActive             bool              `json:"isActive" db:"is_active"`
	LastLoginAt          *time.Time        `json:"lastLoginAt,omitempty" db:"last_login_at"`
	LastLoginIP          *string           `json:"lastLoginIp,omitempty" db:"last_login_ip"`
	PasswordResetToken   *string           `json:"-" db:"password_reset_token"`
	PasswordResetExpires *time.Time        `json:"-" db:"password_reset_expires"`
	CreatedBy            *uuid.UUID        `json:"createdBy,omitempty" db:"created_by"`
	UpdatedBy            *uuid.UUID        `json:"updatedBy,omitempty" db:"updated_by"`
	CreatedAt            time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time         `json:"updatedAt" db:"updated_at"`
}

// HasPermission checks if the admin has the given permission.
func (a *Admin) HasPermission(perm string) bool {
	if a.Role == AdminRoleSuperAdmin {
		return true
	}
	for _, p := range a.Permissions {
		if string(p) == perm {
			return true
		}
	}
	return false
}

// IsSuperAdmin checks if the admin is a super admin.
func (a *Admin) IsSuperAdmin() bool {
	return a.Role == AdminRoleSuperAdmin
}

// AdminRepository defines the interface for admin data access.
type AdminRepository interface {
	Create(ctx context.Context, admin *Admin) error
	FindByID(ctx context.Context, id uuid.UUID) (*Admin, error)
	FindByEmail(ctx context.Context, email string) (*Admin, error)
	FindActive(ctx context.Context) ([]*Admin, error)
	FindByRole(ctx context.Context, role AdminRole) ([]*Admin, error)
	Update(ctx context.Context, admin *Admin) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error
}
