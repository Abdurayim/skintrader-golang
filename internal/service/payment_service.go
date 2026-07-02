package service

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
)

// Payme JSON-RPC error codes
const (
	PaymeErrorInvalidAmount     = -31001
	PaymeErrorTransactionNotFound = -31003
	PaymeErrorCannotPerform     = -31008
	PaymeErrorCannotCancel      = -31007
	PaymeErrorMethodNotFound    = -32601
	PaymeErrorInvalidAuth       = -32504

	// Payme transaction states
	PaymeStateProcessing = 1
	PaymeStateCompleted  = 2
	PaymeStateCancelled  = -1
	PaymeStateRefunded   = -2

	// Payme transaction timeout (12 hours)
	PaymeTransactionTimeout = 12 * time.Hour

	// Click error codes
	ClickErrorSuccess        = 0
	ClickErrorSignFailed     = -1
	ClickErrorInvalidAmount  = -2
	ClickErrorActionNotFound = -3
	ClickErrorAlreadyPaid    = -4
	ClickErrorUserNotFound   = -5
	ClickErrorTransNotFound  = -6
	ClickErrorBadRequest     = -8
	ClickErrorCancelled      = -9
)

// PaymeRPCRequest represents a JSON-RPC 2.0 request from Payme.
type PaymeRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// PaymeRPCResponse represents a JSON-RPC 2.0 response to Payme.
type PaymeRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a JSON-RPC error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Data    string      `json:"data,omitempty"`
}

// PaymeCheckPerformParams contains params for CheckPerformTransaction.
type PaymeCheckPerformParams struct {
	Amount  int64 `json:"amount"`
	Account struct {
		UserID        string `json:"user_id"`
		TransactionID string `json:"transaction_id"`
	} `json:"account"`
}

// PaymeCreateTransactionParams contains params for CreateTransaction.
type PaymeCreateTransactionParams struct {
	ID      string `json:"id"`
	Time    int64  `json:"time"`
	Amount  int64  `json:"amount"`
	Account struct {
		UserID        string `json:"user_id"`
		TransactionID string `json:"transaction_id"`
	} `json:"account"`
}

// PaymePerformTransactionParams contains params for PerformTransaction.
type PaymePerformTransactionParams struct {
	ID string `json:"id"`
}

// PaymeCancelTransactionParams contains params for CancelTransaction.
type PaymeCancelTransactionParams struct {
	ID     string `json:"id"`
	Reason int    `json:"reason"`
}

// PaymeCheckTransactionParams contains params for CheckTransaction.
type PaymeCheckTransactionParams struct {
	ID string `json:"id"`
}

// PaymeGetStatementParams contains params for GetStatement.
type PaymeGetStatementParams struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

// ClickRequest represents parameters from Click PREPARE/COMPLETE.
type ClickRequest struct {
	ClickTransID  string `json:"click_trans_id" form:"click_trans_id"`
	ServiceID     string `json:"service_id" form:"service_id"`
	ClickPaydocID string `json:"click_paydoc_id" form:"click_paydoc_id"`
	MerchantTransID string `json:"merchant_trans_id" form:"merchant_trans_id"`
	Amount        string `json:"amount" form:"amount"`
	Action        string `json:"action" form:"action"`
	Error         string `json:"error" form:"error"`
	ErrorNote     string `json:"error_note" form:"error_note"`
	SignTime      string `json:"sign_time" form:"sign_time"`
	SignString    string `json:"sign_string" form:"sign_string"`
	MerchantPrepareID string `json:"merchant_prepare_id,omitempty" form:"merchant_prepare_id"`
}

// ClickResponse represents the response to Click.
type ClickResponse struct {
	ClickTransID      string `json:"click_trans_id"`
	MerchantTransID   string `json:"merchant_trans_id"`
	MerchantPrepareID string `json:"merchant_prepare_id,omitempty"`
	MerchantConfirmID string `json:"merchant_confirm_id,omitempty"`
	Error             int    `json:"error"`
	ErrorNote         string `json:"error_note"`
}

