package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockAdminRepository implements domain.AdminRepository for testing.
type MockAdminRepository struct {
	CreateFn          func(ctx context.Context, admin *domain.Admin) error
	FindByIDFn        func(ctx context.Context, id uuid.UUID) (*domain.Admin, error)
	FindByEmailFn     func(ctx context.Context, email string) (*domain.Admin, error)
	FindActiveFn      func(ctx context.Context) ([]*domain.Admin, error)
	FindByRoleFn      func(ctx context.Context, role domain.AdminRole) ([]*domain.Admin, error)
	UpdateFn          func(ctx context.Context, admin *domain.Admin) error
	UpdateLastLoginFn func(ctx context.Context, id uuid.UUID, ip string) error
}

func (m *MockAdminRepository) Create(ctx context.Context, admin *domain.Admin) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, admin)
	}
	return nil
}

func (m *MockAdminRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Admin, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockAdminRepository) FindByEmail(ctx context.Context, email string) (*domain.Admin, error) {
	if m.FindByEmailFn != nil {
		return m.FindByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockAdminRepository) FindActive(ctx context.Context) ([]*domain.Admin, error) {
	if m.FindActiveFn != nil {
		return m.FindActiveFn(ctx)
	}
	return nil, nil
}

func (m *MockAdminRepository) FindByRole(ctx context.Context, role domain.AdminRole) ([]*domain.Admin, error) {
	if m.FindByRoleFn != nil {
		return m.FindByRoleFn(ctx, role)
	}
	return nil, nil
}

func (m *MockAdminRepository) Update(ctx context.Context, admin *domain.Admin) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, admin)
	}
	return nil
}

func (m *MockAdminRepository) UpdateLastLogin(ctx context.Context, id uuid.UUID, ip string) error {
	if m.UpdateLastLoginFn != nil {
		return m.UpdateLastLoginFn(ctx, id, ip)
	}
	return nil
}
