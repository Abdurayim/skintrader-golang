package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockTransactionRepository implements domain.TransactionRepository for testing.
type MockTransactionRepository struct {
	CreateFn           func(ctx context.Context, transaction *domain.Transaction) error
	FindByIDFn         func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	FindByExternalIDFn func(ctx context.Context, externalID string) (*domain.Transaction, error)
	FindByUserFn       func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, int64, error)
	UpdateFn           func(ctx context.Context, transaction *domain.Transaction) error
	GetRevenueStatsFn  func(ctx context.Context) (*domain.RevenueStats, error)
}

func (m *MockTransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, transaction)
	}
	return nil
}

func (m *MockTransactionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockTransactionRepository) FindByExternalID(ctx context.Context, externalID string) (*domain.Transaction, error) {
	if m.FindByExternalIDFn != nil {
		return m.FindByExternalIDFn(ctx, externalID)
	}
	return nil, nil
}

func (m *MockTransactionRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, int64, error) {
	if m.FindByUserFn != nil {
		return m.FindByUserFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockTransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, transaction)
	}
	return nil
}

func (m *MockTransactionRepository) GetRevenueStats(ctx context.Context) (*domain.RevenueStats, error) {
	if m.GetRevenueStatsFn != nil {
		return m.GetRevenueStatsFn(ctx)
	}
	return nil, nil
}
