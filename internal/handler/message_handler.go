package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
)

type MessageHandler struct {
	messageRepo      domain.MessageRepository
	conversationRepo domain.ConversationRepository
	userRepo         domain.UserRepository
	logger           zerolog.Logger
}

func NewMessageHandler(messageRepo domain.MessageRepository, conversationRepo domain.ConversationRepository, userRepo domain.UserRepository, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *MessageHandler {
	return &MessageHandler{
		messageRepo:      messageRepo,
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		logger:           logger.With().Str("handler", "message").Logger(),
	}
}

// enrichConversation fills the computed fields the frontend expects:
// otherParticipant, unreadCount (for the requesting user), and lastMessage.
func (h *MessageHandler) enrichConversation(c *gin.Context, conv *domain.Conversation, userID uuid.UUID) {
	otherID := conv.Participant1
	conv.UnreadCount = conv.UnreadCount2
	if conv.Participant1 == userID {
		otherID = conv.Participant2
		conv.UnreadCount = conv.UnreadCount1
	}

	if other, err := h.userRepo.FindByID(c.Request.Context(), otherID); err == nil && other != nil {
		conv.OtherParticipant = &domain.ConversationParticipant{
			ID:          other.ID,
			DisplayName: other.DisplayName,
			AvatarURL:   other.AvatarURL,
		}
	}

	if conv.LastMessageContent != nil {
		conv.LastMessage = &domain.ConversationLastMessage{
			Content:  *conv.LastMessageContent,
			SenderID: conv.LastMessageSender,
			SentAt:   conv.LastMessageAt,
		}
	}
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		ConversationID string `json:"conversationId" binding:"required"`
		Content        string `json:"content" binding:"required"`
		PostID         string `json:"postId,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "conversation_id and content are required")
		return
	}

	convID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		BadRequest(c, "Invalid conversation ID")
		return
	}

	conv, err := h.conversationRepo.FindByID(c.Request.Context(), convID)
	if err != nil {
		Error(c, err)
		return
	}
	if conv.Participant1 != userID && conv.Participant2 != userID {
		Forbidden(c, "Not a participant in this conversation")
		return
	}

	msg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       userID,
		Content:        req.Content,
		Status:         domain.MessageStatusSent,
	}

	if req.PostID != "" {
		if postID, err := uuid.Parse(req.PostID); err == nil {
			msg.PostID = &postID
		}
	}

	if err := h.messageRepo.Create(c.Request.Context(), msg); err != nil {
		h.logger.Error().Err(err).Msg("failed to send message")
		Error(c, err)
		return
	}

	_ = h.conversationRepo.UpdateLastMessage(c.Request.Context(), convID, userID, req.Content)

	recipientID := conv.Participant1
	if recipientID == userID {
		recipientID = conv.Participant2
	}
	_ = h.conversationRepo.IncrementUnread(c.Request.Context(), convID, recipientID)

	Created(c, gin.H{"message": msg}, "Message sent")
}

func (h *MessageHandler) GetConversations(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	conversations, total, err := h.conversationRepo.FindByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get conversations")
		Error(c, err)
		return
	}

	for _, conv := range conversations {
		h.enrichConversation(c, conv, userID)
	}

	Paginated(c, gin.H{"conversations": conversations}, gin.H{"total": total, "limit": limit, "offset": offset}, "Conversations retrieved")
}

func (h *MessageHandler) StartConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		RecipientID string `json:"recipientId" binding:"required"`
		PostID      string `json:"postId,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "recipient_id is required")
		return
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		BadRequest(c, "Invalid recipient ID")
		return
	}

	if recipientID == userID {
		BadRequest(c, "Cannot start a conversation with yourself")
		return
	}

	var postID *uuid.UUID
	if req.PostID != "" {
		if pid, err := uuid.Parse(req.PostID); err == nil {
			postID = &pid
		}
	}

	conv, err := h.conversationRepo.FindOrCreate(c.Request.Context(), userID, recipientID, postID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to start conversation")
		Error(c, err)
		return
	}

	h.enrichConversation(c, conv, userID)

	Success(c, gin.H{"conversation": conv}, "Conversation started")
}

func (h *MessageHandler) GetConversationMessages(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid conversation ID")
		return
	}

	conv, err := h.conversationRepo.FindByID(c.Request.Context(), convID)
	if err != nil {
		Error(c, err)
		return
	}
	if conv.Participant1 != userID && conv.Participant2 != userID {
		Forbidden(c, "Not a participant in this conversation")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var cursor *time.Time
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		if t, err := time.Parse(time.RFC3339Nano, cursorStr); err == nil {
			cursor = &t
		}
	}

	messages, err := h.messageRepo.FindByConversation(c.Request.Context(), convID, cursor, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get messages")
		Error(c, err)
		return
	}

	for _, msg := range messages {
		msg.Read = msg.ReadAt != nil || msg.Status == domain.MessageStatusRead
	}

	var nextCursor string
	if len(messages) == limit {
		nextCursor = messages[len(messages)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	Success(c, gin.H{"messages": messages, "nextCursor": nextCursor}, "Messages retrieved")
}

func (h *MessageHandler) DeleteConversation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid conversation ID")
		return
	}

	if err := h.conversationRepo.SoftDeleteFor(c.Request.Context(), convID, userID); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete conversation")
		Error(c, err)
		return
	}

	Success(c, nil, "Conversation deleted")
}

func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	convID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		BadRequest(c, "Invalid conversation ID")
		return
	}

	if err := h.messageRepo.MarkAsRead(c.Request.Context(), convID, userID); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark as read")
		Error(c, err)
		return
	}

	_ = h.conversationRepo.ResetUnread(c.Request.Context(), convID, userID)

	Success(c, nil, "Messages marked as read")
}

func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	msgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid message ID")
		return
	}

	if err := h.messageRepo.Delete(c.Request.Context(), msgID, userID); err != nil {
		h.logger.Error().Err(err).Msg("failed to delete message")
		Error(c, err)
		return
	}

	Success(c, nil, "Message deleted")
}

func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	count, err := h.conversationRepo.GetTotalUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get unread count")
		Error(c, err)
		return
	}

	Success(c, gin.H{"unreadCount": count}, "Unread count retrieved")
}
