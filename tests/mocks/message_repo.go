package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockMessageRepository implements domain.MessageRepository for testing.
type MockMessageRepository struct {
	CreateFn             func(ctx context.Context, message *domain.Message) error
	FindByConversationFn func(ctx context.Context, conversationID uuid.UUID, cursor *time.Time, limit int) ([]*domain.Message, error)
	MarkAsReadFn         func(ctx context.Context, conversationID uuid.UUID, readerID uuid.UUID) error
	DeleteFn             func(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	GetUnreadCountFn     func(ctx context.Context, userID uuid.UUID) (int64, error)
}

func (m *MockMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, message)
	}
	return nil
}

func (m *MockMessageRepository) FindByConversation(ctx context.Context, conversationID uuid.UUID, cursor *time.Time, limit int) ([]*domain.Message, error) {
	if m.FindByConversationFn != nil {
		return m.FindByConversationFn(ctx, conversationID, cursor, limit)
	}
	return nil, nil
}

func (m *MockMessageRepository) MarkAsRead(ctx context.Context, conversationID uuid.UUID, readerID uuid.UUID) error {
	if m.MarkAsReadFn != nil {
		return m.MarkAsReadFn(ctx, conversationID, readerID)
	}
	return nil
}

func (m *MockMessageRepository) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id, deletedBy)
	}
	return nil
}

func (m *MockMessageRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.GetUnreadCountFn != nil {
		return m.GetUnreadCountFn(ctx, userID)
	}
	return 0, nil
}
