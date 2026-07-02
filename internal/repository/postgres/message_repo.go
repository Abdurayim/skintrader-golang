package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// messageColumns lists all columns for SELECT queries on messages.
const messageColumns = `
	id, conversation_id, sender_id,
	content, status, read_at,
	post_id,
	deleted_at, deleted_by,
	created_at
`

// MessageRepo implements domain.MessageRepository using PostgreSQL.
type MessageRepo struct {
	pool     *pgxpool.Pool
	convRepo *ConversationRepo
}

// NewMessageRepo creates a new MessageRepo.
// It requires a ConversationRepo to update conversation metadata when messages are created/read.
func NewMessageRepo(pool *pgxpool.Pool, convRepo *ConversationRepo) *MessageRepo {
	return &MessageRepo{pool: pool, convRepo: convRepo}
}

// scanMessage scans a single row into a domain.Message.
func scanMessage(row pgx.Row) (*domain.Message, error) {
	var m domain.Message
	err := row.Scan(
		&m.ID, &m.ConversationID, &m.SenderID,
		&m.Content, &m.Status, &m.ReadAt,
		&m.PostID,
		&m.DeletedAt, &m.DeletedBy,
		&m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create inserts a new message into the database.
// It also updates the conversation's last message preview and increments the recipient's unread count.
func (r *MessageRepo) Create(ctx context.Context, message *domain.Message) error {
	if message.ID == uuid.Nil {
		message.ID = uuid.New()
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO messages (id, conversation_id, sender_id, content, status, post_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING %s
	`, messageColumns)

	row := r.pool.QueryRow(ctx, insertQuery,
		message.ID, message.ConversationID, message.SenderID,
		message.Content, message.Status, message.PostID,
	)

	created, err := scanMessage(row)
	if err != nil {
		return fmt.Errorf("inserting message: %w", err)
	}

	// Copy the database-generated fields back to the caller's message.
	message.CreatedAt = created.CreatedAt
	message.Status = created.Status

	// Update the conversation's last message preview.
	if err := r.convRepo.UpdateLastMessage(ctx, message.ConversationID, message.SenderID, message.Content); err != nil {
		return fmt.Errorf("updating last message: %w", err)
	}

	// Determine the recipient and increment their unread count.
	// We need to find which participant is NOT the sender.
	conv, err := r.convRepo.FindByID(ctx, message.ConversationID)
	if err != nil {
		return fmt.Errorf("finding conversation for unread increment: %w", err)
	}

	var recipientID uuid.UUID
	if conv.Participant1 == message.SenderID {
		recipientID = conv.Participant2
	} else {
		recipientID = conv.Participant1
	}

	if err := r.convRepo.IncrementUnread(ctx, message.ConversationID, recipientID); err != nil {
		return fmt.Errorf("incrementing unread count: %w", err)
	}

	return nil
}

// FindByConversation retrieves messages for a conversation using cursor-based pagination.
// If cursor is nil, the most recent messages are returned.
// Messages are ordered by created_at DESC (newest first).
func (r *MessageRepo) FindByConversation(ctx context.Context, conversationID uuid.UUID, cursor *time.Time, limit int) ([]*domain.Message, error) {
	var rows pgx.Rows
	var err error

	if cursor != nil {
		query := fmt.Sprintf(`
			SELECT %s
			FROM messages
			WHERE conversation_id = $1
			  AND created_at < $2
			  AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $3
		`, messageColumns)

		rows, err = r.pool.Query(ctx, query, conversationID, *cursor, limit)
	} else {
		query := fmt.Sprintf(`
			SELECT %s
			FROM messages
			WHERE conversation_id = $1
			  AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $2
		`, messageColumns)

		rows, err = r.pool.Query(ctx, query, conversationID, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("querying messages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// MarkAsRead marks all unread messages in a conversation as read for the given reader.
// Only messages sent by the OTHER participant (sender_id != readerID) are marked.
// It also resets the reader's unread count on the conversation.
func (r *MessageRepo) MarkAsRead(ctx context.Context, conversationID uuid.UUID, readerID uuid.UUID) error {
	query := `
		UPDATE messages SET
			status = 'read',
			read_at = NOW()
		WHERE conversation_id = $1
		  AND sender_id != $2
		  AND status != 'read'
		  AND deleted_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, conversationID, readerID)
	if err != nil {
		return fmt.Errorf("marking messages as read: %w", err)
	}

	// Reset the reader's unread count on the conversation.
	if err := r.convRepo.ResetUnread(ctx, conversationID, readerID); err != nil {
		return fmt.Errorf("resetting unread count: %w", err)
	}

	return nil
}

// Delete performs a soft delete on a message by setting deleted_at and deleted_by.
func (r *MessageRepo) Delete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID) error {
	query := `
		UPDATE messages SET
			deleted_at = NOW(),
			deleted_by = $2
		WHERE id = $1
		  AND deleted_at IS NULL
	`

	cmdTag, err := r.pool.Exec(ctx, query, id, deletedBy)
	if err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("message not found or already deleted")
	}
	return nil
}

// GetUnreadCount returns the total count of unread messages across all conversations
// where the user is a recipient (i.e., messages sent by others that have not been read).
func (r *MessageRepo) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE (c.participant_1 = $1 OR c.participant_2 = $1)
		  AND m.sender_id != $1
		  AND m.status != 'read'
		  AND m.deleted_at IS NULL
	`

	var count int64
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("getting unread count: %w", err)
	}
	return count, nil
}
