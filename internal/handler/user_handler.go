package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	"skintrader-go/internal/service"
)

type UserHandler struct {
	userRepo       domain.UserRepository
	postRepo       domain.PostRepository
	imageService   *service.ImageService
	authMiddleware *middleware.AuthMiddleware
	logger         zerolog.Logger
}

func NewUserHandler(
	userRepo domain.UserRepository,
	postRepo domain.PostRepository,
	imageService *service.ImageService,
	authMiddleware *middleware.AuthMiddleware,
	logger zerolog.Logger,
) *UserHandler {
	return &UserHandler{
		userRepo:       userRepo,
		postRepo:       postRepo,
		imageService:   imageService,
		authMiddleware: authMiddleware,
		logger:         logger.With().Str("handler", "user").Logger(),
	}
}

// GetProfile returns the authenticated user's profile.
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to get profile")
		Error(c, err)
		return
	}

	Success(c, gin.H{"user": user}, "Profile retrieved successfully")
}

// UpdateProfile updates the authenticated user's profile.
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		DisplayName *string          `json:"displayName"`
		Bio         *string          `json:"bio"`
		PhoneNumber *string          `json:"phoneNumber"`
		Language    *string          `json:"language"`
		SocialMedia *json.RawMessage `json:"socialMedia"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Invalid request body")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to find user for update")
		Error(c, err)
		return
	}

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" {
			BadRequest(c, "Display name cannot be empty")
			return
		}
		if len(name) > 50 {
			BadRequest(c, "Display name must be 50 characters or less")
			return
		}
		user.DisplayName = name
	}

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len(bio) > 500 {
			BadRequest(c, "Bio must be 500 characters or less")
			return
		}
		user.Bio = bio
	}

	if req.PhoneNumber != nil {
		user.PhoneNumber = strings.TrimSpace(*req.PhoneNumber)
	}

	if req.Language != nil {
		lang := domain.Language(*req.Language)
		switch lang {
		case domain.LanguageEN, domain.LanguageRU, domain.LanguageUZ:
			user.Language = lang
		default:
			BadRequest(c, "Invalid language. Must be one of: en, ru, uz")
			return
		}
	}

	if req.SocialMedia != nil {
		user.SocialMedia = *req.SocialMedia
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to update profile")
		Error(c, err)
		return
	}

	Success(c, gin.H{"user": user}, "Profile updated successfully")
}

// UpdateAvatar updates the authenticated user's avatar.
func (h *UserHandler) UpdateAvatar(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		BadRequest(c, "File is required")
		return
	}
	defer file.Close()

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowedExts[ext] {
		BadRequest(c, "Invalid file type. Allowed: jpg, jpeg, png, webp")
		return
	}

	// Create upload directory
	if err := h.imageService.EnsureUploadDir("avatars"); err != nil {
		h.logger.Error().Err(err).Msg("failed to create avatar upload directory")
		BadRequest(c, "Failed to process upload")
		return
	}

	// Generate unique filename and save
	filename := h.imageService.GenerateFilename(ext)
	filePath := filepath.Join(h.imageService.GetUploadPath("avatars"), filename)

	dst, err := os.Create(filePath)
	if err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to create avatar file")
		BadRequest(c, "Failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to write avatar file")
		_ = os.Remove(filePath)
		BadRequest(c, "Failed to save file")
		return
	}

	// Process avatar (resize)
	processedPath, err := h.imageService.ProcessAvatar(filePath)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to process avatar")
		_ = os.Remove(filePath)
		BadRequest(c, "Failed to process avatar image")
		return
	}

	// Update user avatar URL
	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to find user for avatar update")
		_ = os.Remove(processedPath)
		Error(c, err)
		return
	}

	// Remove old avatar if exists
	if user.AvatarURL != "" {
		_ = h.imageService.CleanupFile(user.AvatarURL)
	}

	user.AvatarURL = fmt.Sprintf("/uploads/avatars/%s", filename)
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to update avatar")
		_ = os.Remove(processedPath)
		Error(c, err)
		return
	}

	Success(c, gin.H{"avatarUrl": user.AvatarURL}, "Avatar updated successfully")
}

// UpdateLocation updates the authenticated user's location.
func (h *UserHandler) UpdateLocation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Latitude and longitude are required")
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		BadRequest(c, "Latitude must be between -90 and 90")
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		BadRequest(c, "Longitude must be between -180 and 180")
		return
	}

	if err := h.userRepo.UpdateLocation(c.Request.Context(), userID, req.Latitude, req.Longitude); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to update location")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"latitude":  req.Latitude,
		"longitude": req.Longitude,
	}, "Location updated successfully")
}

// GetNearbyUsers returns users near the authenticated user's location.
func (h *UserHandler) GetNearbyUsers(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to find user for nearby search")
		Error(c, err)
		return
	}

	if user.Latitude == nil || user.Longitude == nil {
		BadRequest(c, "You must set your location first")
		return
	}

	radiusStr := c.DefaultQuery("radius", "10")
	radius, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil || radius <= 0 || radius > 100 {
		BadRequest(c, "Radius must be a positive number up to 100 km")
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	nearbyUsers, err := h.userRepo.FindNearby(c.Request.Context(), *user.Latitude, *user.Longitude, radius, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to find nearby users")
		Error(c, err)
		return
	}

	Success(c, gin.H{"users": nearbyUsers}, "Nearby users retrieved successfully")
}

// DeleteAccount handles account deletion (soft delete).
func (h *UserHandler) DeleteAccount(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), userID); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to delete account")
		Error(c, err)
		return
	}

	Success(c, nil, "Account deleted successfully")
}

// GetPublicProfile returns a user's public profile by ID.
func (h *UserHandler) GetPublicProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", idStr).Msg("failed to get public profile")
		NotFound(c, "User not found")
		return
	}

	// Return only public fields
	Success(c, gin.H{
		"id":           user.ID,
		"displayName":  user.DisplayName,
		"bio":          user.Bio,
		"avatarUrl":    user.AvatarURL,
		"postsCount":   user.PostsCount,
		"kycStatus":    user.KYCStatus,
		"createdAt":    user.CreatedAt,
		"lastActiveAt": user.LastActiveAt,
	}, "Public profile retrieved successfully")
}

// GetUserPosts returns a user's posts.
func (h *UserHandler) GetUserPosts(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid user ID")
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	posts, total, err := h.postRepo.FindByUser(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", idStr).Msg("failed to get user posts")
		Error(c, err)
		return
	}

	Paginated(c, gin.H{"posts": posts}, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, "User posts retrieved successfully")
}
