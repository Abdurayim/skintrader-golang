package worker

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
)

// CleanupService defines what the worker needs from the subscription service.
type CleanupService interface {
	FindExpired(ctx context.Context) ([]*domain.Subscription, error)
	FindExpiredGracePeriod(ctx context.Context) ([]*domain.Subscription, error)
	MarkExpired(ctx context.Context, sub *domain.Subscription) error
	StartGracePeriod(ctx context.Context, sub *domain.Subscription, graceDays int) error
}

type SubscriptionCleanup struct {
	service CleanupService
	cron    *cron.Cron
	logger  zerolog.Logger
}

func NewSubscriptionCleanup(svc CleanupService, logger zerolog.Logger) *SubscriptionCleanup {
	return &SubscriptionCleanup{
		service: svc,
		cron:    cron.New(),
		logger:  logger.With().Str("worker", "subscription_cleanup").Logger(),
	}
}

func (w *SubscriptionCleanup) Start() {
	_, err := w.cron.AddFunc("0 0 * * *", w.run)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to schedule subscription cleanup")
		return
	}
	w.cron.Start()
	w.logger.Info().Msg("subscription cleanup worker started (daily at 00:00 UTC)")
}

func (w *SubscriptionCleanup) Stop() {
	ctx := w.cron.Stop()
	<-ctx.Done()
	w.logger.Info().Msg("subscription cleanup worker stopped")
}

func (w *SubscriptionCleanup) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	w.logger.Info().Msg("starting subscription cleanup")

	// 1. Move expired active subscriptions to grace period
	expired, err := w.service.FindExpired(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to find expired subscriptions")
	} else {
		for _, sub := range expired {
			if err := w.service.StartGracePeriod(ctx, sub, 3); err != nil {
				w.logger.Error().Err(err).Str("sub_id", sub.ID.String()).Msg("failed to start grace period")
				continue
			}
		}
	}

	// 2. Expire grace periods that have ended
	expiredGrace, err := w.service.FindExpiredGracePeriod(ctx)
	if err != nil {
		w.logger.Error().Err(err).Msg("failed to find expired grace periods")
	} else {
		for _, sub := range expiredGrace {
			if err := w.service.MarkExpired(ctx, sub); err != nil {
				w.logger.Error().Err(err).Str("sub_id", sub.ID.String()).Msg("failed to expire subscription")
				continue
			}
		}
	}

	w.logger.Info().
		Int("moved_to_grace", len(expired)).
		Int("fully_expired", len(expiredGrace)).
		Msg("subscription cleanup completed")
}
