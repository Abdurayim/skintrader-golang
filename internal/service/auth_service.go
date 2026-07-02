package service

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/api/idtoken"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	apperr "skintrader-go/internal/pkg/errors"
	"skintrader-go/internal/pkg/hash"
	jwtpkg "skintrader-go/internal/pkg/jwt"
)

const (
	appleJWKSURL = "https://appleid.apple.com/auth/keys"
	appleIssuer  = "https://appleid.apple.com"
)

// AuthService handles all authentication operations including
// Google OAuth, Apple Sign-In, and email+password authentication.
type AuthService struct {
	userRepo   domain.UserRepository
	adminRepo  domain.AdminRepository
	jwtManager *jwtpkg.Manager
	authConfig config.AuthConfig
	logger     zerolog.Logger
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(
	userRepo domain.UserRepository,
	adminRepo domain.AdminRepository,
	jwtManager *jwtpkg.Manager,
	authConfig config.AuthConfig,
	logger zerolog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		adminRepo:  adminRepo,
		jwtManager: jwtManager,
		authConfig: authConfig,
		logger:     logger.With().Str("service", "auth").Logger(),
	}
}

// ---------------------------------------------------------------------------
// Google OAuth
// ---------------------------------------------------------------------------

// GoogleAuth authenticates or registers a user via Google OAuth.
// It verifies the Google ID token using the google.golang.org/api/idtoken package,
// extracts user info, and performs find-or-create logic.
func (s *AuthService) GoogleAuth(ctx context.Context, idToken string) (*domain.User, *jwtpkg.TokenPair, error) {
	if idToken == "" {
		return nil, nil, apperr.BadRequest("id token is required")
	}

	// Verify Google ID token against the configured client ID.
	payload, err := idtoken.Validate(ctx, idToken, s.authConfig.GoogleClientID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("google id token validation failed")
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Google ID token", http.StatusUnauthorized)
	}

	// Extract claims from the verified payload.
	googleID, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)

	if googleID == "" {
		return nil, nil, apperr.BadRequest("Google ID token missing sub claim")
	}
	if email == "" {
		return nil, nil, apperr.BadRequest("Google ID token missing email claim")
	}

	user, err := s.findOrCreateUser(ctx, domain.AuthProviderGoogle, googleID, email, name, picture)
	if err != nil {
		return nil, nil, err
	}

	// Update last login timestamp.
	now := time.Now()
	user.LastLoginAt = &now
	if updateErr := s.userRepo.Update(ctx, user); updateErr != nil {
		s.logger.Error().Err(updateErr).Str("user_id", user.ID.String()).Msg("failed to update last login")
	}

	tokens, err := s.jwtManager.GenerateUserTokens(user.ID)
	if err != nil {
		return nil, nil, apperr.Internal("failed to generate tokens")
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("provider", "google").Msg("user authenticated via Google")
	return user, tokens, nil
}

// ---------------------------------------------------------------------------
// Apple Sign-In
// ---------------------------------------------------------------------------

// appleJWKS represents the structure of Apple's JWKS response.
type appleJWKS struct {
	Keys []appleJWK `json:"keys"`
}

// appleJWK represents a single JWK from Apple's key set.
type appleJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// appleClaims represents the claims in an Apple identity token.
type appleClaims struct {
	gojwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"` // Apple may send string "true" or bool
}