// XaznaWebhookPayload represents a webhook from Xazna.
type XaznaWebhookPayload struct {
	TransactionID string  `json:"transaction_id"`
	ExternalID    string  `json:"external_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // "created", "confirmed", "completed", "cancelled"
	Timestamp     int64   `json:"timestamp"`
}

// UzumWebhookPayload represents a webhook from Uzum Bank.
type UzumWebhookPayload struct {
	TransactionID string  `json:"transaction_id"`
	ExternalID    string  `json:"external_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // "created", "confirmed", "completed", "cancelled"
	Timestamp     int64   `json:"timestamp"`
}

type PaymentService struct {
	txRepo    domain.TransactionRepository
	subRepo   domain.SubscriptionRepository
	userRepo  domain.UserRepository
	paymeCfg  config.PaymeConfig
	clickCfg  config.ClickConfig
	xaznaCfg  config.XaznaConfig
	uzumCfg   config.UzumConfig
	subCfg    config.SubscriptionConfig
	logger    zerolog.Logger
}

func NewPaymentService(txRepo domain.TransactionRepository, subRepo domain.SubscriptionRepository, userRepo domain.UserRepository, paymentCfg config.PaymentConfig, subCfg config.SubscriptionConfig, logger zerolog.Logger) *PaymentService {
	return &PaymentService{
		txRepo:   txRepo,
		subRepo:  subRepo,
		userRepo: userRepo,
		paymeCfg: paymentCfg.Payme,
		clickCfg: paymentCfg.Click,
		xaznaCfg: paymentCfg.Xazna,
		uzumCfg:  paymentCfg.Uzum,
		subCfg:   subCfg,
		logger:   logger.With().Str("service", "payment").Logger(),
	}
}

// InitiatePayment creates a transaction and returns a payment URL.
func (s *PaymentService) InitiatePayment(ctx context.Context, userID uuid.UUID, method domain.PaymentMethod, currency domain.Currency) (string, *domain.Transaction, error) {
	// Determine amount based on currency
	var amount float64
	switch currency {
	case domain.CurrencyUZS:
		amount = float64(s.subCfg.PriceUZS)
	case domain.CurrencyUSD:
		amount = float64(s.subCfg.PriceUSD)
	default:
		return "", nil, fmt.Errorf("unsupported currency: %s", currency)
	}

	// Create transaction
	tx := &domain.Transaction{
		ID:            uuid.New(),
		UserID:        userID,
		Amount:        amount,
		Currency:      currency,
		Status:        domain.TransactionStatusPending,
		PaymentMethod: method,
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return "", nil, fmt.Errorf("create transaction: %w", err)
	}

	// Generate payment URL based on method
	var paymentURL string
	var err error

	switch method {
	case domain.PaymentMethodPayme:
		paymentURL, err = s.generatePaymeURL(tx)
	case domain.PaymentMethodClick:
		paymentURL, err = s.generateClickURL(tx)
	case domain.PaymentMethodXazna:
		paymentURL, err = s.generateXaznaURL(tx)
	case domain.PaymentMethodUzum:
		paymentURL, err = s.generateUzumURL(tx)
	default:
		return "", nil, fmt.Errorf("unsupported payment method: %s", method)
	}

	if err != nil {
		return "", nil, fmt.Errorf("generate payment URL: %w", err)
	}

	return paymentURL, tx, nil
}

