package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Conversation represents a chat conversation between two users.
type Conversation struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	Participant1      uuid.UUID  `json:"participant1" db:"participant1"`
	Participant2      uuid.UUID  `json:"participant2" db:"participant2"`
	LastMessageContent *string   `json:"lastMessageContent,omitempty" db:"last_message_content"`
	LastMessageSender  *uuid.UUID `json:"lastMessageSender,omitempty" db:"last_message_sender"`
	LastMessageAt      *time.Time `json:"lastMessageAt,omitempty" db:"last_message_at"`
	UnreadCount1      int        `json:"unreadCount1" db:"unread_count1"`
	UnreadCount2      int        `json:"unreadCount2" db:"unread_count2"`
	InitialPostID     *uuid.UUID `json:"initialPostId,omitempty" db:"initial_post_id"`
	DeletedFor1At     *time.Time `json:"deletedFor1At,omitempty" db:"deleted_for1_at"`
	DeletedFor2At     *time.Time `json:"deletedFor2At,omitempty" db:"deleted_for2_at"`
	CreatedAt         time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt         time.Time  `json:"updatedAt" db:"updated_at"`

	// Computed fields for the requesting user (not persisted)
	OtherParticipant *ConversationParticipant `json:"otherParticipant,omitempty" db:"-"`
	UnreadCount      int                      `json:"unreadCount" db:"-"`
	LastMessage      *ConversationLastMessage `json:"lastMessage,omitempty" db:"-"`
}

// ConversationParticipant is the public info of the other user in a conversation.
type ConversationParticipant struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
}

// ConversationLastMessage is a lightweight view of the last message in a conversation.
type ConversationLastMessage struct {
	Content  string     `json:"content"`
	SenderID *uuid.UUID `json:"senderId,omitempty"`
	SentAt   *time.Time `json:"sentAt,omitempty"`
}

// ConversationRepository defines the interface for conversation data access.
type ConversationRepository interface {
	FindOrCreate(ctx context.Context, participant1, participant2 uuid.UUID, initialPostID *uuid.UUID) (*Conversation, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Conversation, int64, error)
	UpdateLastMessage(ctx context.Context, conversationID uuid.UUID, senderID uuid.UUID, content string) error
	IncrementUnread(ctx context.Context, conversationID uuid.UUID, recipientID uuid.UUID) error
	ResetUnread(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	SoftDeleteFor(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error
	GetTotalUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
}