// AppleAuth authenticates or registers a user via Apple Sign-In.
// It verifies the Apple identity token by fetching Apple's JWKS, finding the matching
// key, and validating issuer, audience, and expiry.
func (s *AuthService) AppleAuth(ctx context.Context, identityToken string, fullName *string) (*domain.User, *jwtpkg.TokenPair, error) {
	if identityToken == "" {
		return nil, nil, apperr.BadRequest("identity token is required")
	}

	// Parse the JWT header to extract the kid (key ID).
	parts := strings.Split(identityToken, ".")
	if len(parts) != 3 {
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Apple identity token format", http.StatusUnauthorized)
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Apple identity token header", http.StatusUnauthorized)
	}

	var header struct {
		Kid string `json:"kid"`
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Apple identity token header", http.StatusUnauthorized)
	}
	if header.Kid == "" {
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "Apple identity token missing kid", http.StatusUnauthorized)
	}

	// Fetch Apple's public keys (JWKS) and find the matching key.
	rsaKey, err := s.fetchApplePublicKey(ctx, header.Kid)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to fetch Apple public key")
		return nil, nil, apperr.Internal("failed to verify Apple identity token")
	}

	// Parse and validate the token with the RSA public key.
	claims := &appleClaims{}
	token, err := gojwt.ParseWithClaims(identityToken, claims, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return rsaKey, nil
	},
		gojwt.WithIssuer(appleIssuer),
		gojwt.WithAudience(s.authConfig.AppleBundleID),
		gojwt.WithExpirationRequired(),
	)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Apple identity token validation failed")
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Apple identity token", http.StatusUnauthorized)
	}
	if !token.Valid {
		return nil, nil, apperr.New(apperr.CodeInvalidToken, "invalid Apple identity token", http.StatusUnauthorized)
	}

	// Extract Apple user ID (sub) and email.
	appleID := claims.Subject
	if appleID == "" {
		return nil, nil, apperr.BadRequest("Apple identity token missing sub claim")
	}

	email := claims.Email

	// Determine display name: Apple only sends fullName on the first sign-in.
	displayName := ""
	if fullName != nil && *fullName != "" {
		displayName = *fullName
	}
	if displayName == "" && email != "" {
		displayName = strings.Split(email, "@")[0]
	}

	user, err := s.findOrCreateUser(ctx, domain.AuthProviderApple, appleID, email, displayName, "")
	if err != nil {
		return nil, nil, err
	}

	// Update last login timestamp.
	now := time.Now()
	user.LastLoginAt = &now
	if updateErr := s.userRepo.Update(ctx, user); updateErr != nil {
		s.logger.Error().Err(updateErr).Str("user_id", user.ID.String()).Msg("failed to update last login")
	}

	tokens, err := s.jwtManager.GenerateUserTokens(user.ID)
	if err != nil {
		return nil, nil, apperr.Internal("failed to generate tokens")
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("provider", "apple").Msg("user authenticated via Apple")
	return user, tokens, nil
}

// fetchApplePublicKey fetches Apple's JWKS and returns the RSA public key matching the given kid.
func (s *AuthService) fetchApplePublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleJWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch Apple JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Apple JWKS returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Apple JWKS body: %w", err)
	}

	var jwks appleJWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, fmt.Errorf("parse Apple JWKS: %w", err)
	}

	for _, key := range jwks.Keys {
		if key.Kid == kid {
			return jwkToRSAPublicKey(key)
		}
	}

	return nil, fmt.Errorf("Apple public key with kid %q not found", kid)
}

// jwkToRSAPublicKey converts an Apple JWK to an *rsa.PublicKey.
func jwkToRSAPublicKey(key appleJWK) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// ---------------------------------------------------------------------------
// Email + Password
// ---------------------------------------------------------------------------

// Register creates a new user with email and password.
func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*domain.User, *jwtpkg.TokenPair, error) {
	// Validate email format.
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, nil, apperr.BadRequest("invalid email format")
	}

	// Validate password length.
	if len(password) < 8 {
		return nil, nil, apperr.BadRequest("password must be at least 8 characters")
	}

	// Validate display name.
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return nil, nil, apperr.BadRequest("display name is required")
	}

	// Normalize email to lowercase.
	email = strings.ToLower(strings.TrimSpace(email))

	// Check if email already exists.
	existing, _ := s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, nil, apperr.ErrEmailExists
	}

	// Hash password.
	passwordHash, err := hash.HashPassword(password)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to hash password")
		return nil, nil, apperr.Internal("failed to create account")
	}

	now := time.Now()
	user := &domain.User{
		ID:                 uuid.New(),
		AuthProvider:       domain.AuthProviderEmail,
		Email:              email,
		PasswordHash:       &passwordHash,
		EmailVerified:      false,
		DisplayName:        displayName,
		Language:           domain.LanguageEN,
		Status:             domain.UserStatusActive,
		KYCStatus:          domain.KYCStatusNotSubmitted,
		SubscriptionStatus: domain.SubscriptionStatusNone,
		LastLoginAt:        &now,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error().Err(err).Str("email", email).Msg("failed to create user")
		return nil, nil, apperr.Internal("failed to create account")
	}

	tokens, err := s.jwtManager.GenerateUserTokens(user.ID)
	if err != nil {
		return nil, nil, apperr.Internal("failed to generate tokens")
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("email", email).Msg("user registered via email")
	return user, tokens, nil
}

