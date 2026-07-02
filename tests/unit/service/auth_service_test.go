package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	"skintrader-go/internal/pkg/hash"
	jwtpkg "skintrader-go/internal/pkg/jwt"
	"skintrader-go/internal/service"
	"skintrader-go/tests/mocks"
	"skintrader-go/tests/testutil"
)

// newTestAuthService creates an AuthService with the provided mock repos.
func newTestAuthService(userRepo *mocks.MockUserRepository, adminRepo *mocks.MockAdminRepository) (*service.AuthService, *jwtpkg.Manager) {
	jwtManager := jwtpkg.NewManager("test-secret-key-for-jwt-testing", 15*time.Minute, 7*24*time.Hour)
	authConfig := config.AuthConfig{
		GoogleClientID: "test-google-client-id",
		AppleBundleID:  "com.test.skintrader",
	}
	logger := zerolog.Nop()

	svc := service.NewAuthService(userRepo, adminRepo, jwtManager, authConfig, logger)
	return svc, jwtManager
}

// ---------------------------------------------------------------------------
// Register tests
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, fmt.Errorf("not found")
		},
		CreateFn: func(ctx context.Context, user *domain.User) error {
			return nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Register(context.Background(), "test@example.com", "password123", "Test User")
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, user)
	testutil.AssertNotNil(t, tokens)
	testutil.AssertEqual(t, user.Email, "test@example.com")
	testutil.AssertEqual(t, user.DisplayName, "Test User")
	testutil.AssertEqual(t, user.AuthProvider, domain.AuthProviderEmail)
	testutil.AssertEqual(t, user.Status, domain.UserStatusActive)
	testutil.AssertEqual(t, user.KYCStatus, domain.KYCStatusNotSubmitted)

	if tokens.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	existingUser := testutil.NewTestUser()
	existingUser.Email = "existing@example.com"

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return existingUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Register(context.Background(), "existing@example.com", "password123", "Test User")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on duplicate email")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on duplicate email")
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Register(context.Background(), "test@example.com", "short", "Test User")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on short password")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on short password")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Register(context.Background(), "not-an-email", "password123", "Test User")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on invalid email")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on invalid email")
	}
}

func TestRegister_EmptyDisplayName(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Register(context.Background(), "test@example.com", "password123", "")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on empty display name")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on empty display name")
	}
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	passwordHash, _ := hash.HashPassword("password123")
	existingUser := testutil.NewTestUser()
	existingUser.Email = "user@example.com"
	existingUser.PasswordHash = &passwordHash

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return existingUser, nil
		},
		UpdateFn: func(ctx context.Context, user *domain.User) error {
			return nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "user@example.com", "password123")
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, user)
	testutil.AssertNotNil(t, tokens)
	testutil.AssertEqual(t, user.Email, "user@example.com")
}

func TestLogin_WrongPassword(t *testing.T) {
	passwordHash, _ := hash.HashPassword("password123")
	existingUser := testutil.NewTestUser()
	existingUser.Email = "user@example.com"
	existingUser.PasswordHash = &passwordHash

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return existingUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "user@example.com", "wrongpassword")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on wrong password")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on wrong password")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "nonexistent@example.com", "password123")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user when not found")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens when not found")
	}
}

func TestLogin_EmptyCredentials(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "", "")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user on empty credentials")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens on empty credentials")
	}
}

func TestLogin_BannedUser(t *testing.T) {
	passwordHash, _ := hash.HashPassword("password123")
	bannedUser := testutil.NewTestUser()
	bannedUser.Email = "banned@example.com"
	bannedUser.PasswordHash = &passwordHash
	bannedUser.Status = domain.UserStatusBanned

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return bannedUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "banned@example.com", "password123")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user for banned account")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens for banned account")
	}
}

func TestLogin_SuspendedUser(t *testing.T) {
	passwordHash, _ := hash.HashPassword("password123")
	suspendedUser := testutil.NewTestUser()
	suspendedUser.Email = "suspended@example.com"
	suspendedUser.PasswordHash = &passwordHash
	suspendedUser.Status = domain.UserStatusSuspended

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return suspendedUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "suspended@example.com", "password123")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user for suspended account")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens for suspended account")
	}
}

