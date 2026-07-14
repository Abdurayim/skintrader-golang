package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// Repositories aggregates all PostgreSQL repository implementations.
type Repositories struct {
	User         domain.UserRepository
	Post         domain.PostRepository
	Game         domain.GameRepository
	Subscription domain.SubscriptionRepository
	Transaction  domain.TransactionRepository
	Message      domain.MessageRepository
	Conversation domain.ConversationRepository
	Report       domain.ReportRepository
	Admin        domain.AdminRepository
	AdminLog     domain.AdminLogRepository
	BalanceTopup domain.BalanceTopupRepository
	pool         *pgxpool.Pool
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	convRepo := NewConversationRepo(pool)
	return &Repositories{
		User:         NewUserRepo(pool),
		Post:         NewPostRepo(pool),
		Game:         NewGameRepo(pool),
		Subscription: NewSubscriptionRepo(pool),
		Transaction:  NewTransactionRepo(pool),
		Message:      NewMessageRepo(pool, convRepo),
		Conversation: convRepo,
		Report:       NewReportRepo(pool),
		Admin:        NewAdminRepo(pool),
		AdminLog:     NewAdminLogRepo(pool),
		BalanceTopup: NewBalanceTopupRepo(pool),
		pool:         pool,
	}
}
