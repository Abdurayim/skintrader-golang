package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
)

type SubscriptionService struct {
	subRepo   domain.SubscriptionRepository
	txRepo    domain.TransactionRepository
	userRepo  domain.UserRepository
	config    config.SubscriptionConfig
	logger    zerolog.Logger
}

func NewSubscriptionService(subRepo domain.SubscriptionRepository, txRepo domain.TransactionRepository, userRepo domain.UserRepository, cfg config.SubscriptionConfig, logger zerolog.Logger) *SubscriptionService {
	return &SubscriptionService{
		subRepo:  subRepo,
		txRepo:   txRepo,
		userRepo: userRepo,
		config:   cfg,
		logger:   logger.With().Str("service", "subscription").Logger(),
	}
}

// GetStatus returns the current subscription status for a user.
func (s *SubscriptionService) GetStatus(ctx context.Context, userID uuid.UUID) (*domain.Subscription, error) {
	return s.subRepo.FindActive(ctx, userID)
}

// GetHistory returns subscription history for a user.
func (s *SubscriptionService) GetHistory(ctx context.Context, userID uuid.UUID) ([]*domain.Subscription, error) {
	return s.subRepo.FindByUser(ctx, userID)
}

// Cancel cancels auto-renewal for a subscription.
func (s *SubscriptionService) Cancel(ctx context.Context, subID uuid.UUID, reason string) error {
	sub, err := s.subRepo.FindByID(ctx, subID)
	if err != nil {
		return fmt.Errorf("find subscription: %w", err)
	}
	if sub == nil {
		return fmt.Errorf("subscription not found")
	}

	now := time.Now()
	sub.AutoRenew = false
	sub.CancelledAt = &now
	sub.CancelReason = &reason

	return s.subRepo.Update(ctx, sub)
}

// ActivateFromTransaction creates and activates a subscription from a completed payment.
func (s *SubscriptionService) ActivateFromTransaction(ctx context.Context, userID uuid.UUID, txID uuid.UUID) error {
	now := time.Now()
	endDate := now.AddDate(0, 0, s.config.DurationDays)

	sub := &domain.Subscription{
		ID:            uuid.New(),
		UserID:        userID,
		Status:        domain.SubscriptionStatusActive,
		Plan:          domain.SubscriptionPlanMonthly,
		StartDate:     &now,
		EndDate:       &endDate,
		AutoRenew:     false,
		LastPaymentID: &txID,
	}

	if err := s.subRepo.Create(ctx, sub); err != nil {
		return fmt.Errorf("create subscription: %w", err)
	}

	expiresAt := endDate
	if err := s.userRepo.UpdateSubscriptionStatus(ctx, userID, domain.SubscriptionStatusActive, &sub.ID, &expiresAt); err != nil {
		return fmt.Errorf("update user subscription status: %w", err)
	}

	return nil
}

// FindExpired finds all active subscriptions that have expired.
func (s *SubscriptionService) FindExpired(ctx context.Context) ([]*domain.Subscription, error) {
	return s.subRepo.FindExpired(ctx)
}

// FindExpiredGracePeriod finds all subscriptions with expired grace periods.
func (s *SubscriptionService) FindExpiredGracePeriod(ctx context.Context) ([]*domain.Subscription, error) {
	return s.subRepo.FindExpiredGracePeriod(ctx)
}

// MarkExpired marks a subscription as expired.
func (s *SubscriptionService) MarkExpired(ctx context.Context, sub *domain.Subscription) error {
	sub.Status = domain.SubscriptionStatusExpired
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return err
	}
	return s.userRepo.UpdateSubscriptionStatus(ctx, sub.UserID, domain.SubscriptionStatusExpired, nil, nil)
}

// StartGracePeriod moves a subscription to grace period.
func (s *SubscriptionService) StartGracePeriod(ctx context.Context, sub *domain.Subscription, graceDays int) error {
	now := time.Now()
	graceEnd := now.AddDate(0, 0, graceDays)

	sub.Status = domain.SubscriptionStatusGracePeriod
	sub.GracePeriodStarted = &now
	if err := s.subRepo.Update(ctx, sub); err != nil {
		return err
	}

	return s.userRepo.UpdateSubscriptionStatus(ctx, sub.UserID, domain.SubscriptionStatusGracePeriod, &sub.ID, &graceEnd)
}