// Login authenticates a user with email and password.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *jwtpkg.TokenPair, error) {
	if email == "" || password == "" {
		return nil, nil, apperr.BadRequest("email and password are required")
	}

	// Normalize email.
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return nil, nil, apperr.ErrInvalidCredentials
	}

	// Check if this is an OAuth-only account (no password set).
	if user.PasswordHash == nil {
		return nil, nil, apperr.New(apperr.CodeBadRequest, "this account uses OAuth login (Google or Apple)", http.StatusBadRequest)
	}

	// Verify password.
	if !hash.CheckPassword(password, *user.PasswordHash) {
		return nil, nil, apperr.ErrInvalidCredentials
	}

	// Check account status.
	switch user.Status {
	case domain.UserStatusBanned:
		return nil, nil, apperr.ErrAccountBanned
	case domain.UserStatusSuspended:
		return nil, nil, apperr.ErrAccountSuspended
	}

	// Update last login timestamp.
	now := time.Now()
	user.LastLoginAt = &now
	if updateErr := s.userRepo.Update(ctx, user); updateErr != nil {
		s.logger.Error().Err(updateErr).Str("user_id", user.ID.String()).Msg("failed to update last login")
	}

	tokens, err := s.jwtManager.GenerateUserTokens(user.ID)
	if err != nil {
		return nil, nil, apperr.Internal("failed to generate tokens")
	}

	s.logger.Info().Str("user_id", user.ID.String()).Str("email", email).Msg("user logged in via email")
	return user, tokens, nil
}

// ---------------------------------------------------------------------------
// Token management
// ---------------------------------------------------------------------------

// RefreshToken generates new tokens from a valid refresh token.
// Supports both user and admin token refresh.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwtpkg.TokenPair, error) {
	if refreshToken == "" {
		return nil, apperr.BadRequest("refresh token is required")
	}

	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, apperr.ErrInvalidToken
	}

	if claims.Type != jwtpkg.RefreshToken {
		return nil, apperr.New(apperr.CodeInvalidToken, "invalid token type, expected refresh token", http.StatusUnauthorized)
	}

	switch claims.Role {
	case jwtpkg.RoleUser:
		// Verify the user still exists and is in good standing.
		user, findErr := s.userRepo.FindByID(ctx, claims.UserID)
		if findErr != nil || user == nil {
			return nil, apperr.NotFound("user")
		}
		if user.Status == domain.UserStatusBanned {
			return nil, apperr.ErrAccountBanned
		}
		if user.Status == domain.UserStatusSuspended {
			return nil, apperr.ErrAccountSuspended
		}

		tokens, genErr := s.jwtManager.GenerateUserTokens(user.ID)
		if genErr != nil {
			return nil, apperr.Internal("failed to generate tokens")
		}
		return tokens, nil

	case jwtpkg.RoleAdmin:
		// Verify the admin still exists and is active.
		admin, findErr := s.adminRepo.FindByID(ctx, claims.AdminID)
		if findErr != nil || admin == nil {
			return nil, apperr.NotFound("admin")
		}
		if !admin.IsActive {
			return nil, apperr.Forbidden("admin account is deactivated")
		}

		tokens, genErr := s.jwtManager.GenerateAdminTokens(admin.ID)
		if genErr != nil {
			return nil, apperr.Internal("failed to generate tokens")
		}
		return tokens, nil

	default:
		return nil, apperr.New(apperr.CodeInvalidToken, "invalid token role", http.StatusUnauthorized)
	}
}

// Logout invalidates the user's current session by validating that the
// provided refresh token belongs to the requesting user.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, refreshToken string) error {
	if refreshToken == "" {
		return apperr.BadRequest("refresh token is required")
	}

	// Validate the refresh token and confirm ownership.
	claims, err := s.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return apperr.ErrInvalidToken
	}

	if claims.Type != jwtpkg.RefreshToken {
		return apperr.New(apperr.CodeInvalidToken, "invalid token type", http.StatusUnauthorized)
	}

	if claims.UserID != userID {
		return apperr.Forbidden("token does not belong to this user")
	}

	// The token is validated and belongs to the user.
	// In a system with a refresh token store (e.g., Redis or DB table),
	// the specific token hash would be removed here using:
	//   jwtpkg.HashToken(refreshToken) to get the hash, then delete from store.
	s.logger.Info().Str("user_id", userID.String()).Msg("user logged out")
	return nil
}

