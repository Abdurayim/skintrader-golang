package service

import (
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	jwtpkg "skintrader-go/internal/pkg/jwt"
	"skintrader-go/internal/repository/cache"
	"skintrader-go/internal/repository/postgres"
)

// Services aggregates all service layer implementations.
type Services struct {
	Auth         *AuthService
	User         domain.UserRepository
	Post         domain.PostRepository
	Game         domain.GameRepository
	Subscription *SubscriptionService
	Payment      *PaymentService
	Message      domain.MessageRepository
	Conversation domain.ConversationRepository
	Report       domain.ReportRepository
	Admin        domain.AdminRepository
	KYC          *KYCService
	AdminLog     domain.AdminLogRepository
	Cache        *CacheService
	Image        *ImageService
}

func NewServices(cfg *config.Config, repos *postgres.Repositories, cacheRepo *cache.RedisCache, jwtManager *jwtpkg.Manager, logger zerolog.Logger) *Services {
	cacheService := NewCacheService(cacheRepo, cfg.Cache)
	imageService := NewImageService(cfg.Upload)

	authService := NewAuthService(
		repos.User,
		repos.Admin,
		jwtManager,
		cfg.Auth,
		logger,
	)

	subscriptionService := NewSubscriptionService(
		repos.Subscription,
		repos.Transaction,
		repos.User,
		cfg.Subscription,
		logger,
	)

	paymentService := NewPaymentService(
		repos.Transaction,
		repos.Subscription,
		repos.User,
		cfg.Payment,
		cfg.Subscription,
		logger,
	)

	kycService := NewKYCService(
		repos.User,
		cfg.KYC,
		cfg.FaceService,
		cfg.Upload,
		logger,
	)

	return &Services{
		Auth:         authService,
		User:         repos.User,
		Post:         repos.Post,
		Game:         repos.Game,
		Subscription: subscriptionService,
		Payment:      paymentService,
		Message:      repos.Message,
		Conversation: repos.Conversation,
		Report:       repos.Report,
		Admin:        repos.Admin,
		KYC:          kycService,
		AdminLog:     repos.AdminLog,
		Cache:        cacheService,
		Image:        imageService,
	}
}
