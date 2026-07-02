package pkg_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	jwtpkg "skintrader-go/internal/pkg/jwt"
	"skintrader-go/tests/testutil"
)

func newTestJWTManager() *jwtpkg.Manager {
	return jwtpkg.NewManager("test-jwt-secret-key-12345", 15*time.Minute, 7*24*time.Hour)
}

// ---------------------------------------------------------------------------
// GenerateUserTokens tests
// ---------------------------------------------------------------------------

func TestGenerateUserTokens(t *testing.T) {
	manager := newTestJWTManager()
	userID := uuid.New()

	tokens, err := manager.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, tokens)

	if tokens.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	// Validate the access token.
	claims, err := manager.ValidateToken(tokens.AccessToken)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, claims.UserID, userID)
	testutil.AssertEqual(t, claims.Role, jwtpkg.RoleUser)
	testutil.AssertEqual(t, claims.Type, jwtpkg.AccessToken)

	// Validate the refresh token.
	refreshClaims, err := manager.ValidateToken(tokens.RefreshToken)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, refreshClaims.UserID, userID)
	testutil.AssertEqual(t, refreshClaims.Role, jwtpkg.RoleUser)
	testutil.AssertEqual(t, refreshClaims.Type, jwtpkg.RefreshToken)
}

// ---------------------------------------------------------------------------
// GenerateAdminTokens tests
// ---------------------------------------------------------------------------

func TestGenerateAdminTokens(t *testing.T) {
	manager := newTestJWTManager()
	adminID := uuid.New()

	tokens, err := manager.GenerateAdminTokens(adminID)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, tokens)

	if tokens.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if tokens.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}

	// Validate the access token.
	claims, err := manager.ValidateToken(tokens.AccessToken)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, claims.AdminID, adminID)
	testutil.AssertEqual(t, claims.Role, jwtpkg.RoleAdmin)
	testutil.AssertEqual(t, claims.Type, jwtpkg.AccessToken)

	// Validate the refresh token.
	refreshClaims, err := manager.ValidateToken(tokens.RefreshToken)
	testutil.AssertNoError(t, err)
	testutil.AssertEqual(t, refreshClaims.AdminID, adminID)
	testutil.AssertEqual(t, refreshClaims.Role, jwtpkg.RoleAdmin)
	testutil.AssertEqual(t, refreshClaims.Type, jwtpkg.RefreshToken)
}

// ---------------------------------------------------------------------------
// ValidateToken tests
// ---------------------------------------------------------------------------

func TestValidateToken_Valid(t *testing.T) {
	manager := newTestJWTManager()
	userID := uuid.New()

	tokens, err := manager.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)

	claims, err := manager.ValidateToken(tokens.AccessToken)
	testutil.AssertNoError(t, err)
	testutil.AssertNotNil(t, claims)
	testutil.AssertEqual(t, claims.UserID, userID)
	testutil.AssertEqual(t, claims.Role, jwtpkg.RoleUser)
	testutil.AssertEqual(t, claims.Type, jwtpkg.AccessToken)
}

func TestValidateToken_Expired(t *testing.T) {
	// Create a manager with very short expiry.
	manager := jwtpkg.NewManager("test-jwt-secret-key-12345", 1*time.Millisecond, 1*time.Millisecond)
	userID := uuid.New()

	tokens, err := manager.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)

	// Wait for the token to expire.
	time.Sleep(10 * time.Millisecond)

	_, err = manager.ValidateToken(tokens.AccessToken)
	testutil.AssertError(t, err)
}

func TestValidateToken_InvalidString(t *testing.T) {
	manager := newTestJWTManager()

	_, err := manager.ValidateToken("not.a.valid.token")
	testutil.AssertError(t, err)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	manager1 := jwtpkg.NewManager("secret-one", 15*time.Minute, 7*24*time.Hour)
	manager2 := jwtpkg.NewManager("secret-two", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	tokens, err := manager1.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)

	// Validate with wrong secret should fail.
	_, err = manager2.ValidateToken(tokens.AccessToken)
	testutil.AssertError(t, err)
}

func TestValidateToken_EmptyString(t *testing.T) {
	manager := newTestJWTManager()

	_, err := manager.ValidateToken("")
	testutil.AssertError(t, err)
}

// ---------------------------------------------------------------------------
// HashToken tests
// ---------------------------------------------------------------------------

func TestHashToken(t *testing.T) {
	token := "some-token-string"

	hash1 := jwtpkg.HashToken(token)
	hash2 := jwtpkg.HashToken(token)

	// Same input should produce same hash.
	testutil.AssertEqual(t, hash1, hash2)

	// Hash should not be empty.
	if hash1 == "" {
		t.Fatal("expected non-empty hash")
	}

	// Different input should produce different hash.
	hash3 := jwtpkg.HashToken("different-token")
	if hash1 == hash3 {
		t.Fatal("expected different hash for different input")
	}
}

// ---------------------------------------------------------------------------
// Token uniqueness tests
// ---------------------------------------------------------------------------

func TestTokensAreUnique(t *testing.T) {
	manager := newTestJWTManager()
	userID := uuid.New()

	tokens1, err := manager.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)

	tokens2, err := manager.GenerateUserTokens(userID)
	testutil.AssertNoError(t, err)

	// Tokens should be different each time they are generated.
	if tokens1.AccessToken == tokens2.AccessToken {
		t.Fatal("expected different access tokens for sequential generation")
	}
	if tokens1.RefreshToken == tokens2.RefreshToken {
		t.Fatal("expected different refresh tokens for sequential generation")
	}
}