// LogoutAll invalidates all sessions for the given user.
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	// Verify the user exists.
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil || user == nil {
		return apperr.NotFound("user")
	}

	// In a system with a refresh token store, all tokens for this user
	// would be removed here. The client should discard all stored tokens.
	s.logger.Info().Str("user_id", userID.String()).Msg("user logged out from all sessions")
	return nil
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

// GetMe retrieves the current authenticated user's profile.
func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to find user")
		return nil, apperr.NotFound("user")
	}
	if user == nil {
		return nil, apperr.NotFound("user")
	}

	return user, nil
}

// ---------------------------------------------------------------------------
// Shared OAuth helpers
// ---------------------------------------------------------------------------

// findOrCreateUser implements the shared find-or-create logic for OAuth providers.
// 1. Try to find the user by provider-specific ID (GoogleID or AppleID).
// 2. If not found, try to find by email and link the provider.
// 3. If not found at all, create a new user.
func (s *AuthService) findOrCreateUser(
	ctx context.Context,
	provider domain.AuthProvider,
	providerID, email, name, avatarURL string,
) (*domain.User, error) {
	// Step 1: Look up by provider ID.
	var user *domain.User
	var err error

	switch provider {
	case domain.AuthProviderGoogle:
		user, err = s.userRepo.FindByGoogleID(ctx, providerID)
	case domain.AuthProviderApple:
		user, err = s.userRepo.FindByAppleID(ctx, providerID)
	default:
		return nil, apperr.BadRequest("unsupported auth provider")
	}

	if err == nil && user != nil {
		if user.Status == domain.UserStatusBanned {
			return nil, apperr.ErrAccountBanned
		}
		if user.Status == domain.UserStatusSuspended {
			return nil, apperr.ErrAccountSuspended
		}
		return user, nil
	}

	// Step 2: Look up by email and link the provider.
	if email != "" {
		user, err = s.userRepo.FindByEmail(ctx, email)
		if err == nil && user != nil {
			if user.Status == domain.UserStatusBanned {
				return nil, apperr.ErrAccountBanned
			}
			if user.Status == domain.UserStatusSuspended {
				return nil, apperr.ErrAccountSuspended
			}

			if linkErr := s.linkProvider(ctx, user, provider, providerID); linkErr != nil {
				return nil, linkErr
			}
			return user, nil
		}
	}

	// Step 3: Create a new user.
	displayName := name
	if displayName == "" && email != "" {
		displayName = strings.Split(email, "@")[0]
	}
	if displayName == "" {
		displayName = "User"
	}

	now := time.Now()
	user = &domain.User{
		ID:                 uuid.New(),
		AuthProvider:       provider,
		Email:              email,
		EmailVerified:      true, // OAuth-verified emails are trusted.
		DisplayName:        displayName,
		AvatarURL:          avatarURL,
		Language:           domain.LanguageEN,
		Status:             domain.UserStatusActive,
		KYCStatus:          domain.KYCStatusNotSubmitted,
		SubscriptionStatus: domain.SubscriptionStatusNone,
		LastLoginAt:        &now,
		CreatedAt:          now,
		UpdatedAt:          now,
		IsNewUser:          true,
	}

	switch provider {
	case domain.AuthProviderGoogle:
		user.GoogleID = &providerID
	case domain.AuthProviderApple:
		user.AppleID = &providerID
	}

	if createErr := s.userRepo.Create(ctx, user); createErr != nil {
		s.logger.Error().Err(createErr).
			Str("email", email).
			Str("provider", string(provider)).
			Msg("failed to create OAuth user")
		return nil, apperr.Internal("failed to create account")
	}

	s.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", email).
		Str("provider", string(provider)).
		Msg("new user created via OAuth")

	return user, nil
}

// linkProvider links a new OAuth provider to an existing user account.
func (s *AuthService) linkProvider(ctx context.Context, user *domain.User, provider domain.AuthProvider, providerID string) error {
	switch provider {
	case domain.AuthProviderGoogle:
		user.GoogleID = &providerID
	case domain.AuthProviderApple:
		user.AppleID = &providerID
	default:
		return apperr.BadRequest("unsupported auth provider")
	}

	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error().Err(err).
			Str("user_id", user.ID.String()).
			Str("provider", string(provider)).
			Msg("failed to link provider to user")
		return apperr.Internal("failed to link account")
	}

	s.logger.Info().
		Str("user_id", user.ID.String()).
		Str("provider", string(provider)).
		Msg("provider linked to existing user")

	return nil
}