// generatePaymeURL generates a Payme checkout URL with base64 encoded params.
func (s *PaymentService) generatePaymeURL(tx *domain.Transaction) (string, error) {
	if s.paymeCfg.MerchantID == "" {
		return "", fmt.Errorf("PayMe merchant ID not configured")
	}

	// Convert amount to tiyin (smallest unit): 1 UZS = 100 tiyin
	amountInTiyin := int64(tx.Amount * 100)

	// Currency code: 860 = UZS, 840 = USD
	currencyCode := 860
	if tx.Currency == domain.CurrencyUSD {
		currencyCode = 840
	}

	// Build merchant params
	merchantParams := map[string]interface{}{
		"merchant_id": s.paymeCfg.MerchantID,
		"account": map[string]string{
			"user_id":        tx.UserID.String(),
			"transaction_id": tx.ID.String(),
		},
		"amount":      amountInTiyin,
		"currency":    currencyCode,
		"callback":    s.paymeCfg.CallbackURL,
		"description": "Skintrader Monthly Subscription",
	}

	paramsJSON, err := json.Marshal(merchantParams)
	if err != nil {
		return "", fmt.Errorf("marshal merchant params: %w", err)
	}

	paramsBase64 := base64.StdEncoding.EncodeToString(paramsJSON)

	baseURL := "https://checkout.paycom.uz"
	if s.paymeCfg.TestMode {
		baseURL = "https://checkout.test.paycom.uz"
	}

	return fmt.Sprintf("%s/%s", baseURL, paramsBase64), nil
}

// generateClickURL generates a Click checkout URL.
func (s *PaymentService) generateClickURL(tx *domain.Transaction) (string, error) {
	if s.clickCfg.MerchantID == "" || s.clickCfg.ServiceID == "" {
		return "", fmt.Errorf("Click merchant or service ID not configured")
	}

	return fmt.Sprintf(
		"https://my.click.uz/services/pay?service_id=%s&merchant_id=%s&amount=%.2f&transaction_param=%s&return_url=%s",
		s.clickCfg.ServiceID,
		s.clickCfg.MerchantID,
		tx.Amount,
		tx.ID.String(),
		s.clickCfg.WebhookURL,
	), nil
}

// generateXaznaURL generates an Xazna checkout URL.
func (s *PaymentService) generateXaznaURL(tx *domain.Transaction) (string, error) {
	if s.xaznaCfg.BaseURL == "" || s.xaznaCfg.APIKey == "" {
		return "", fmt.Errorf("Xazna base URL or API key not configured")
	}

	return fmt.Sprintf(
		"%s/pay?api_key=%s&amount=%.2f&transaction_id=%s",
		s.xaznaCfg.BaseURL,
		s.xaznaCfg.APIKey,
		tx.Amount,
		tx.ID.String(),
	), nil
}

// generateUzumURL generates an Uzum Bank checkout URL.
func (s *PaymentService) generateUzumURL(tx *domain.Transaction) (string, error) {
	if s.uzumCfg.BaseURL == "" || s.uzumCfg.TerminalID == "" {
		return "", fmt.Errorf("Uzum base URL or terminal ID not configured")
	}

	return fmt.Sprintf(
		"%s/pay?terminal_id=%s&amount=%.2f&transaction_id=%s",
		s.uzumCfg.BaseURL,
		s.uzumCfg.TerminalID,
		tx.Amount,
		tx.ID.String(),
	), nil
}

// VerifyPaymeAuth checks the Basic Auth header for Payme webhook.
// The expected format is "Basic base64(merchant_id:secret_key)"
func (s *PaymentService) VerifyPaymeAuth(authHeader string) bool {
	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		return false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	expectedKey := s.paymeCfg.SecretKey
	if s.paymeCfg.TestMode {
		// In test mode, Payme sends test key
		expectedKey = s.paymeCfg.SecretKey
	}

	return parts[0] == s.paymeCfg.MerchantID && parts[1] == expectedKey
}

// HandlePaymeWebhook handles PayMe JSON-RPC 2.0 webhook.
func (s *PaymentService) HandlePaymeWebhook(ctx context.Context, authHeader string, payload []byte) *PaymeRPCResponse {
	var req PaymeRPCRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      0,
			Error: &RPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	// Verify Basic Auth
	if !s.VerifyPaymeAuth(authHeader) {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorInvalidAuth,
				Message: "Authentication failed",
			},
		}
	}

	s.logger.Info().Str("method", req.Method).Msg("processing Payme webhook")

	switch req.Method {
	case "CheckPerformTransaction":
		return s.paymeCheckPerformTransaction(ctx, req)
	case "CreateTransaction":
		return s.paymeCreateTransaction(ctx, req)
	case "PerformTransaction":
		return s.paymePerformTransaction(ctx, req)
	case "CancelTransaction":
		return s.paymeCancelTransaction(ctx, req)
	case "CheckTransaction":
		return s.paymeCheckTransaction(ctx, req)
	case "GetStatement":
		return s.paymeGetStatement(ctx, req)
	default:
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorMethodNotFound,
				Message: "Method not found",
			},
		}
	}
}

