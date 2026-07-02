package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message within a conversation.
type Message struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	ConversationID uuid.UUID     `json:"conversationId" db:"conversation_id"`
	SenderID       uuid.UUID     `json:"senderId" db:"sender_id"`
	Content        string        `json:"content" db:"content"`
	Status         MessageStatus `json:"status" db:"status"`
	ReadAt         *time.Time    `json:"readAt,omitempty" db:"read_at"`
	PostID         *uuid.UUID    `json:"postId,omitempty" db:"post_id"`
	DeletedAt      *time.Time    `json:"deletedAt,omitempty" db:"deleted_at"`
	DeletedBy      *uuid.UUID    `json:"deletedBy,omitempty" db:"deleted_by"`
	CreatedAt      time.Time     `json:"createdAt" db:"created_at"`

	// Computed field (not persisted): true when the message has been read
	Read bool `json:"read" db:"-"`
}

// MessageRepository defines the interface for message data access.
type MessageRepository interface {
	Create(ctx context.Context, message *Message) error
	FindByConversation(ctx context.Context, conversationID uuid.UUID, cursor *time.Time, limit int) ([]*Message, error)
	MarkAsRead(ctx context.Context, conversationID uuid.UUID, readerID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error
	GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}
