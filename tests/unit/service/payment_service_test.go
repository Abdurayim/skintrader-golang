package service_test

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	"skintrader-go/internal/service"
	"skintrader-go/tests/mocks"
	"skintrader-go/tests/testutil"
)

const (
	testPaymeMerchantID = "test-merchant-id"
	testPaymeSecretKey  = "test-secret-key"
	testClickSecretKey  = "click-secret-key"
	testClickServiceID  = "click-service-id"
	testClickMerchantID = "click-merchant-id"
)

func newTestPaymentService(
	txRepo *mocks.MockTransactionRepository,
	subRepo *mocks.MockSubscriptionRepository,
	userRepo *mocks.MockUserRepository,
) *service.PaymentService {
	paymentCfg := config.PaymentConfig{
		Payme: config.PaymeConfig{
			MerchantID:  testPaymeMerchantID,
			SecretKey:   testPaymeSecretKey,
			CallbackURL: "https://example.com/callback",
			WebhookURL:  "https://example.com/webhook",
			TestMode:    true,
		},
		Click: config.ClickConfig{
			MerchantID: testClickMerchantID,
			ServiceID:  testClickServiceID,
			SecretKey:  testClickSecretKey,
			WebhookURL: "https://example.com/click/webhook",
		},
		Xazna: config.XaznaConfig{
			BaseURL:   "https://xazna.example.com",
			APIKey:    "xazna-api-key",
			SecretKey: "xazna-secret-key",
		},
		Uzum: config.UzumConfig{
			BaseURL:    "https://uzum.example.com",
			TerminalID: "uzum-terminal-id",
			SecretKey:  "uzum-secret-key",
		},
	}
	subCfg := config.SubscriptionConfig{
		PriceUZS:     10000,
		PriceUSD:     1,
		DurationDays: 30,
		GraceDays:    3,
	}
	logger := zerolog.Nop()
	return service.NewPaymentService(txRepo, subRepo, userRepo, paymentCfg, subCfg, logger)
}

// ---------------------------------------------------------------------------
// InitiatePayment tests
// ---------------------------------------------------------------------------