func TestLogin_OAuthOnlyAccount(t *testing.T) {
	oauthUser := testutil.NewTestGoogleUser()
	oauthUser.Email = "oauth@example.com"
	// PasswordHash is nil for OAuth users

	userRepo := &mocks.MockUserRepository{
		FindByEmailFn: func(ctx context.Context, email string) (*domain.User, error) {
			return oauthUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, tokens, err := svc.Login(context.Background(), "oauth@example.com", "password123")
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user for OAuth-only account")
	}
	if tokens != nil {
		t.Fatal("expected nil tokens for OAuth-only account")
	}
}

// ---------------------------------------------------------------------------
// RefreshToken tests
// ---------------------------------------------------------------------------

func TestRefreshToken_Success(t *testing.T) {
	existingUser := testutil.NewTestUser()

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, jwtManager := newTestAuthService(userRepo, adminRepo)

	// Generate initial tokens for the user.
	originalTokens, err := jwtManager.GenerateUserTokens(existingUser.ID)
	testutil.AssertNoError(t, err)

	// Use refresh token to get new tokens.
	newTokens, err := svc.RefreshToken(context.Background(), originalTokens.RefreshToken)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, newTokens)
	if newTokens.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if newTokens.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
}

func TestRefreshToken_Expired(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}

	// Create a JWT manager with very short expiry to test expiration.
	expiredJwtManager := jwtpkg.NewManager("test-secret-key-for-jwt-testing", 1*time.Millisecond, 1*time.Millisecond)

	authConfig := config.AuthConfig{
		GoogleClientID: "test-google-client-id",
	}
	logger := zerolog.Nop()
	svc := service.NewAuthService(userRepo, adminRepo, expiredJwtManager, authConfig, logger)

	// Generate tokens that will expire almost immediately.
	user := testutil.NewTestUser()
	tokens, err := expiredJwtManager.GenerateUserTokens(user.ID)
	testutil.AssertNoError(t, err)

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	// Now try to refresh with the expired token.
	newTokens, err := svc.RefreshToken(context.Background(), tokens.RefreshToken)
	testutil.AssertError(t, err)
	if newTokens != nil {
		t.Fatal("expected nil tokens for expired refresh token")
	}
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	newTokens, err := svc.RefreshToken(context.Background(), "")
	testutil.AssertError(t, err)
	if newTokens != nil {
		t.Fatal("expected nil tokens for empty refresh token")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	newTokens, err := svc.RefreshToken(context.Background(), "invalid.token.string")
	testutil.AssertError(t, err)
	if newTokens != nil {
		t.Fatal("expected nil tokens for invalid refresh token")
	}
}

func TestRefreshToken_AccessTokenInsteadOfRefresh(t *testing.T) {
	existingUser := testutil.NewTestUser()

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, jwtManager := newTestAuthService(userRepo, adminRepo)

	// Generate tokens.
	tokens, err := jwtManager.GenerateUserTokens(existingUser.ID)
	testutil.AssertNoError(t, err)

	// Try using the access token as a refresh token.
	newTokens, err := svc.RefreshToken(context.Background(), tokens.AccessToken)
	testutil.AssertError(t, err)
	if newTokens != nil {
		t.Fatal("expected nil tokens when using access token as refresh token")
	}
}

func TestRefreshToken_BannedUser(t *testing.T) {
	bannedUser := testutil.NewTestUser()
	bannedUser.Status = domain.UserStatusBanned

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return bannedUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, jwtManager := newTestAuthService(userRepo, adminRepo)

	tokens, err := jwtManager.GenerateUserTokens(bannedUser.ID)
	testutil.AssertNoError(t, err)

	newTokens, err := svc.RefreshToken(context.Background(), tokens.RefreshToken)
	testutil.AssertError(t, err)
	if newTokens != nil {
		t.Fatal("expected nil tokens for banned user")
	}
}

func TestRefreshToken_AdminSuccess(t *testing.T) {
	existingAdmin := testutil.NewTestAdmin()

	userRepo := &mocks.MockUserRepository{}
	adminRepo := &mocks.MockAdminRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.Admin, error) {
			return existingAdmin, nil
		},
	}
	svc, jwtManager := newTestAuthService(userRepo, adminRepo)

	// Generate admin tokens.
	tokens, err := jwtManager.GenerateAdminTokens(existingAdmin.ID)
	testutil.AssertNoError(t, err)

	// Use admin refresh token.
	newTokens, err := svc.RefreshToken(context.Background(), tokens.RefreshToken)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, newTokens)
}

// ---------------------------------------------------------------------------
// GetMe tests
// ---------------------------------------------------------------------------

func TestGetMe_Success(t *testing.T) {
	existingUser := testutil.NewTestUser()

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, err := svc.GetMe(context.Background(), existingUser.ID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, user)
	testutil.AssertEqual(t, user.ID, existingUser.ID)
}

func TestGetMe_NotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	adminRepo := &mocks.MockAdminRepository{}
	svc, _ := newTestAuthService(userRepo, adminRepo)

	user, err := svc.GetMe(context.Background(), uuid.New())
	testutil.AssertError(t, err)
	if user != nil {
		t.Fatal("expected nil user when not found")
	}
}
