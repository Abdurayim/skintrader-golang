package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// NotificationRepo implements domain.NotificationRepository using PostgreSQL.
type NotificationRepo struct {
	pool *pgxpool.Pool
}

// NewNotificationRepo creates a new NotificationRepo.
func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool}
}

func scanNotification(row pgx.Row) (*domain.Notification, error) {
	var n domain.Notification
	var data []byte
	if err := row.Scan(&n.ID, &n.UserID, &n.Type, &data, &n.ReadAt, &n.CreatedAt); err != nil {
		return nil, err
	}
	if data != nil {
		n.Data = data
	}
	n.Read = n.ReadAt != nil
	return &n, nil
}

// Create inserts a notification.
func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	var data []byte
	if n.Data != nil {
		data = []byte(n.Data)
	}
	return r.pool.QueryRow(ctx,
		`INSERT INTO notifications (id, user_id, type, data) VALUES ($1, $2, $3, $4)
		 RETURNING created_at`,
		n.ID, n.UserID, n.Type, data,
	).Scan(&n.CreatedAt)
}

// FindByUser returns a user's notifications, newest first.
func (r *NotificationRepo) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Notification, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting notifications: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, data, read_at, created_at FROM notifications
		 WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("querying notifications: %w", err)
	}
	defer rows.Close()

	notifications := []*domain.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning notification: %w", err)
		}
		notifications = append(notifications, n)
	}
	return notifications, total, rows.Err()
}

// CountUnread returns the number of unread notifications for a user.
func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID,
	).Scan(&count)
	return count, err
}

// MarkRead marks a single notification as read (only the owner's).
func (r *NotificationRepo) MarkRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`,
		id, userID,
	)
	return err
}

// MarkAllRead marks all of a user's notifications as read.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`,
		userID,
	)
	return err
}
