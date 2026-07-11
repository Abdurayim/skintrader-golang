package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	"skintrader-go/internal/service"
)

type AuthHandler struct {
	authService    *service.AuthService
	kycService     *service.KYCService
	authMiddleware *middleware.AuthMiddleware
	logger         zerolog.Logger
}

func NewAuthHandler(authService *service.AuthService, kycService *service.KYCService, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		kycService:     kycService,
		authMiddleware: authMiddleware,
		logger:         logger.With().Str("handler", "auth").Logger(),
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// GoogleAuth handles Google OAuth authentication.
func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req struct {
		IDToken string `json:"idToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "idToken is required")
		return
	}

	user, tokens, err := h.authService.GoogleAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		h.logger.Error().Err(err).Msg("google auth failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"user":   user,
		"tokens": tokens,
	}, "Google authentication successful")
}

// AppleAuth handles Apple OAuth authentication.
func (h *AuthHandler) AppleAuth(c *gin.Context) {
	var req struct {
		IdentityToken string  `json:"identityToken" binding:"required"`
		FullName      *string `json:"fullName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "identityToken is required")
		return
	}

	user, tokens, err := h.authService.AppleAuth(c.Request.Context(), req.IdentityToken, req.FullName)
	if err != nil {
		h.logger.Error().Err(err).Msg("apple auth failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"user":   user,
		"tokens": tokens,
	}, "Apple authentication successful")
}

// Register handles user registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	// Validate email format
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || !emailRegex.MatchString(req.Email) {
		BadRequest(c, "Invalid email format")
		return
	}

	// Validate password length
	if len(req.Password) < 8 {
		BadRequest(c, "Password must be at least 8 characters")
		return
	}

	// Validate display name
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		BadRequest(c, "Display name is required")
		return
	}

	user, tokens, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("registration failed")
		Error(c, err)
		return
	}

	Created(c, gin.H{
		"user":   user,
		"tokens": tokens,
	}, "Registration successful")
}

// Login handles user login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Email and password are required")
		return
	}

	user, tokens, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.logger.Warn().Err(err).Str("email", req.Email).Msg("login failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"user":   user,
		"tokens": tokens,
	}, "Login successful")
}

// RefreshToken handles token refresh.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "refreshToken is required")
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Warn().Err(err).Msg("token refresh failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"tokens": tokens,
	}, "Token refreshed successfully")
}

// Logout handles user logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "refreshToken is required")
		return
	}

	if err := h.authService.Logout(c.Request.Context(), userID, req.RefreshToken); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("logout failed")
		Error(c, err)
		return
	}

	Success(c, nil, "Logged out successfully")
}

// LogoutAll handles logging out from all sessions.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	if err := h.authService.LogoutAll(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("logout all failed")
		Error(c, err)
		return
	}

	Success(c, nil, "Logged out from all sessions successfully")
}

// GetMe returns the current authenticated user's information.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.authService.GetMe(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("get me failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"user": user,
	}, "User profile retrieved")
}

// UploadKYCDocument handles KYC document upload.
func (h *AuthHandler) UploadKYCDocument(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	// Parse multipart form. The deployed frontend sends the file under
	// "document"; newer builds use "file" — accept both.
	file, header, err := c.Request.FormFile("file")
	if err != nil && errors.Is(err, http.ErrMissingFile) {
		file, header, err = c.Request.FormFile("document")
	}
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Success: false,
				Message: "File is too large. Maximum request size is 15MB",
				Code:    "PAYLOAD_TOO_LARGE",
			})
			return
		}
		h.logger.Warn().Err(err).Msg("kyc upload: failed to parse multipart file")
		BadRequest(c, "File is required")
		return
	}
	defer file.Close()

	// Validate document type
	docTypeStr := c.PostForm("documentType")
	var docType domain.KYCDocumentType
	switch docTypeStr {
	case string(domain.KYCDocumentTypeIDCard):
		docType = domain.KYCDocumentTypeIDCard
	case string(domain.KYCDocumentTypePassport):
		docType = domain.KYCDocumentTypePassport
	case string(domain.KYCDocumentTypeSelfie):
		docType = domain.KYCDocumentTypeSelfie
	default:
		BadRequest(c, "Invalid document type. Must be one of: id_card, passport, selfie")
		return
	}

	// Create upload directory
	uploadDir := "uploads/kyc"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error().Err(err).Msg("failed to create upload directory")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to process upload",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filePath := filepath.Join(uploadDir, filename)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to create file")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to save file",
			Code:    "INTERNAL_ERROR",
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to write file")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "Failed to save file",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Call KYC service
	if err := h.kycService.UploadDocument(c.Request.Context(), userID, filePath, docType); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("kyc upload failed")
		// Clean up the saved file on error
		_ = os.Remove(filePath)
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"filePath":     filePath,
		"documentType": docType,
	}, "KYC document uploaded successfully")
}

// VerifyKYC handles KYC verification.
func (h *AuthHandler) VerifyKYC(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	if err := h.kycService.AutoVerify(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("kyc verification failed")
		Error(c, err)
		return
	}

	// Report the resulting status so the frontend can distinguish
	// auto-verified from pending manual review.
	verified := false
	kycStatus := domain.KYCStatusPending
	if user, statusErr := h.kycService.GetStatus(c.Request.Context(), userID); statusErr == nil {
		kycStatus = user.KYCStatus
		verified = user.KYCStatus == domain.KYCStatusVerified
	}

	Success(c, gin.H{
		"verified":  verified,
		"kycStatus": kycStatus,
	}, "KYC verification initiated successfully")
}

// GetKYCStatus returns the KYC verification status.
func (h *AuthHandler) GetKYCStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.kycService.GetStatus(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("get kyc status failed")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"kycStatus":          user.KYCStatus,
		"kycRejectionReason": user.KYCRejectionReason,
		"kycVerifiedAt":      user.KYCVerifiedAt,
		"faceMatchScore":     user.FaceMatchScore,
	}, "KYC status retrieved")
}
