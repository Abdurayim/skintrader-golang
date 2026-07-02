package pkg_test

import (
	"testing"

	"skintrader-go/internal/pkg/hash"
	"skintrader-go/tests/testutil"
)

// ---------------------------------------------------------------------------
// HashPassword tests
// ---------------------------------------------------------------------------

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123"

	hashed, err := hash.HashPassword(password)
	testutil.AssertNoError(t, err)

	if hashed == "" {
		t.Fatal("expected non-empty hash")
	}

	// Hash should not equal the original password.
	if hashed == password {
		t.Fatal("hash should not equal the original password")
	}

	// Two hashes of the same password should be different (bcrypt uses random salts).
	hashed2, err := hash.HashPassword(password)
	testutil.AssertNoError(t, err)
	if hashed == hashed2 {
		t.Fatal("expected different hashes for the same password (bcrypt uses random salts)")
	}
}

func TestHashPassword_EmptyString(t *testing.T) {
	hashed, err := hash.HashPassword("")
	testutil.AssertNoError(t, err)
	if hashed == "" {
		t.Fatal("expected non-empty hash even for empty string")
	}
}

// ---------------------------------------------------------------------------
// CheckPassword tests
// ---------------------------------------------------------------------------

func TestCheckPassword_Correct(t *testing.T) {
	password := "mySecurePassword123"

	hashed, err := hash.HashPassword(password)
	testutil.AssertNoError(t, err)

	result := hash.CheckPassword(password, hashed)
	testutil.AssertTrue(t, result)
}

func TestCheckPassword_Wrong(t *testing.T) {
	password := "mySecurePassword123"

	hashed, err := hash.HashPassword(password)
	testutil.AssertNoError(t, err)

	result := hash.CheckPassword("wrongPassword", hashed)
	testutil.AssertFalse(t, result)
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hashed, err := hash.HashPassword("realPassword")
	testutil.AssertNoError(t, err)

	result := hash.CheckPassword("", hashed)
	testutil.AssertFalse(t, result)
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	result := hash.CheckPassword("password", "not-a-valid-bcrypt-hash")
	testutil.AssertFalse(t, result)
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	result := hash.CheckPassword("password", "")
	testutil.AssertFalse(t, result)
}

func TestCheckPassword_VerifyEmptyPasswordHash(t *testing.T) {
	// Hash the empty string and verify it matches itself.
	hashed, err := hash.HashPassword("")
	testutil.AssertNoError(t, err)

	result := hash.CheckPassword("", hashed)
	testutil.AssertTrue(t, result)

	// But a non-empty password should not match.
	result2 := hash.CheckPassword("something", hashed)
	testutil.AssertFalse(t, result2)
}
