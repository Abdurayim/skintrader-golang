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

// conversationColumns lists all columns for SELECT queries on conversations.
const conversationColumns = `
	id, participant_1, participant_2,
	last_message_content, last_message_sender, last_message_at,
	unread_count_1, unread_count_2,
	initial_post_id,
	deleted_for_1_at, deleted_for_2_at,
	created_at, updated_at
`

// ConversationRepo implements domain.ConversationRepository using PostgreSQL.
type ConversationRepo struct {
	pool *pgxpool.Pool
}

// NewConversationRepo creates a new ConversationRepo.
func NewConversationRepo(pool *pgxpool.Pool) *ConversationRepo {
	return &ConversationRepo{pool: pool}
}

// scanConversation scans a single row into a domain.Conversation.
func scanConversation(row pgx.Row) (*domain.Conversation, error) {
	var c domain.Conversation
	err := row.Scan(
		&c.ID, &c.Participant1, &c.Participant2,
		&c.LastMessageContent, &c.LastMessageSender, &c.LastMessageAt,
		&c.UnreadCount1, &c.UnreadCount2,
		&c.InitialPostID,
		&c.DeletedFor1At, &c.DeletedFor2At,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// canonicalOrder returns the two UUIDs in canonical order (smaller first)
// to satisfy the CHECK (participant_1 < participant_2) constraint.
func canonicalOrder(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

// FindOrCreate finds an existing conversation between two participants or creates a new one.
// Participant UUIDs are canonically ordered (participant_1 < participant_2) before querying/inserting.
func (r *ConversationRepo) FindOrCreate(ctx context.Context, participant1, participant2 uuid.UUID, initialPostID *uuid.UUID) (*domain.Conversation, error) {
	p1, p2 := canonicalOrder(participant1, participant2)

	// Try to find existing conversation first.
	selectQuery := fmt.Sprintf(`SELECT %s FROM conversations WHERE participant_1 = $1 AND participant_2 = $2`, conversationColumns)

	row := r.pool.QueryRow(ctx, selectQuery, p1, p2)
	conv, err := scanConversation(row)
	if err == nil {
		// Found existing conversation. Clear any soft-delete for these participants
		// so it reappears if it was previously deleted.
		return conv, nil
	}
	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("finding conversation: %w", err)
	}

	// Not found, insert a new conversation.
	insertQuery := fmt.Sprintf(`
		INSERT INTO conversations (participant_1, participant_2, initial_post_id)
		VALUES ($1, $2, $3)
		RETURNING %s
	`, conversationColumns)

	row = r.pool.QueryRow(ctx, insertQuery, p1, p2, initialPostID)
	conv, err = scanConversation(row)
	if err != nil {
		return nil, fmt.Errorf("creating conversation: %w", err)
	}

	return conv, nil
}

// FindByID retrieves a conversation by its UUID.
func (r *ConversationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	query := fmt.Sprintf(`SELECT %s FROM conversations WHERE id = $1`, conversationColumns)

	row := r.pool.QueryRow(ctx, query, id)
	conv, err := scanConversation(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, err
	}
	return conv, nil
}

// FindByUser retrieves all conversations for a user, excluding soft-deleted ones.
// Results are ordered by last_message_at DESC (NULLS LAST), then updated_at DESC.
// Uses offset-based pagination.
func (r *ConversationRepo) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Conversation, int64, error) {
	// Count total non-deleted conversations for pagination.
	countQuery := `
		SELECT COUNT(*)
		FROM conversations
		WHERE (participant_1 = $1 AND (deleted_for_1_at IS NULL))
		   OR (participant_2 = $1 AND (deleted_for_2_at IS NULL))
	`

	var total int64
	err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("counting conversations: %w", err)
	}

	if total == 0 {
		return []*domain.Conversation{}, 0, nil
	}

	// Fetch conversations.
	dataQuery := fmt.Sprintf(`
		SELECT %s
		FROM conversations
		WHERE (participant_1 = $1 AND (deleted_for_1_at IS NULL))
		   OR (participant_2 = $1 AND (deleted_for_2_at IS NULL))
		ORDER BY last_message_at DESC NULLS LAST, updated_at DESC
		LIMIT $2 OFFSET $3
	`, conversationColumns)

	rows, err := r.pool.Query(ctx, dataQuery, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying conversations: %w", err)
	}
	defer rows.Close()

	var conversations []*domain.Conversation
	for rows.Next() {
		conv, err := scanConversation(rows)
		if err != nil {
			return nil, 0, err
		}
		conversations = append(conversations, conv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// UpdateLastMessage updates the last message preview fields on a conversation.
func (r *ConversationRepo) UpdateLastMessage(ctx context.Context, conversationID uuid.UUID, senderID uuid.UUID, content string) error {
	// Truncate content to 100 chars to match the VARCHAR(100) column.
	if len(content) > 100 {
		content = content[:100]
	}

	query := `
		UPDATE conversations SET
			last_message_content = $2,
			last_message_sender = $3,
			last_message_at = $4
		WHERE id = $1
	`

	now := time.Now()
	cmdTag, err := r.pool.Exec(ctx, query, conversationID, content, senderID, now)
	if err != nil {
		return fmt.Errorf("updating last message: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

// IncrementUnread increments the unread count for the recipient.
// If the recipient is participant_1, unread_count_1 is incremented; otherwise unread_count_2.
func (r *ConversationRepo) IncrementUnread(ctx context.Context, conversationID uuid.UUID, recipientID uuid.UUID) error {
	query := `
		UPDATE conversations SET
			unread_count_1 = CASE WHEN participant_1 = $2 THEN unread_count_1 + 1 ELSE unread_count_1 END,
			unread_count_2 = CASE WHEN participant_2 = $2 THEN unread_count_2 + 1 ELSE unread_count_2 END
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, conversationID, recipientID)
	if err != nil {
		return fmt.Errorf("incrementing unread count: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

// ResetUnread sets the unread count to 0 for the given user in the conversation.
// If the user is participant_1, unread_count_1 is reset; otherwise unread_count_2.
func (r *ConversationRepo) ResetUnread(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE conversations SET
			unread_count_1 = CASE WHEN participant_1 = $2 THEN 0 ELSE unread_count_1 END,
			unread_count_2 = CASE WHEN participant_2 = $2 THEN 0 ELSE unread_count_2 END
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, conversationID, userID)
	if err != nil {
		return fmt.Errorf("resetting unread count: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

// SoftDeleteFor sets the deleted_for timestamp for the given user in the conversation.
// If the user is participant_1, deleted_for_1_at is set; otherwise deleted_for_2_at.
func (r *ConversationRepo) SoftDeleteFor(ctx context.Context, conversationID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()

	query := `
		UPDATE conversations SET
			deleted_for_1_at = CASE WHEN participant_1 = $2 THEN $3 ELSE deleted_for_1_at END,
			deleted_for_2_at = CASE WHEN participant_2 = $2 THEN $3 ELSE deleted_for_2_at END
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, conversationID, userID, now)
	if err != nil {
		return fmt.Errorf("soft deleting conversation: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("conversation not found")
	}
	return nil
}

// GetTotalUnreadCount returns the total unread message count across all conversations
// where the user is a participant (and the conversation is not soft-deleted for them).
func (r *ConversationRepo) GetTotalUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `
		SELECT COALESCE(SUM(
			CASE
				WHEN participant_1 = $1 AND deleted_for_1_at IS NULL THEN unread_count_1
				WHEN participant_2 = $1 AND deleted_for_2_at IS NULL THEN unread_count_2
				ELSE 0
			END
		), 0)
		FROM conversations
		WHERE participant_1 = $1 OR participant_2 = $1
	`

	var total int64
	err := r.pool.QueryRow(ctx, query, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("getting total unread count: %w", err)
	}
	return total, nil
}
