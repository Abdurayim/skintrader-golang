package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockAdminLogRepository implements domain.AdminLogRepository for testing.
type MockAdminLogRepository struct {
	CreateFn          func(ctx context.Context, log *domain.AdminLog) error
	FindByAdminFn     func(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error)
	FindByTargetFn    func(ctx context.Context, targetType string, targetID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error)
	FindByActionFn    func(ctx context.Context, action domain.AdminAction, limit, offset int) ([]*domain.AdminLog, int64, error)
	GetRecentLogsFn   func(ctx context.Context, limit int) ([]*domain.AdminLog, error)
	ListWithFiltersFn func(ctx context.Context, filter domain.AdminLogListFilter) ([]*domain.AdminLog, int64, error)
}

func (m *MockAdminLogRepository) Create(ctx context.Context, log *domain.AdminLog) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, log)
	}
	return nil
}

func (m *MockAdminLogRepository) FindByAdmin(ctx context.Context, adminID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error) {
	if m.FindByAdminFn != nil {
		return m.FindByAdminFn(ctx, adminID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockAdminLogRepository) FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, limit, offset int) ([]*domain.AdminLog, int64, error) {
	if m.FindByTargetFn != nil {
		return m.FindByTargetFn(ctx, targetType, targetID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockAdminLogRepository) FindByAction(ctx context.Context, action domain.AdminAction, limit, offset int) ([]*domain.AdminLog, int64, error) {
	if m.FindByActionFn != nil {
		return m.FindByActionFn(ctx, action, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockAdminLogRepository) GetRecentLogs(ctx context.Context, limit int) ([]*domain.AdminLog, error) {
	if m.GetRecentLogsFn != nil {
		return m.GetRecentLogsFn(ctx, limit)
	}
	return nil, nil
}

func (m *MockAdminLogRepository) ListWithFilters(ctx context.Context, filter domain.AdminLogListFilter) ([]*domain.AdminLog, int64, error) {
	if m.ListWithFiltersFn != nil {
		return m.ListWithFiltersFn(ctx, filter)
	}
	return nil, 0, nil
}
