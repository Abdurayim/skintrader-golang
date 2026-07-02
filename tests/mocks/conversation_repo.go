package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockConversationRepository implements domain.ConversationRepository for testing.
type MockConversationRepository struct {
	FindOrCreateFn       func(ctx context.Context, participant1, participant2 uuid.UUID, initialPostID *uuid.UUID) (*domain.Conversation, error)
	FindByIDFn           func(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	FindByUserFn         func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, int64, error)
	UpdateLastMessageFn  func(ctx context.Context, conversationID uuid.UUID, senderID uuid.UUID, content string) error
	IncrementUnreadFn    func(ctx context.Context, conversationID uuid.UUID, recipientID uuid.UUID) error
	ResetUnreadFn        func(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	SoftDeleteForFn      func(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	GetTotalUnreadCountFn func(ctx context.Context, userID uuid.UUID) (int64, error)
}

func (m *MockConversationRepository) FindOrCreate(ctx context.Context, participant1, participant2 uuid.UUID, initialPostID *uuid.UUID) (*domain.Conversation, error) {
	if m.FindOrCreateFn != nil {
		return m.FindOrCreateFn(ctx, participant1, participant2, initialPostID)
	}
	return nil, nil
}

func (m *MockConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockConversationRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, int64, error) {
	if m.FindByUserFn != nil {
		return m.FindByUserFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockConversationRepository) UpdateLastMessage(ctx context.Context, conversationID uuid.UUID, senderID uuid.UUID, content string) error {
	if m.UpdateLastMessageFn != nil {
		return m.UpdateLastMessageFn(ctx, conversationID, senderID, content)
	}
	return nil
}

func (m *MockConversationRepository) IncrementUnread(ctx context.Context, conversationID uuid.UUID, recipientID uuid.UUID) error {
	if m.IncrementUnreadFn != nil {
		return m.IncrementUnreadFn(ctx, conversationID, recipientID)
	}
	return nil
}

func (m *MockConversationRepository) ResetUnread(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	if m.ResetUnreadFn != nil {
		return m.ResetUnreadFn(ctx, conversationID, userID)
	}
	return nil
}

func (m *MockConversationRepository) SoftDeleteFor(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	if m.SoftDeleteForFn != nil {
		return m.SoftDeleteForFn(ctx, conversationID, userID)
	}
	return nil
}

func (m *MockConversationRepository) GetTotalUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.GetTotalUnreadCountFn != nil {
		return m.GetTotalUnreadCountFn(ctx, userID)
	}
	return 0, nil
}
