package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockSubscriptionRepository implements domain.SubscriptionRepository for testing.
type MockSubscriptionRepository struct {
	CreateFn                  func(ctx context.Context, subscription *domain.Subscription) error
	FindByIDFn                func(ctx context.Context, id uuid.UUID) (*domain.Subscription, error)
	FindByUserFn              func(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error)
	FindActiveFn              func(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error)
	UpdateFn                  func(ctx context.Context, subscription *domain.Subscription) error
	FindExpiredFn             func(ctx context.Context) ([]*domain.Subscription, error)
	FindExpiredGracePeriodFn  func(ctx context.Context) ([]*domain.Subscription, error)
	GetStatsFn                func(ctx context.Context) (*domain.SubscriptionStats, error)
}

func (m *MockSubscriptionRepository) Create(ctx context.Context, subscription *domain.Subscription) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, subscription)
	}
	return nil
}

func (m *MockSubscriptionRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error) {
	if m.FindByUserFn != nil {
		return m.FindByUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) FindActive(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	if m.FindActiveFn != nil {
		return m.FindActiveFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) Update(ctx context.Context, subscription *domain.Subscription) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, subscription)
	}
	return nil
}

func (m *MockSubscriptionRepository) FindExpired(ctx context.Context) ([]*domain.Subscription, error) {
	if m.FindExpiredFn != nil {
		return m.FindExpiredFn(ctx)
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) FindExpiredGracePeriod(ctx context.Context) ([]*domain.Subscription, error) {
	if m.FindExpiredGracePeriodFn != nil {
		return m.FindExpiredGracePeriodFn(ctx)
	}
	return nil, nil
}

func (m *MockSubscriptionRepository) GetStats(ctx context.Context) (*domain.SubscriptionStats, error) {
	if m.GetStatsFn != nil {
		return m.GetStatsFn(ctx)
	}
	return nil, nil
}
