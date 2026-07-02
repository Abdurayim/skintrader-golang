package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	"skintrader-go/internal/service"
)

type SubscriptionHandler struct {
	subscriptionSvc *service.SubscriptionService
	paymentSvc      *service.PaymentService
	logger          zerolog.Logger
}

func NewSubscriptionHandler(subscriptionSvc *service.SubscriptionService, paymentSvc *service.PaymentService, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionSvc: subscriptionSvc,
		paymentSvc:      paymentSvc,
		logger:          logger.With().Str("handler", "subscription").Logger(),
	}
}

func (h *SubscriptionHandler) GetStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	sub, err := h.subscriptionSvc.GetStatus(c.Request.Context(), userID)
	if err != nil {
		Success(c, gin.H{
			"subscription":          nil,
			"status":                domain.SubscriptionStatusNone,
			"subscriptionStatus":    domain.SubscriptionStatusNone,
			"hasActiveSubscription": false,
		}, "No active subscription")
		return
	}

	Success(c, gin.H{
		"subscription":          sub,
		"status":                sub.Status,
		"subscriptionStatus":    sub.Status,
		"hasActiveSubscription": sub.Status == domain.SubscriptionStatusActive,
		"subscriptionExpiresAt": sub.EndDate,
	}, "Subscription status retrieved")
}

func (h *SubscriptionHandler) Initiate(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		PaymentMethod string `json:"paymentMethod"`
		Currency      string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body — defaults will apply
		req.Currency = ""
		req.PaymentMethod = ""
	}

	if req.PaymentMethod == "" {
		req.PaymentMethod = string(domain.PaymentMethodPayme)
	}

	method := domain.PaymentMethod(req.PaymentMethod)
	switch method {
	case domain.PaymentMethodPayme, domain.PaymentMethodClick, domain.PaymentMethodXazna, domain.PaymentMethodUzum:
	default:
		BadRequest(c, "Invalid payment method. Use: payme, click, xazna, uzum")
		return
	}

	currency := domain.CurrencyUZS
	if req.Currency != "" {
		currency = domain.Currency(req.Currency)
	}

	paymentURL, tx, err := h.paymentSvc.InitiatePayment(c.Request.Context(), userID, method, currency)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to initiate payment")
		Error(c, err)
		return
	}

	Created(c, gin.H{
		"paymentUrl":    paymentURL,
		"transactionId": tx.ID,
		"amount":         tx.Amount,
		"currency":       tx.Currency,
		"paymentMethod": tx.PaymentMethod,
	}, "Payment initiated")
}

func (h *SubscriptionHandler) GetHistory(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	subs, err := h.subscriptionSvc.GetHistory(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get subscription history")
		Error(c, err)
		return
	}

	Success(c, gin.H{"subscriptions": subs}, "Subscription history retrieved")
}

func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	sub, err := h.subscriptionSvc.GetStatus(c.Request.Context(), userID)
	if err != nil {
		BadRequest(c, "No active subscription to cancel")
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "User requested cancellation"
	}

	if err := h.subscriptionSvc.Cancel(c.Request.Context(), sub.ID, reason); err != nil {
		h.logger.Error().Err(err).Msg("failed to cancel subscription")
		Error(c, err)
		return
	}

	Success(c, nil, "Subscription cancelled")
}
