package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	"skintrader-go/internal/service"
	"skintrader-go/tests/mocks"
	"skintrader-go/tests/testutil"
)

// newTestSubscriptionService creates a SubscriptionService with mock repos.
func newTestSubscriptionService(
	subRepo *mocks.MockSubscriptionRepository,
	txRepo *mocks.MockTransactionRepository,
	userRepo *mocks.MockUserRepository,
) *service.SubscriptionService {
	cfg := config.SubscriptionConfig{
		PriceUZS:     10000,
		PriceUSD:     1,
		DurationDays: 30,
		GraceDays:    3,
	}
	logger := zerolog.Nop()
	return service.NewSubscriptionService(subRepo, txRepo, userRepo, cfg, logger)
}

// ---------------------------------------------------------------------------
// GetStatus tests
// ---------------------------------------------------------------------------

func TestGetStatus_Active(t *testing.T) {
	user := testutil.NewTestUser()
	sub := testutil.NewTestSubscription(user.ID)

	subRepo := &mocks.MockSubscriptionRepository{
		FindActiveFn: func(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
			if userID == user.ID {
				return sub, nil
			}
			return nil, nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	result, err := svc.GetStatus(context.Background(), user.ID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	testutil.AssertEqual(t, result.Status, domain.SubscriptionStatusActive)
	testutil.AssertEqual(t, result.UserID, user.ID)
}

func TestGetStatus_NoSubscription(t *testing.T) {
	subRepo := &mocks.MockSubscriptionRepository{
		FindActiveFn: func(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
			return nil, nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	result, err := svc.GetStatus(context.Background(), uuid.New())
	testutil.AssertNoError(t, err)
	if result != nil {
		t.Fatal("expected nil subscription for user with no active subscription")
	}
}

// ---------------------------------------------------------------------------
// Cancel tests
// ---------------------------------------------------------------------------

func TestCancel_Success(t *testing.T) {
	user := testutil.NewTestUser()
	sub := testutil.NewTestSubscription(user.ID)

	var updatedSub *domain.Subscription
	subRepo := &mocks.MockSubscriptionRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
			if id == sub.ID {
				return sub, nil
			}
			return nil, fmt.Errorf("not found")
		},
		UpdateFn: func(ctx context.Context, subscription *domain.Subscription) error {
			updatedSub = subscription
			return nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.Cancel(context.Background(), sub.ID, "I want to cancel")
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, updatedSub)
	testutil.AssertFalse(t, updatedSub.AutoRenew)
	testutil.AssertNotNil(t, updatedSub.CancelledAt)
	testutil.AssertNotNil(t, updatedSub.CancelReason)
	testutil.AssertEqual(t, *updatedSub.CancelReason, "I want to cancel")
}

func TestCancel_NotFound(t *testing.T) {
	subRepo := &mocks.MockSubscriptionRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.Cancel(context.Background(), uuid.New(), "reason")
	testutil.AssertError(t, err)
}

func TestCancel_NilSubscription(t *testing.T) {
	subRepo := &mocks.MockSubscriptionRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Subscription, error) {
			return nil, nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.Cancel(context.Background(), uuid.New(), "reason")
	testutil.AssertError(t, err)
}

// ---------------------------------------------------------------------------
// ActivateFromTransaction tests
// ---------------------------------------------------------------------------

func TestActivateFromTransaction_Success(t *testing.T) {
	user := testutil.NewTestUser()
	txID := uuid.New()

	var createdSub *domain.Subscription
	subRepo := &mocks.MockSubscriptionRepository{
		CreateFn: func(ctx context.Context, subscription *domain.Subscription) error {
			createdSub = subscription
			return nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{
		UpdateSubscriptionStatusFn: func(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error {
			return nil
		},
	}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.ActivateFromTransaction(context.Background(), user.ID, txID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, createdSub)
	testutil.AssertEqual(t, createdSub.UserID, user.ID)
	testutil.AssertEqual(t, createdSub.Status, domain.SubscriptionStatusActive)
	testutil.AssertEqual(t, createdSub.Plan, domain.SubscriptionPlanMonthly)
	testutil.AssertNotNil(t, createdSub.StartDate)
	testutil.AssertNotNil(t, createdSub.EndDate)
	testutil.AssertNotNil(t, createdSub.LastPaymentID)
	testutil.AssertEqual(t, *createdSub.LastPaymentID, txID)
}

func TestActivateFromTransaction_CreateError(t *testing.T) {
	subRepo := &mocks.MockSubscriptionRepository{
		CreateFn: func(ctx context.Context, subscription *domain.Subscription) error {
			return fmt.Errorf("db error")
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.ActivateFromTransaction(context.Background(), uuid.New(), uuid.New())
	testutil.AssertError(t, err)
}

// ---------------------------------------------------------------------------
// GetHistory tests
// ---------------------------------------------------------------------------

func TestGetHistory_Success(t *testing.T) {
	user := testutil.NewTestUser()
	sub1 := testutil.NewTestSubscription(user.ID)
	sub2 := testutil.NewTestSubscription(user.ID)
	sub2.Status = domain.SubscriptionStatusExpired

	subRepo := &mocks.MockSubscriptionRepository{
		FindByUserFn: func(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error) {
			return []*domain.Subscription{sub1, sub2}, nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	subs, err := svc.GetHistory(context.Background(), user.ID)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(subs), 2)
}

// ---------------------------------------------------------------------------
// MarkExpired tests
// ---------------------------------------------------------------------------

func TestMarkExpired_Success(t *testing.T) {
	user := testutil.NewTestUser()
	sub := testutil.NewTestSubscription(user.ID)

	subRepo := &mocks.MockSubscriptionRepository{
		UpdateFn: func(ctx context.Context, subscription *domain.Subscription) error {
			return nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{
		UpdateSubscriptionStatusFn: func(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error {
			return nil
		},
	}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.MarkExpired(context.Background(), sub)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, sub.Status, domain.SubscriptionStatusExpired)
}

// ---------------------------------------------------------------------------
// StartGracePeriod tests
// ---------------------------------------------------------------------------

func TestStartGracePeriod_Success(t *testing.T) {
	user := testutil.NewTestUser()
	sub := testutil.NewTestSubscription(user.ID)

	subRepo := &mocks.MockSubscriptionRepository{
		UpdateFn: func(ctx context.Context, subscription *domain.Subscription) error {
			return nil
		},
	}
	txRepo := &mocks.MockTransactionRepository{}
	userRepo := &mocks.MockUserRepository{
		UpdateSubscriptionStatusFn: func(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error {
			return nil
		},
	}
	svc := newTestSubscriptionService(subRepo, txRepo, userRepo)

	err := svc.StartGracePeriod(context.Background(), sub, 3)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, sub.Status, domain.SubscriptionStatusGracePeriod)
	testutil.AssertNotNil(t, sub.GracePeriodStarted)
}