// paymeCheckPerformTransaction checks if a transaction can be performed.
func (s *PaymentService) paymeCheckPerformTransaction(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymeCheckPerformParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	// Check if user exists
	userID, err := uuid.Parse(params.Account.UserID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "User not found",
			},
		}
	}

	_, err = s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "User not found",
			},
		}
	}

	// Check transaction if provided
	if params.Account.TransactionID != "" {
		txID, err := uuid.Parse(params.Account.TransactionID)
		if err != nil {
			return &PaymeRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    PaymeErrorTransactionNotFound,
					Message: "Invalid transaction ID",
				},
			}
		}

		tx, err := s.txRepo.FindByID(ctx, txID)
		if err != nil {
			return &PaymeRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    PaymeErrorTransactionNotFound,
					Message: "Transaction not found",
				},
			}
		}

		if tx.Status != domain.TransactionStatusPending {
			return &PaymeRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    PaymeErrorCannotPerform,
					Message: "Transaction already processed",
				},
			}
		}

		// Validate amount (Payme sends in tiyin)
		expectedTiyin := int64(tx.Amount * 100)
		if params.Amount != expectedTiyin {
			return &PaymeRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    PaymeErrorInvalidAmount,
					Message: "Incorrect amount",
				},
			}
		}
	}

	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"allow": true,
		},
	}
}

// paymeCreateTransaction creates a transaction in the Payme system.
func (s *PaymentService) paymeCreateTransaction(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymeCreateTransactionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	// Check if Payme transaction already exists by external ID
	existingTx, err := s.txRepo.FindByExternalID(ctx, params.ID)
	if err == nil && existingTx != nil {
		// Transaction already exists
		state := PaymeStateProcessing
		if existingTx.Status == domain.TransactionStatusCompleted {
			state = PaymeStateCompleted
		} else if existingTx.Status == domain.TransactionStatusCancelled {
			state = PaymeStateCancelled
		}

		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"create_time": existingTx.CreatedAt.UnixMilli(),
				"transaction": existingTx.ID.String(),
				"state":       state,
			},
		}
	}

	// Parse our internal transaction ID
	txID, err := uuid.Parse(params.Account.TransactionID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "Invalid transaction ID",
			},
		}
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "Transaction not found",
			},
		}
	}

	// Check if transaction is still pending
	if tx.Status != domain.TransactionStatusPending {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotPerform,
				Message: "Transaction already processed",
			},
		}
	}

	// Check if transaction has timed out
	if time.Since(tx.CreatedAt) > PaymeTransactionTimeout {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotPerform,
				Message: "Transaction has expired",
			},
		}
	}

	// Validate amount
	expectedTiyin := int64(tx.Amount * 100)
	if params.Amount != expectedTiyin {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorInvalidAmount,
				Message: "Incorrect amount",
			},
		}
	}

	// Update transaction: set external ID and status to processing
	externalID := params.ID
	tx.ExternalTransactionID = &externalID
	tx.Status = domain.TransactionStatusProcessing
	tx.WebhookReceived = true
	webhookTime := time.UnixMilli(params.Time)
	tx.WebhookReceivedAt = &webhookTime

	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to update transaction for Payme CreateTransaction")
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotPerform,
				Message: "Internal error",
			},
		}
	}

	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"create_time": params.Time,
			"transaction": tx.ID.String(),
			"state":       PaymeStateProcessing,
		},
	}
}

