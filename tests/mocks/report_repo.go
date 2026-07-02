package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockReportRepository implements domain.ReportRepository for testing.
type MockReportRepository struct {
	CreateFn          func(ctx context.Context, report *domain.Report) error
	FindByIDFn        func(ctx context.Context, id uuid.UUID) (*domain.Report, error)
	FindByReporterFn  func(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]*domain.Report, int64, error)
	FindByTargetFn    func(ctx context.Context, targetID uuid.UUID) ([]*domain.Report, error)
	FindPendingFn     func(ctx context.Context, limit, offset int) ([]*domain.Report, int64, error)
	UpdateFn          func(ctx context.Context, report *domain.Report) error
	CheckDuplicateFn  func(ctx context.Context, reportHash string) (bool, error)
	CountByTargetFn   func(ctx context.Context, targetID uuid.UUID) (int64, error)
	GetStatsFn        func(ctx context.Context) (*domain.ReportStats, error)
	ListWithFiltersFn func(ctx context.Context, filter domain.ReportListFilter) ([]*domain.Report, int64, error)
}

func (m *MockReportRepository) Create(ctx context.Context, report *domain.Report) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, report)
	}
	return nil
}

func (m *MockReportRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Report, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockReportRepository) FindByReporter(ctx context.Context, reporterID uuid.UUID, limit, offset int) ([]*domain.Report, int64, error) {
	if m.FindByReporterFn != nil {
		return m.FindByReporterFn(ctx, reporterID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockReportRepository) FindByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.Report, error) {
	if m.FindByTargetFn != nil {
		return m.FindByTargetFn(ctx, targetID)
	}
	return nil, nil
}

func (m *MockReportRepository) FindPending(ctx context.Context, limit, offset int) ([]*domain.Report, int64, error) {
	if m.FindPendingFn != nil {
		return m.FindPendingFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockReportRepository) Update(ctx context.Context, report *domain.Report) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, report)
	}
	return nil
}

func (m *MockReportRepository) CheckDuplicate(ctx context.Context, reportHash string) (bool, error) {
	if m.CheckDuplicateFn != nil {
		return m.CheckDuplicateFn(ctx, reportHash)
	}
	return false, nil
}

func (m *MockReportRepository) CountByTarget(ctx context.Context, targetID uuid.UUID) (int64, error) {
	if m.CountByTargetFn != nil {
		return m.CountByTargetFn(ctx, targetID)
	}
	return 0, nil
}

func (m *MockReportRepository) GetStats(ctx context.Context) (*domain.ReportStats, error) {
	if m.GetStatsFn != nil {
		return m.GetStatsFn(ctx)
	}
	return nil, nil
}

func (m *MockReportRepository) ListWithFilters(ctx context.Context, filter domain.ReportListFilter) ([]*domain.Report, int64, error) {
	if m.ListWithFiltersFn != nil {
		return m.ListWithFiltersFn(ctx, filter)
	}
	return nil, 0, nil
}
