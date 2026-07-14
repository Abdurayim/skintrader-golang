package handler

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
)

// NotificationHandler serves users' on-site notifications.
type NotificationHandler struct {
	notifRepo domain.NotificationRepository
	logger    zerolog.Logger
}

func NewNotificationHandler(notifRepo domain.NotificationRepository, logger zerolog.Logger) *NotificationHandler {
	return &NotificationHandler{
		notifRepo: notifRepo,
		logger:    logger.With().Str("handler", "notification").Logger(),
	}
}

// notify creates a notification, logging (not propagating) failures so a
// notification problem never breaks the main operation.
func notify(ctx context.Context, repo domain.NotificationRepository, logger zerolog.Logger, userID uuid.UUID, nType domain.NotificationType, data any) {
	if repo == nil {
		return
	}
	var payload json.RawMessage
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			payload = b
		}
	}
	n := &domain.Notification{UserID: userID, Type: nType, Data: payload}
	if err := repo.Create(ctx, n); err != nil {
		logger.Error().Err(err).Str("userID", userID.String()).Str("type", string(nType)).Msg("failed to create notification")
	}
}

// GetNotifications returns the user's notifications plus the unread count.
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	notifications, total, err := h.notifRepo.FindByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to list notifications")
		Error(c, err)
		return
	}

	unread, err := h.notifRepo.CountUnread(c.Request.Context(), userID)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to count unread notifications")
	}

	Paginated(c, gin.H{
		"notifications": notifications,
		"unreadCount":   unread,
	}, gin.H{"total": total, "limit": limit, "offset": offset}, "Notifications retrieved")
}

// MarkRead marks one notification as read.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid notification ID")
		return
	}

	if err := h.notifRepo.MarkRead(c.Request.Context(), id, userID); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark notification read")
		Error(c, err)
		return
	}

	Success(c, nil, "Notification marked as read")
}

// MarkAllRead marks all of the user's notifications as read.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	if err := h.notifRepo.MarkAllRead(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark all notifications read")
		Error(c, err)
		return
	}

	Success(c, nil, "All notifications marked as read")
}