// paymePerformTransaction marks a transaction as completed.
func (s *PaymentService) paymePerformTransaction(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymePerformTransactionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	tx, err := s.txRepo.FindByExternalID(ctx, params.ID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "Transaction not found",
			},
		}
	}

	// If already completed, return idempotent result
	if tx.Status == domain.TransactionStatusCompleted {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"transaction":  tx.ID.String(),
				"perform_time": tx.UpdatedAt.UnixMilli(),
				"state":        PaymeStateCompleted,
			},
		}
	}

	// Can only perform processing transactions
	if tx.Status != domain.TransactionStatusProcessing {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotPerform,
				Message: "Cannot perform transaction in current state",
			},
		}
	}

	// Mark as completed
	tx.Status = domain.TransactionStatusCompleted
	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to update transaction for Payme PerformTransaction")
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotPerform,
				Message: "Internal error",
			},
		}
	}

	// Activate subscription
	s.activateSubscription(ctx, tx)

	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"transaction":  tx.ID.String(),
			"perform_time": time.Now().UnixMilli(),
			"state":        PaymeStateCompleted,
		},
	}
}

// paymeCancelTransaction cancels a transaction.
func (s *PaymentService) paymeCancelTransaction(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymeCancelTransactionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	tx, err := s.txRepo.FindByExternalID(ctx, params.ID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "Transaction not found",
			},
		}
	}

	// Already cancelled
	if tx.Status == domain.TransactionStatusCancelled {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"transaction": tx.ID.String(),
				"cancel_time": tx.UpdatedAt.UnixMilli(),
				"state":       PaymeStateCancelled,
			},
		}
	}

	// Cannot cancel completed transactions via Payme cancel (must refund)
	if tx.Status == domain.TransactionStatusCompleted {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotCancel,
				Message: "Cannot cancel completed transaction",
			},
		}
	}

	// Cancel the transaction
	tx.Status = domain.TransactionStatusCancelled
	errMsg := fmt.Sprintf("Cancelled by Payme, reason: %d", params.Reason)
	tx.ErrorMessage = &errMsg

	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to update transaction for Payme CancelTransaction")
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorCannotCancel,
				Message: "Internal error",
			},
		}
	}

	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"transaction": tx.ID.String(),
			"cancel_time": time.Now().UnixMilli(),
			"state":       PaymeStateCancelled,
		},
	}
}

// paymeCheckTransaction checks the current state of a transaction.
func (s *PaymentService) paymeCheckTransaction(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymeCheckTransactionParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	tx, err := s.txRepo.FindByExternalID(ctx, params.ID)
	if err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    PaymeErrorTransactionNotFound,
				Message: "Transaction not found",
			},
		}
	}

	state := s.transactionStatusToPaymeState(tx.Status)

	var performTime int64
	if tx.Status == domain.TransactionStatusCompleted {
		performTime = tx.UpdatedAt.UnixMilli()
	}

	var cancelTime int64
	if tx.Status == domain.TransactionStatusCancelled || tx.Status == domain.TransactionStatusRefunded {
		cancelTime = tx.UpdatedAt.UnixMilli()
	}

	var reason interface{}
	if tx.ErrorMessage != nil {
		reason = *tx.ErrorMessage
	}

	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"create_time":  tx.CreatedAt.UnixMilli(),
			"perform_time": performTime,
			"cancel_time":  cancelTime,
			"transaction":  tx.ID.String(),
			"state":        state,
			"reason":       reason,
		},
	}
}

// paymeGetStatement returns transaction list for a time period.
func (s *PaymentService) paymeGetStatement(ctx context.Context, req PaymeRPCRequest) *PaymeRPCResponse {
	var params PaymeGetStatementParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &PaymeRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32600,
				Message: "Invalid params",
			},
		}
	}

	// GetStatement is typically used for reconciliation.
	// Return an empty list; a full implementation would query by date range.
	return &PaymeRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"transactions": []interface{}{},
		},
	}
}

// transactionStatusToPaymeState converts internal status to Payme state.
func (s *PaymentService) transactionStatusToPaymeState(status domain.TransactionStatus) int {
	switch status {
	case domain.TransactionStatusPending:
		return 0
	case domain.TransactionStatusProcessing:
		return PaymeStateProcessing
	case domain.TransactionStatusCompleted:
		return PaymeStateCompleted
	case domain.TransactionStatusCancelled, domain.TransactionStatusFailed:
		return PaymeStateCancelled
	case domain.TransactionStatusRefunded:
		return PaymeStateRefunded
	default:
		return 0
	}
}