func TestInitiatePayment_Payme(t *testing.T) {
	userID := uuid.New()
	var createdTx *domain.Transaction

	txRepo := &mocks.MockTransactionRepository{
		CreateFn: func(ctx context.Context, transaction *domain.Transaction) error {
			createdTx = transaction
			return nil
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	url, tx, err := svc.InitiatePayment(context.Background(), userID, domain.PaymentMethodPayme, domain.CurrencyUZS)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, tx)
	if url == "" {
		t.Fatal("expected non-empty payment URL")
	}
	testutil.AssertNotNil(t, createdTx)
	testutil.AssertEqual(t, createdTx.UserID, userID)
	testutil.AssertEqual(t, createdTx.PaymentMethod, domain.PaymentMethodPayme)
	testutil.AssertEqual(t, createdTx.Status, domain.TransactionStatusPending)
	testutil.AssertEqual(t, createdTx.Amount, float64(10000))
	testutil.AssertEqual(t, createdTx.Currency, domain.CurrencyUZS)
}

func TestInitiatePayment_Click(t *testing.T) {
	userID := uuid.New()

	txRepo := &mocks.MockTransactionRepository{
		CreateFn: func(ctx context.Context, transaction *domain.Transaction) error {
			return nil
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	url, tx, err := svc.InitiatePayment(context.Background(), userID, domain.PaymentMethodClick, domain.CurrencyUZS)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, tx)
	if url == "" {
		t.Fatal("expected non-empty payment URL for Click")
	}
}

func TestInitiatePayment_InvalidMethod(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{
		CreateFn: func(ctx context.Context, transaction *domain.Transaction) error {
			return nil
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	url, tx, err := svc.InitiatePayment(context.Background(), uuid.New(), domain.PaymentMethod("invalid"), domain.CurrencyUZS)
	testutil.AssertError(t, err)
	if url != "" {
		t.Fatal("expected empty URL for invalid payment method")
	}
	if tx != nil {
		t.Fatal("expected nil transaction for invalid payment method")
	}
}

func TestInitiatePayment_USDCurrency(t *testing.T) {
	var createdTx *domain.Transaction

	txRepo := &mocks.MockTransactionRepository{
		CreateFn: func(ctx context.Context, transaction *domain.Transaction) error {
			createdTx = transaction
			return nil
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	url, tx, err := svc.InitiatePayment(context.Background(), uuid.New(), domain.PaymentMethodPayme, domain.CurrencyUSD)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, tx)
	if url == "" {
		t.Fatal("expected non-empty URL")
	}
	testutil.AssertEqual(t, createdTx.Amount, float64(1))
	testutil.AssertEqual(t, createdTx.Currency, domain.CurrencyUSD)
}

func TestInitiatePayment_InvalidCurrency(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	url, tx, err := svc.InitiatePayment(context.Background(), uuid.New(), domain.PaymentMethodPayme, domain.Currency("EUR"))
	testutil.AssertError(t, err)
	if url != "" {
		t.Fatal("expected empty URL for unsupported currency")
	}
	if tx != nil {
		t.Fatal("expected nil tx for unsupported currency")
	}
}

// ---------------------------------------------------------------------------
// VerifyPaymeAuth tests
// ---------------------------------------------------------------------------

func TestVerifyPaymeAuth_Valid(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	// Encode "test-merchant-id:test-secret-key" in Base64.
	credentials := base64.StdEncoding.EncodeToString([]byte(testPaymeMerchantID + ":" + testPaymeSecretKey))
	authHeader := "Basic " + credentials

	result := svc.VerifyPaymeAuth(authHeader)
	testutil.AssertTrue(t, result)
}

func TestVerifyPaymeAuth_Invalid(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	// Wrong credentials.
	credentials := base64.StdEncoding.EncodeToString([]byte("wrong-id:wrong-key"))
	authHeader := "Basic " + credentials

	result := svc.VerifyPaymeAuth(authHeader)
	testutil.AssertFalse(t, result)
}

func TestVerifyPaymeAuth_NoPrefix(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	result := svc.VerifyPaymeAuth("not-basic-auth")
	testutil.AssertFalse(t, result)
}

func TestVerifyPaymeAuth_MalformedBase64(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	result := svc.VerifyPaymeAuth("Basic not-valid-base64!@#$")
	testutil.AssertFalse(t, result)
}

// ---------------------------------------------------------------------------
// VerifyClickSign tests
// ---------------------------------------------------------------------------

func TestVerifyClickSign_Valid(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	clickReq := &service.ClickRequest{
		ClickTransID:    "12345",
		ServiceID:       testClickServiceID,
		MerchantTransID: uuid.New().String(),
		Amount:          "10000",
		Action:          "0",
		SignTime:        "2026-03-15 10:00:00",
	}

	// Compute expected sign: MD5(click_trans_id + service_id + secret_key + merchant_trans_id + amount + action + sign_time)
	signStr := fmt.Sprintf("%s%s%s%s%s%s%s",
		clickReq.ClickTransID,
		clickReq.ServiceID,
		testClickSecretKey,
		clickReq.MerchantTransID,
		clickReq.Amount,
		clickReq.Action,
		clickReq.SignTime,
	)
	h := md5.Sum([]byte(signStr))
	clickReq.SignString = hex.EncodeToString(h[:])

	result := svc.VerifyClickSign(clickReq)
	testutil.AssertTrue(t, result)
}

func TestVerifyClickSign_Invalid(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	clickReq := &service.ClickRequest{
		ClickTransID:    "12345",
		ServiceID:       testClickServiceID,
		MerchantTransID: uuid.New().String(),
		Amount:          "10000",
		Action:          "0",
		SignTime:        "2026-03-15 10:00:00",
		SignString:      "invalid-sign-hash",
	}

	result := svc.VerifyClickSign(clickReq)
	testutil.AssertFalse(t, result)
}

// ---------------------------------------------------------------------------
// HandlePaymeWebhook tests
// ---------------------------------------------------------------------------

func TestHandlePaymeWebhook_CheckPerform(t *testing.T) {
	user := testutil.NewTestUser()
	tx := testutil.NewTestTransaction(user.ID)

	txRepo := &mocks.MockTransactionRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			if id == tx.ID {
				return tx, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	// Build a CheckPerformTransaction request.
	params := map[string]interface{}{
		"amount": int64(tx.Amount * 100), // tiyin
		"account": map[string]string{
			"user_id":        user.ID.String(),
			"transaction_id": tx.ID.String(),
		},
	}
	paramsJSON, _ := json.Marshal(params)
	rpcReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "CheckPerformTransaction",
		"params":  json.RawMessage(paramsJSON),
	}
	payload, _ := json.Marshal(rpcReq)

	// Valid auth header.
	credentials := base64.StdEncoding.EncodeToString([]byte(testPaymeMerchantID + ":" + testPaymeSecretKey))
	authHeader := "Basic " + credentials

	resp := svc.HandlePaymeWebhook(context.Background(), authHeader, payload)
	if resp.Error != nil {
		t.Fatalf("expected no error, got code=%d message=%v", resp.Error.Code, resp.Error.Message)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestHandlePaymeWebhook_InvalidAuth(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	rpcReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "CheckPerformTransaction",
		"params":  json.RawMessage(`{}`),
	}
	payload, _ := json.Marshal(rpcReq)

	// Wrong auth header.
	credentials := base64.StdEncoding.EncodeToString([]byte("wrong:credentials"))
	authHeader := "Basic " + credentials

	resp := svc.HandlePaymeWebhook(context.Background(), authHeader, payload)
	if resp.Error == nil {
		t.Fatal("expected error for invalid auth")
	}
	testutil.AssertEqual(t, resp.Error.Code, -32504) // PaymeErrorInvalidAuth
}

func TestHandlePaymeWebhook_MethodNotFound(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	rpcReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "NonExistentMethod",
		"params":  json.RawMessage(`{}`),
	}
	payload, _ := json.Marshal(rpcReq)

	credentials := base64.StdEncoding.EncodeToString([]byte(testPaymeMerchantID + ":" + testPaymeSecretKey))
	authHeader := "Basic " + credentials

	resp := svc.HandlePaymeWebhook(context.Background(), authHeader, payload)
	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	testutil.AssertEqual(t, resp.Error.Code, -32601) // PaymeErrorMethodNotFound
}

func TestHandlePaymeWebhook_InvalidJSON(t *testing.T) {
	txRepo := &mocks.MockTransactionRepository{}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(testPaymeMerchantID+":"+testPaymeSecretKey))

	resp := svc.HandlePaymeWebhook(context.Background(), authHeader, []byte("not-json"))
	if resp.Error == nil {
		t.Fatal("expected error for invalid JSON")
	}
	testutil.AssertEqual(t, resp.Error.Code, -32700) // Parse error
}

// ---------------------------------------------------------------------------
// GetUserTransactions tests
// ---------------------------------------------------------------------------

func TestGetUserTransactions(t *testing.T) {
	userID := uuid.New()
	tx1 := testutil.NewTestTransaction(userID)
	tx2 := testutil.NewTestTransaction(userID)

	txRepo := &mocks.MockTransactionRepository{
		FindByUserFn: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.Transaction, int64, error) {
			return []*domain.Transaction{tx1, tx2}, 2, nil
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	txs, total, err := svc.GetUserTransactions(context.Background(), userID, 10, 0)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, len(txs), 2)
	testutil.AssertEqual(t, total, int64(2))
}

// ---------------------------------------------------------------------------
// GetTransactionByID tests
// ---------------------------------------------------------------------------

func TestGetTransactionByID_Success(t *testing.T) {
	userID := uuid.New()
	tx := testutil.NewTestTransaction(userID)

	txRepo := &mocks.MockTransactionRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
			if id == tx.ID {
				return tx, nil
			}
			return nil, fmt.Errorf("not found")
		},
	}
	subRepo := &mocks.MockSubscriptionRepository{}
	userRepo := &mocks.MockUserRepository{}
	svc := newTestPaymentService(txRepo, subRepo, userRepo)

	result, err := svc.GetTransactionByID(context.Background(), tx.ID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, result)
	testutil.AssertEqual(t, result.ID, tx.ID)
}

// Suppress unused import warning for time.
var _ = time.Now
