package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// RandomUUID generates a new random UUID.
func RandomUUID() uuid.UUID {
	return uuid.New()
}

// RandomString generates a random hex string of the given length.
func RandomString(length int) string {
	bytes := make([]byte, (length+1)/2)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// RandomEmail generates a random email address.
func RandomEmail() string {
	return fmt.Sprintf("%s@test.com", RandomString(10))
}

// RandomInt generates a random integer between min and max (inclusive).
func RandomInt(min, max int64) int64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(max-min+1))
	return n.Int64() + min
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

// TimePtr returns a pointer to the given time.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// UUIDPtr returns a pointer to the given UUID.
func UUIDPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

// Float32Ptr returns a pointer to the given float32.
func Float32Ptr(f float32) *float32 {
	return &f
}

// Float64Ptr returns a pointer to the given float64.
func Float64Ptr(f float64) *float64 {
	return &f
}

// ---------------------------------------------------------------------------
// Test factories
// ---------------------------------------------------------------------------

// NewTestUser creates a domain.User with sensible defaults.
// Override any field after creation as needed.
func NewTestUser() *domain.User {
	now := time.Now()
	return &domain.User{
		ID:                 uuid.New(),
		AuthProvider:       domain.AuthProviderEmail,
		Email:              RandomEmail(),
		PasswordHash:       StringPtr("$2a$12$hashedpassword"),
		EmailVerified:      true,
		DisplayName:        "Test User " + RandomString(4),
		PhoneNumber:        "",
		Bio:                "",
		AvatarURL:          "",
		SocialMedia:        json.RawMessage(`{}`),
		Language:           domain.LanguageEN,
		Status:             domain.UserStatusActive,
		KYCStatus:          domain.KYCStatusNotSubmitted,
		SubscriptionStatus: domain.SubscriptionStatusNone,
		PostsCount:         0,
		ReportsReceived:    0,
		ReportsMade:        0,
		LastLoginAt:        &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewTestGoogleUser creates a domain.User that was authenticated via Google.
func NewTestGoogleUser() *domain.User {
	u := NewTestUser()
	u.AuthProvider = domain.AuthProviderGoogle
	googleID := "google_" + RandomString(16)
	u.GoogleID = &googleID
	u.PasswordHash = nil
	u.EmailVerified = true
	return u
}

// NewTestAdmin creates a domain.Admin with sensible defaults.
func NewTestAdmin() *domain.Admin {
	now := time.Now()
	return &domain.Admin{
		ID:           uuid.New(),
		Email:        RandomEmail(),
		PasswordHash: "$2a$12$hashedpassword",
		Name:         "Test Admin " + RandomString(4),
		Role:         domain.AdminRoleSuperAdmin,
		Permissions:  domain.RolePermissions[domain.AdminRoleSuperAdmin],
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewTestModeratorAdmin creates a domain.Admin with moderator role.
func NewTestModeratorAdmin() *domain.Admin {
	admin := NewTestAdmin()
	admin.Role = domain.AdminRoleModerator
	admin.Permissions = domain.RolePermissions[domain.AdminRoleModerator]
	return admin
}

// NewTestPost creates a domain.Post with sensible defaults.
func NewTestPost(userID, gameID uuid.UUID) *domain.Post {
	now := time.Now()
	return &domain.Post{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Test Post " + RandomString(6),
		Description: "Test description for post",
		Price:       float64(RandomInt(1000, 100000)),
		Currency:    domain.CurrencyUZS,
		GameID:      gameID,
		Genre:       "FPS",
		Type:        domain.PostTypeSkin,
		Status:      domain.PostStatusActive,
		ViewsCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewTestGame creates a domain.Game with sensible defaults.
func NewTestGame() *domain.Game {
	now := time.Now()
	name := "Test Game " + RandomString(4)
	return &domain.Game{
		ID:         uuid.New(),
		Name:       name,
		Slug:       "test-game-" + RandomString(4),
		Icon:       "",
		Genres:     []string{"FPS", "Action"},
		PostsCount: 0,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// NewTestTransaction creates a domain.Transaction with sensible defaults.
func NewTestTransaction(userID uuid.UUID) *domain.Transaction {
	now := time.Now()
	return &domain.Transaction{
		ID:            uuid.New(),
		UserID:        userID,
		Amount:        10000,
		Currency:      domain.CurrencyUZS,
		Status:        domain.TransactionStatusPending,
		PaymentMethod: domain.PaymentMethodPayme,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// NewTestSubscription creates a domain.Subscription with sensible defaults.
func NewTestSubscription(userID uuid.UUID) *domain.Subscription {
	now := time.Now()
	endDate := now.AddDate(0, 0, 30)
	return &domain.Subscription{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    domain.SubscriptionStatusActive,
		Plan:      domain.SubscriptionPlanMonthly,
		StartDate: &now,
		EndDate:   &endDate,
		AutoRenew: false,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewTestReport creates a domain.Report with sensible defaults.
func NewTestReport(reporterID, targetID uuid.UUID) *domain.Report {
	now := time.Now()
	return &domain.Report{
		ID:          uuid.New(),
		ReporterID:  reporterID,
		ReportType:  domain.ReportTypePost,
		TargetID:    targetID,
		TargetModel: "posts",
		Category:    domain.ReportCategoryScam,
		Description: "Test report description",
		Status:      domain.ReportStatusPending,
		Priority:    domain.ReportPriorityMedium,
		ReportHash:  RandomString(32),
		ReportCount: 1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewTestMessage creates a domain.Message with sensible defaults.
func NewTestMessage(conversationID, senderID uuid.UUID) *domain.Message {
	now := time.Now()
	return &domain.Message{
		ID:             uuid.New(),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        "Test message " + RandomString(8),
		Status:         domain.MessageStatusSent,
		CreatedAt:      now,
	}
}

// NewTestConversation creates a domain.Conversation with sensible defaults.
func NewTestConversation(user1, user2 uuid.UUID) *domain.Conversation {
	now := time.Now()
	return &domain.Conversation{
		ID:           uuid.New(),
		Participant1: user1,
		Participant2: user2,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error but got nil")
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected no error but got: %v", err)
	}
}

// AssertEqual fails the test if got != want.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// AssertTrue fails the test if value is false.
func AssertTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Fatal("expected true, got false")
	}
}

// AssertFalse fails the test if value is true.
func AssertFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Fatal("expected false, got true")
	}
}

// AssertNotNil fails the test if value is nil.
func AssertNotNil(t *testing.T, value interface{}) {
	t.Helper()
	if value == nil {
		t.Fatal("expected non-nil value, got nil")
	}
}

// AssertNil fails the test if value is not nil.
func AssertNil(t *testing.T, value interface{}) {
	t.Helper()
	if value != nil {
		t.Fatalf("expected nil, got %v", value)
	}
}