// activateSubscription creates a subscription after successful payment.
func (s *PaymentService) activateSubscription(ctx context.Context, tx *domain.Transaction) {
	if tx.SubscriptionID != nil {
		// Already has a subscription linked, skip
		return
	}

	now := time.Now()
	endDate := now.AddDate(0, 0, s.subCfg.DurationDays)

	sub := &domain.Subscription{
		ID:            uuid.New(),
		UserID:        tx.UserID,
		Status:        domain.SubscriptionStatusActive,
		Plan:          domain.SubscriptionPlanMonthly,
		StartDate:     &now,
		EndDate:       &endDate,
		AutoRenew:     false,
		LastPaymentID: &tx.ID,
	}

	if err := s.subRepo.Create(ctx, sub); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to create subscription after payment")
		return
	}

	// Link subscription to transaction
	tx.SubscriptionID = &sub.ID
	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to link subscription to transaction")
	}

	// Update user subscription status
	if err := s.userRepo.UpdateSubscriptionStatus(ctx, tx.UserID, domain.SubscriptionStatusActive, &sub.ID, &endDate); err != nil {
		s.logger.Error().Err(err).Str("user_id", tx.UserID.String()).Msg("failed to update user subscription status")
	}

	s.logger.Info().
		Str("user_id", tx.UserID.String()).
		Str("subscription_id", sub.ID.String()).
		Str("tx_id", tx.ID.String()).
		Time("end_date", endDate).
		Msg("subscription activated after payment")
}

// VerifyClickSign verifies the Click SIGN hash.
// SIGN = MD5(click_trans_id + service_id + secret_key + merchant_trans_id + amount + action + sign_time)
func (s *PaymentService) VerifyClickSign(req *ClickRequest) bool {
	signStr := fmt.Sprintf("%s%s%s%s%s%s%s",
		req.ClickTransID,
		req.ServiceID,
		s.clickCfg.SecretKey,
		req.MerchantTransID,
		req.Amount,
		req.Action,
		req.SignTime,
	)

	hash := md5.Sum([]byte(signStr))
	expectedSign := hex.EncodeToString(hash[:])

	return hmac.Equal([]byte(expectedSign), []byte(req.SignString))
}

// HandleClickPrepare handles Click PREPARE callback.
func (s *PaymentService) HandleClickPrepare(ctx context.Context, req *ClickRequest) *ClickResponse {
	resp := &ClickResponse{
		ClickTransID:    req.ClickTransID,
		MerchantTransID: req.MerchantTransID,
	}

	// Verify SIGN
	if !s.VerifyClickSign(req) {
		resp.Error = ClickErrorSignFailed
		resp.ErrorNote = "SIGN check failed"
		return resp
	}

	// Check if there's an error from Click
	if req.Error != "" && req.Error != "0" {
		resp.Error = ClickErrorCancelled
		resp.ErrorNote = "Transaction cancelled"
		return resp
	}

	// Find the transaction
	txID, err := uuid.Parse(req.MerchantTransID)
	if err != nil {
		resp.Error = ClickErrorTransNotFound
		resp.ErrorNote = "Invalid transaction ID"
		return resp
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		resp.Error = ClickErrorTransNotFound
		resp.ErrorNote = "Transaction not found"
		return resp
	}

	// Check user
	_, err = s.userRepo.FindByID(ctx, tx.UserID)
	if err != nil {
		resp.Error = ClickErrorUserNotFound
		resp.ErrorNote = "User not found"
		return resp
	}

	// Check transaction status
	if tx.Status != domain.TransactionStatusPending {
		if tx.Status == domain.TransactionStatusCancelled {
			resp.Error = ClickErrorCancelled
			resp.ErrorNote = "Transaction cancelled"
		} else {
			resp.Error = ClickErrorAlreadyPaid
			resp.ErrorNote = "Transaction already paid"
		}
		return resp
	}

	// Update transaction to processing
	externalID := req.ClickTransID
	tx.ExternalTransactionID = &externalID
	tx.Status = domain.TransactionStatusProcessing
	tx.WebhookReceived = true
	now := time.Now()
	tx.WebhookReceivedAt = &now

	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to update transaction for Click prepare")
		resp.Error = ClickErrorBadRequest
		resp.ErrorNote = "Internal error"
		return resp
	}

	resp.Error = ClickErrorSuccess
	resp.ErrorNote = "Success"
	resp.MerchantPrepareID = tx.ID.String()
	return resp
}

