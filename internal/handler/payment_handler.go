package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/middleware"
	"skintrader-go/internal/service"
)

type PaymentHandler struct {
	paymentSvc *service.PaymentService
	logger     zerolog.Logger
}

func NewPaymentHandler(paymentSvc *service.PaymentService, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentSvc: paymentSvc,
		logger:     logger.With().Str("handler", "payment").Logger(),
	}
}

func (h *PaymentHandler) HandlePaymeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
		return
	}

	authHeader := c.GetHeader("Authorization")
	resp := h.paymentSvc.HandlePaymeWebhook(c.Request.Context(), authHeader, body)
	c.JSON(http.StatusOK, resp)
}

func (h *PaymentHandler) HandlePaymeCallback(c *gin.Context) {
	// Payme redirects user here after payment.
	// Simply acknowledge and let the frontend handle the redirect.
	Success(c, gin.H{"status": "callback_received"}, "Payment callback received")
}

func (h *PaymentHandler) HandleClickPrepare(c *gin.Context) {
	var req service.ClickRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusOK, &service.ClickResponse{
			Error:     -8,
			ErrorNote: "Invalid request",
		})
		return
	}

	resp := h.paymentSvc.HandleClickPrepare(c.Request.Context(), &req)
	c.JSON(http.StatusOK, resp)
}

func (h *PaymentHandler) HandleClickComplete(c *gin.Context) {
	var req service.ClickRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusOK, &service.ClickResponse{
			Error:     -8,
			ErrorNote: "Invalid request",
		})
		return
	}

	resp := h.paymentSvc.HandleClickComplete(c.Request.Context(), &req)
	c.JSON(http.StatusOK, resp)
}

func (h *PaymentHandler) HandleXaznaWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
		return
	}

	signature := c.GetHeader("X-Signature")
	if err := h.paymentSvc.HandleXaznaWebhook(c.Request.Context(), signature, body); err != nil {
		h.logger.Error().Err(err).Msg("xazna webhook error")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook processing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PaymentHandler) HandleUzumWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
		return
	}

	signature := c.GetHeader("X-Signature")
	if err := h.paymentSvc.HandleUzumWebhook(c.Request.Context(), signature, body); err != nil {
		h.logger.Error().Err(err).Msg("uzum webhook error")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook processing failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PaymentHandler) GetTransactions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	transactions, total, err := h.paymentSvc.GetUserTransactions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get transactions")
		Error(c, err)
		return
	}

	Paginated(c, gin.H{"transactions": transactions}, gin.H{"total": total, "limit": limit, "offset": offset}, "Transactions retrieved")
}

func (h *PaymentHandler) GetTransactionByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	txID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid transaction ID")
		return
	}

	tx, err := h.paymentSvc.GetTransactionByID(c.Request.Context(), txID)
	if err != nil {
		Error(c, err)
		return
	}

	// Verify the transaction belongs to the requesting user
	if tx.UserID != userID {
		Forbidden(c, "Not your transaction")
		return
	}

	Success(c, gin.H{"transaction": tx}, "Transaction retrieved")
}