// HandleClickComplete handles Click COMPLETE callback.
func (s *PaymentService) HandleClickComplete(ctx context.Context, req *ClickRequest) *ClickResponse {
	resp := &ClickResponse{
		ClickTransID:    req.ClickTransID,
		MerchantTransID: req.MerchantTransID,
	}

	// Verify SIGN
	if !s.VerifyClickSign(req) {
		resp.Error = ClickErrorSignFailed
		resp.ErrorNote = "SIGN check failed"
		return resp
	}

	// Check if there's an error from Click
	if req.Error != "" && req.Error != "0" {
		// Click is reporting an error, cancel the transaction
		txID, parseErr := uuid.Parse(req.MerchantTransID)
		if parseErr == nil {
			if tx, findErr := s.txRepo.FindByID(ctx, txID); findErr == nil {
				tx.Status = domain.TransactionStatusCancelled
				errMsg := fmt.Sprintf("Click error: %s - %s", req.Error, req.ErrorNote)
				tx.ErrorMessage = &errMsg
				_ = s.txRepo.Update(ctx, tx)
			}
		}
		resp.Error = ClickErrorCancelled
		resp.ErrorNote = "Transaction cancelled by Click"
		return resp
	}

	// Find the transaction
	txID, err := uuid.Parse(req.MerchantTransID)
	if err != nil {
		resp.Error = ClickErrorTransNotFound
		resp.ErrorNote = "Invalid transaction ID"
		return resp
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		resp.Error = ClickErrorTransNotFound
		resp.ErrorNote = "Transaction not found"
		return resp
	}

	// Check transaction status
	if tx.Status == domain.TransactionStatusCompleted {
		resp.Error = ClickErrorSuccess
		resp.ErrorNote = "Already completed"
		resp.MerchantConfirmID = tx.ID.String()
		return resp
	}

	if tx.Status == domain.TransactionStatusCancelled {
		resp.Error = ClickErrorCancelled
		resp.ErrorNote = "Transaction cancelled"
		return resp
	}

	if tx.Status != domain.TransactionStatusProcessing {
		resp.Error = ClickErrorBadRequest
		resp.ErrorNote = "Invalid transaction state"
		return resp
	}

	// Mark as completed
	tx.Status = domain.TransactionStatusCompleted
	if err := s.txRepo.Update(ctx, tx); err != nil {
		s.logger.Error().Err(err).Str("tx_id", tx.ID.String()).Msg("failed to update transaction for Click complete")
		resp.Error = ClickErrorBadRequest
		resp.ErrorNote = "Internal error"
		return resp
	}

	// Activate subscription
	s.activateSubscription(ctx, tx)

	resp.Error = ClickErrorSuccess
	resp.ErrorNote = "Success"
	resp.MerchantConfirmID = tx.ID.String()
	return resp
}

// VerifyXaznaSignature verifies the HMAC-SHA256 signature for Xazna webhooks.
func (s *PaymentService) VerifyXaznaSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.xaznaCfg.SecretKey))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// HandleXaznaWebhook handles Xazna webhook.
func (s *PaymentService) HandleXaznaWebhook(ctx context.Context, signature string, payload []byte) error {
	// Verify signature
	if !s.VerifyXaznaSignature(payload, signature) {
		return fmt.Errorf("invalid signature")
	}

	var webhookPayload XaznaWebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	s.logger.Info().
		Str("external_id", webhookPayload.ExternalID).
		Str("status", webhookPayload.Status).
		Msg("processing Xazna webhook")

	// Find internal transaction
	txID, err := uuid.Parse(webhookPayload.ExternalID)
	if err != nil {
		return fmt.Errorf("invalid external ID: %w", err)
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Store external transaction ID
	externalID := webhookPayload.TransactionID
	tx.ExternalTransactionID = &externalID
	tx.WebhookReceived = true
	now := time.Now()
	tx.WebhookReceivedAt = &now

	switch webhookPayload.Status {
	case "created":
		if tx.Status == domain.TransactionStatusPending {
			tx.Status = domain.TransactionStatusProcessing
		}
	case "confirmed":
		if tx.Status == domain.TransactionStatusPending || tx.Status == domain.TransactionStatusProcessing {
			tx.Status = domain.TransactionStatusProcessing
		}
	case "completed":
		if tx.Status != domain.TransactionStatusCompleted {
			tx.Status = domain.TransactionStatusCompleted
			if err := s.txRepo.Update(ctx, tx); err != nil {
				return fmt.Errorf("update transaction: %w", err)
			}
			s.activateSubscription(ctx, tx)
			return nil
		}
	case "cancelled":
		tx.Status = domain.TransactionStatusCancelled
		errMsg := "Cancelled by Xazna"
		tx.ErrorMessage = &errMsg
	default:
		return fmt.Errorf("unknown status: %s", webhookPayload.Status)
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return fmt.Errorf("update transaction: %w", err)
	}

	return nil
}

// VerifyUzumSignature verifies the HMAC-SHA256 signature for Uzum Bank webhooks.
func (s *PaymentService) VerifyUzumSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.uzumCfg.SecretKey))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}

// HandleUzumWebhook handles Uzum Bank webhook.
func (s *PaymentService) HandleUzumWebhook(ctx context.Context, signature string, payload []byte) error {
	// Verify signature
	if !s.VerifyUzumSignature(payload, signature) {
		return fmt.Errorf("invalid signature")
	}

	var webhookPayload UzumWebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	s.logger.Info().
		Str("external_id", webhookPayload.ExternalID).
		Str("status", webhookPayload.Status).
		Msg("processing Uzum webhook")

	// Find internal transaction
	txID, err := uuid.Parse(webhookPayload.ExternalID)
	if err != nil {
		return fmt.Errorf("invalid external ID: %w", err)
	}

	tx, err := s.txRepo.FindByID(ctx, txID)
	if err != nil {
		return fmt.Errorf("transaction not found: %w", err)
	}

	// Store external transaction ID
	externalID := webhookPayload.TransactionID
	tx.ExternalTransactionID = &externalID
	tx.WebhookReceived = true
	now := time.Now()
	tx.WebhookReceivedAt = &now

	switch webhookPayload.Status {
	case "created":
		if tx.Status == domain.TransactionStatusPending {
			tx.Status = domain.TransactionStatusProcessing
		}
	case "confirmed":
		if tx.Status == domain.TransactionStatusPending || tx.Status == domain.TransactionStatusProcessing {
			tx.Status = domain.TransactionStatusProcessing
		}
	case "completed":
		if tx.Status != domain.TransactionStatusCompleted {
			tx.Status = domain.TransactionStatusCompleted
			if err := s.txRepo.Update(ctx, tx); err != nil {
				return fmt.Errorf("update transaction: %w", err)
			}
			s.activateSubscription(ctx, tx)
			return nil
		}
	case "cancelled":
		tx.Status = domain.TransactionStatusCancelled
		errMsg := "Cancelled by Uzum"
		tx.ErrorMessage = &errMsg
	default:
		return fmt.Errorf("unknown status: %s", webhookPayload.Status)
	}

	if err := s.txRepo.Update(ctx, tx); err != nil {
		return fmt.Errorf("update transaction: %w", err)
	}

	return nil
}

// GetUserTransactions returns transaction history for a user.
func (s *PaymentService) GetUserTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, int64, error) {
	return s.txRepo.FindByUser(ctx, userID, limit, offset)
}

// GetTransactionByID returns a specific transaction.
func (s *PaymentService) GetTransactionByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	return s.txRepo.FindByID(ctx, id)
}
