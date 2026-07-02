package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	"skintrader-go/internal/service"
)

type PostHandler struct {
	postRepo       domain.PostRepository
	userRepo       domain.UserRepository
	gameRepo       domain.GameRepository
	imageService   *service.ImageService
	authMiddleware *middleware.AuthMiddleware
	logger         zerolog.Logger
}

func NewPostHandler(
	postRepo domain.PostRepository,
	userRepo domain.UserRepository,
	gameRepo domain.GameRepository,
	imageService *service.ImageService,
	authMiddleware *middleware.AuthMiddleware,
	logger zerolog.Logger,
) *PostHandler {
	return &PostHandler{
		postRepo:       postRepo,
		userRepo:       userRepo,
		gameRepo:       gameRepo,
		imageService:   imageService,
		authMiddleware: authMiddleware,
		logger:         logger.With().Str("handler", "post").Logger(),
	}
}

// GetPosts returns a list of active posts with filtering, sorting, and cursor-compatible pagination.
func (h *PostHandler) GetPosts(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "12")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 12
	}

	// Support both cursor (offset encoded) and offset params
	offset := 0
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		if v, err := strconv.Atoi(cursorStr); err == nil && v >= 0 {
			offset = v
		}
	} else if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	// Build filter — public endpoint always shows active, non-deleted posts
	activeStatus := domain.PostStatusActive
	filter := domain.PostListFilter{
		Page:      (offset / limit) + 1,
		Limit:     limit,
		Status:    &activeStatus,
		Search:    c.Query("q"),
		SortBy:    c.DefaultQuery("sortBy", "created_at"),
		SortOrder: parseSortOrder(c.DefaultQuery("sortOrder", "desc")),
	}

	if gameID := c.Query("gameId"); gameID != "" {
		if id, err := uuid.Parse(gameID); err == nil {
			filter.GameID = &id
		}
	}

	if postType := c.Query("type"); postType != "" {
		t := domain.PostType(postType)
		filter.Type = &t
	}

	if minPrice := c.Query("minPrice"); minPrice != "" {
		if v, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter.MinPrice = &v
		}
	}

	if maxPrice := c.Query("maxPrice"); maxPrice != "" {
		if v, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filter.MaxPrice = &v
		}
	}

	posts, total, err := h.postRepo.ListWithFilters(c.Request.Context(), filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get posts")
		Error(c, err)
		return
	}

	// Enrich posts with seller and game data
	h.enrichPosts(c.Request.Context(), posts)

	// Build cursor-compatible pagination
	nextOffset := offset + limit
	hasMore := int64(nextOffset) < total
	var nextCursor *string
	if hasMore {
		s := strconv.Itoa(nextOffset)
		nextCursor = &s
	}

	Paginated(c, gin.H{"posts": posts}, gin.H{
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"hasMore":    hasMore,
		"nextCursor": nextCursor,
	}, "Posts retrieved successfully")
}

// SearchPosts searches for posts using full-text search.
func (h *PostHandler) SearchPosts(c *gin.Context) {
	query := c.Query("q")
	if strings.TrimSpace(query) == "" {
		BadRequest(c, "Search query is required")
		return
	}

	limitStr := c.DefaultQuery("limit", "12")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 12
	}

	offset := 0
	if cursorStr := c.Query("cursor"); cursorStr != "" {
		if v, err := strconv.Atoi(cursorStr); err == nil && v >= 0 {
			offset = v
		}
	} else if offsetStr := c.Query("offset"); offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}

	posts, total, err := h.postRepo.Search(c.Request.Context(), query, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("query", query).Msg("failed to search posts")
		Error(c, err)
		return
	}

	// Enrich posts
	h.enrichPosts(c.Request.Context(), posts)

	nextOffset := offset + limit
	hasMore := int64(nextOffset) < total
	var nextCursor *string
	if hasMore {
		s := strconv.Itoa(nextOffset)
		nextCursor = &s
	}

	Paginated(c, gin.H{"posts": posts}, gin.H{
		"total":      total,
		"limit":      limit,
		"offset":     offset,
		"hasMore":    hasMore,
		"nextCursor": nextCursor,
		"query":      query,
	}, "Search results retrieved successfully")
}

// GetMyPosts returns the authenticated user's posts, optionally filtered by status.
func (h *PostHandler) GetMyPosts(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
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

	// Optional status filter (active, sold, draft)
	var statusFilter *domain.PostStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := domain.PostStatus(statusStr)
		switch s {
		case domain.PostStatusActive, domain.PostStatusSold, domain.PostStatusDraft:
			statusFilter = &s
		}
	}

	posts, total, err := h.postRepo.FindByUserWithStatus(c.Request.Context(), userID, statusFilter, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to get my posts")
		Error(c, err)
		return
	}

	// Enrich posts
	h.enrichPosts(c.Request.Context(), posts)

	Paginated(c, gin.H{"posts": posts}, gin.H{
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, "Your posts retrieved successfully")
}

// CreatePost creates a new post. Accepts multipart/form-data with text fields and optional image files.
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	// Read text fields from form (works with both multipart and url-encoded)
	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		BadRequest(c, "Title is required")
		return
	}
	if len(title) > 200 {
		BadRequest(c, "Title must be 200 characters or less")
		return
	}

	priceStr := c.PostForm("price")
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil || price <= 0 || price > 1_000_000_000 {
		BadRequest(c, "Price must be a number between 0 and 1,000,000,000")
		return
	}

	gameIDStr := c.PostForm("gameId")
	gameID, err := uuid.Parse(gameIDStr)
	if err != nil {
		BadRequest(c, "Invalid or missing gameId")
		return
	}

	description := strings.TrimSpace(c.PostForm("description"))

	// Validate currency
	currency := domain.CurrencyUZS
	if cur := c.PostForm("currency"); cur != "" {
		switch domain.Currency(cur) {
		case domain.CurrencyUZS, domain.CurrencyUSD:
			currency = domain.Currency(cur)
		default:
			BadRequest(c, "Invalid currency. Must be UZS or USD")
			return
		}
	}

	// Validate post type
	postType := domain.PostTypeSkin
	if t := c.PostForm("type"); t != "" {
		switch domain.PostType(t) {
		case domain.PostTypeSkin, domain.PostTypeProfile:
			postType = domain.PostType(t)
		default:
			BadRequest(c, "Invalid post type. Must be skin or profile")
			return
		}
	}

	genre := c.PostForm("genre")

	var contactInfo json.RawMessage
	if ci := c.PostForm("contactInfo"); ci != "" {
		contactInfo = json.RawMessage(ci)
	}

	post := &domain.Post{
		UserID:      userID,
		Title:       title,
		Description: description,
		Price:       price,
		Currency:    currency,
		GameID:      gameID,
		Genre:       genre,
		Type:        postType,
		ContactInfo: contactInfo,
		Status:      domain.PostStatusActive,
	}

	if err := h.postRepo.Create(c.Request.Context(), post); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to create post")
		Error(c, err)
		return
	}

	// Handle image uploads if present in the same request
	form, _ := c.MultipartForm()
	if form != nil {
		files := form.File["images"]
		if len(files) == 0 {
			// Frontend may use "files" key as well
			files = form.File["files"]
		}
		if len(files) > 0 {
			uploadedImages := h.processImageUploads(c, post.ID, post, files)
			post.Images = uploadedImages
		}
	}

	// Enrich with seller and game info
	h.enrichPost(c.Request.Context(), post)

	Created(c, gin.H{"post": post}, "Post created successfully")
}

// GetPost returns a specific post by ID and increments views.
func (h *PostHandler) GetPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	post, err := h.postRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to get post")
		NotFound(c, "Post not found")
		return
	}

	// Increment views in the background (non-blocking, use detached context)
	go func(postID uuid.UUID) {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error().Interface("panic", r).Str("postID", postID.String()).Msg("panic in increment views goroutine")
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if viewErr := h.postRepo.IncrementViews(ctx, postID); viewErr != nil {
			h.logger.Warn().Err(viewErr).Str("postID", postID.String()).Msg("failed to increment views")
		}
	}(id)

	// Enrich post with seller and game info directly on the post object
	h.enrichPost(c.Request.Context(), post)

	Success(c, gin.H{"post": post}, "Post retrieved successfully")
}

// UpdatePost updates an existing post (owner only). Accepts multipart/form-data.
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	post, err := h.postRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to find post for update")
		NotFound(c, "Post not found")
		return
	}

	if post.UserID != userID {
		Forbidden(c, "You can only update your own posts")
		return
	}

	// Read form fields (multipart or url-encoded)
	if title := c.PostForm("title"); title != "" {
		title = strings.TrimSpace(title)
		if title == "" {
			BadRequest(c, "Title cannot be empty")
			return
		}
		if len(title) > 200 {
			BadRequest(c, "Title must be 200 characters or less")
			return
		}
		post.Title = title
	}

	if desc := c.PostForm("description"); desc != "" {
		post.Description = strings.TrimSpace(desc)
	}

	if priceStr := c.PostForm("price"); priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 || price > 1_000_000_000 {
			BadRequest(c, "Price must be a number between 0 and 1,000,000,000")
			return
		}
		post.Price = price
	}

	if cur := c.PostForm("currency"); cur != "" {
		switch domain.Currency(cur) {
		case domain.CurrencyUZS, domain.CurrencyUSD:
			post.Currency = domain.Currency(cur)
		default:
			BadRequest(c, "Invalid currency. Must be UZS or USD")
			return
		}
	}

	if gameIDStr := c.PostForm("gameId"); gameIDStr != "" {
		gameID, err := uuid.Parse(gameIDStr)
		if err != nil {
			BadRequest(c, "Invalid game_id")
			return
		}
		post.GameID = gameID
	}

	if genre := c.PostForm("genre"); genre != "" {
		post.Genre = genre
	}

	if t := c.PostForm("type"); t != "" {
		switch domain.PostType(t) {
		case domain.PostTypeSkin, domain.PostTypeProfile:
			post.Type = domain.PostType(t)
		default:
			BadRequest(c, "Invalid post type. Must be skin or profile")
			return
		}
	}

	if ci := c.PostForm("contactInfo"); ci != "" {
		post.ContactInfo = json.RawMessage(ci)
	}

	if err := h.postRepo.Update(c.Request.Context(), post); err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to update post")
		Error(c, err)
		return
	}

	// Handle new image uploads if present
	form, _ := c.MultipartForm()
	if form != nil {
		files := form.File["images"]
		if len(files) == 0 {
			files = form.File["files"]
		}
		if len(files) > 0 {
			uploadedImages := h.processImageUploads(c, post.ID, post, files)
			// Reload all images for the post
			if len(uploadedImages) > 0 {
				if allImages, err := h.postRepo.FindByID(c.Request.Context(), post.ID); err == nil {
					post.Images = allImages.Images
				}
			}
		}
	}

	// Enrich with seller and game info
	h.enrichPost(c.Request.Context(), post)

	Success(c, gin.H{"post": post}, "Post updated successfully")
}

// UpdatePostStatus changes the status of a post (owner only).
func (h *PostHandler) UpdatePostStatus(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	post, err := h.postRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to find post for status update")
		NotFound(c, "Post not found")
		return
	}

	if post.UserID != userID {
		Forbidden(c, "You can only update your own posts")
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "Status is required")
		return
	}

	status := domain.PostStatus(req.Status)
	switch status {
	case domain.PostStatusActive, domain.PostStatusSold, domain.PostStatusDraft:
		// valid
	default:
		BadRequest(c, "Invalid status. Must be active, sold, or draft")
		return
	}

	if err := h.postRepo.UpdateStatus(c.Request.Context(), id, status); err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to update post status")
		Error(c, err)
		return
	}

	Success(c, gin.H{"status": status}, "Post status updated successfully")
}

// DeletePost soft deletes a post (owner only).
func (h *PostHandler) DeletePost(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	post, err := h.postRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to find post for deletion")
		NotFound(c, "Post not found")
		return
	}

	if post.UserID != userID {
		Forbidden(c, "You can only delete your own posts")
		return
	}

	if err := h.postRepo.SoftDelete(c.Request.Context(), id, userID, "user"); err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to delete post")
		Error(c, err)
		return
	}

	Success(c, nil, "Post deleted successfully")
}

// AddImages handles multipart image uploads for a post.
func (h *PostHandler) AddImages(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	idStr := c.Param("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	post, err := h.postRepo.FindByID(c.Request.Context(), postID)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to find post for image upload")
		NotFound(c, "Post not found")
		return
	}

	if post.UserID != userID {
		Forbidden(c, "You can only add images to your own posts")
		return
	}

	// Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		BadRequest(c, "Failed to parse multipart form")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		BadRequest(c, "At least one file is required")
		return
	}

	if len(files) > 10 {
		BadRequest(c, "Maximum 10 images per upload")
		return
	}

	// Ensure upload directory exists
	if err := h.imageService.EnsureUploadDir("posts"); err != nil {
		h.logger.Error().Err(err).Msg("failed to create post upload directory")
		BadRequest(c, "Failed to process upload")
		return
	}

	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	var uploadedImages []*domain.PostImage

	for i, fileHeader := range files {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !allowedExts[ext] {
			BadRequest(c, fmt.Sprintf("Invalid file type for '%s'. Allowed: jpg, jpeg, png, webp", fileHeader.Filename))
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			h.logger.Error().Err(err).Str("filename", fileHeader.Filename).Msg("failed to open uploaded file")
			continue
		}

		filename := h.imageService.GenerateFilename(ext)
		filePath := filepath.Join(h.imageService.GetUploadPath("posts"), filename)

		dst, err := os.Create(filePath)
		if err != nil {
			file.Close()
			h.logger.Error().Err(err).Str("path", filePath).Msg("failed to create file")
			continue
		}

		if _, err := io.Copy(dst, file); err != nil {
			file.Close()
			dst.Close()
			_ = os.Remove(filePath)
			h.logger.Error().Err(err).Str("path", filePath).Msg("failed to write file")
			continue
		}
		file.Close()
		dst.Close()

		// Process image
		processed, err := h.imageService.ProcessPostImage(filePath)
		if err != nil {
			h.logger.Warn().Err(err).Str("path", filePath).Msg("failed to process image, using original")
		}

		var thumbnailPath *string
		if processed != nil && processed.ThumbnailPath != "" {
			tp := fmt.Sprintf("/uploads/posts/%s", filepath.Base(processed.ThumbnailPath))
			thumbnailPath = &tp
		}

		mimeType := "image/jpeg"
		size := int(fileHeader.Size)
		if processed != nil {
			mimeType = processed.MimeType
			size = int(processed.Size)
		}

		image := &domain.PostImage{
			PostID:        postID,
			OriginalPath:  fmt.Sprintf("/uploads/posts/%s", filename),
			ThumbnailPath: thumbnailPath,
			Filename:      fileHeader.Filename,
			Size:          size,
			MimeType:      mimeType,
			SortOrder:     int16(len(post.Images) + i),
		}

		if err := h.postRepo.AddImage(c.Request.Context(), image); err != nil {
			h.logger.Error().Err(err).Str("postID", idStr).Msg("failed to add image to post")
			_ = os.Remove(filePath)
			continue
		}

		uploadedImages = append(uploadedImages, image)
	}

	if len(uploadedImages) == 0 {
		BadRequest(c, "No images were uploaded successfully")
		return
	}

	Created(c, gin.H{"images": uploadedImages}, fmt.Sprintf("%d image(s) uploaded successfully", len(uploadedImages)))
}

// RemoveImage deletes a post image.
func (h *PostHandler) RemoveImage(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		BadRequest(c, "Invalid post ID")
		return
	}

	imageIDStr := c.Param("imageId")
	imageID, err := uuid.Parse(imageIDStr)
	if err != nil {
		BadRequest(c, "Invalid image ID")
		return
	}

	// Verify post ownership
	post, err := h.postRepo.FindByID(c.Request.Context(), postID)
	if err != nil {
		h.logger.Error().Err(err).Str("postID", postIDStr).Msg("failed to find post for image removal")
		NotFound(c, "Post not found")
		return
	}

	if post.UserID != userID {
		Forbidden(c, "You can only remove images from your own posts")
		return
	}

	if err := h.postRepo.RemoveImage(c.Request.Context(), imageID); err != nil {
		h.logger.Error().Err(err).Str("imageID", imageIDStr).Msg("failed to remove image")
		Error(c, err)
		return
	}

	Success(c, nil, "Image removed successfully")
}

// ==========================================================================
// Helper methods
// ==========================================================================

// enrichPost populates Seller and Game info on a single post.
func (h *PostHandler) enrichPost(ctx context.Context, post *domain.Post) {
	if post == nil {
		return
	}
	// Seller info
	if seller, err := h.userRepo.FindByID(ctx, post.UserID); err == nil && seller != nil {
		post.Seller = &domain.PostSellerInfo{
			ID:          seller.ID,
			DisplayName: seller.DisplayName,
			AvatarURL:   seller.AvatarURL,
			KYCStatus:   seller.KYCStatus,
			CreatedAt:   seller.CreatedAt,
		}
	}
	// Game info
	if game, err := h.gameRepo.FindByID(ctx, post.GameID); err == nil && game != nil {
		post.Game = &domain.PostGameInfo{
			ID:   game.ID,
			Name: game.Name,
			Slug: game.Slug,
			Icon: game.Icon,
		}
	}
}

// enrichPosts populates Seller and Game info for a list of posts (batch).
func (h *PostHandler) enrichPosts(ctx context.Context, posts []*domain.Post) {
	if len(posts) == 0 {
		return
	}

	// Collect unique user IDs and game IDs
	userIDs := make(map[uuid.UUID]bool)
	gameIDs := make(map[uuid.UUID]bool)
	for _, p := range posts {
		userIDs[p.UserID] = true
		gameIDs[p.GameID] = true
	}

	// Batch load users
	users := make(map[uuid.UUID]*domain.PostSellerInfo)
	for uid := range userIDs {
		if u, err := h.userRepo.FindByID(ctx, uid); err == nil && u != nil {
			users[uid] = &domain.PostSellerInfo{
				ID:          u.ID,
				DisplayName: u.DisplayName,
				AvatarURL:   u.AvatarURL,
				KYCStatus:   u.KYCStatus,
				CreatedAt:   u.CreatedAt,
			}
		}
	}

	// Batch load games
	games := make(map[uuid.UUID]*domain.PostGameInfo)
	for gid := range gameIDs {
		if g, err := h.gameRepo.FindByID(ctx, gid); err == nil && g != nil {
			games[gid] = &domain.PostGameInfo{
				ID:   g.ID,
				Name: g.Name,
				Slug: g.Slug,
				Icon: g.Icon,
			}
		}
	}

	// Assign to posts
	for _, p := range posts {
		p.Seller = users[p.UserID]
		p.Game = games[p.GameID]
	}
}

// processImageUploads handles image file uploads for a post (used by CreatePost and UpdatePost).
func (h *PostHandler) processImageUploads(c *gin.Context, postID uuid.UUID, post *domain.Post, files []*multipart.FileHeader) []*domain.PostImage {
	if len(files) > 10 {
		files = files[:10]
	}

	if err := h.imageService.EnsureUploadDir("posts"); err != nil {
		h.logger.Error().Err(err).Msg("failed to create post upload directory")
		return nil
	}

	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	var uploadedImages []*domain.PostImage

	for i, fileHeader := range files {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !allowedExts[ext] {
			continue
		}

		file, err := fileHeader.Open()
		if err != nil {
			h.logger.Error().Err(err).Str("filename", fileHeader.Filename).Msg("failed to open uploaded file")
			continue
		}

		filename := h.imageService.GenerateFilename(ext)
		filePath := filepath.Join(h.imageService.GetUploadPath("posts"), filename)

		dst, err := os.Create(filePath)
		if err != nil {
			file.Close()
			h.logger.Error().Err(err).Str("path", filePath).Msg("failed to create file")
			continue
		}

		if _, err := io.Copy(dst, file); err != nil {
			file.Close()
			dst.Close()
			_ = os.Remove(filePath)
			h.logger.Error().Err(err).Str("path", filePath).Msg("failed to write file")
			continue
		}
		file.Close()
		dst.Close()

		processed, err := h.imageService.ProcessPostImage(filePath)
		if err != nil {
			h.logger.Warn().Err(err).Str("path", filePath).Msg("failed to process image, using original")
		}

		var thumbnailPath *string
		if processed != nil && processed.ThumbnailPath != "" {
			tp := fmt.Sprintf("/uploads/posts/%s", filepath.Base(processed.ThumbnailPath))
			thumbnailPath = &tp
		}

		mimeType := "image/jpeg"
		size := int(fileHeader.Size)
		if processed != nil {
			mimeType = processed.MimeType
			size = int(processed.Size)
		}

		existingCount := 0
		if post != nil && post.Images != nil {
			existingCount = len(post.Images)
		}

		image := &domain.PostImage{
			PostID:        postID,
			OriginalPath:  fmt.Sprintf("/uploads/posts/%s", filename),
			ThumbnailPath: thumbnailPath,
			Filename:      fileHeader.Filename,
			Size:          size,
			MimeType:      mimeType,
			SortOrder:     int16(existingCount + i),
		}

		if err := h.postRepo.AddImage(c.Request.Context(), image); err != nil {
			h.logger.Error().Err(err).Str("postID", postID.String()).Msg("failed to add image to post")
			_ = os.Remove(filePath)
			continue
		}

		uploadedImages = append(uploadedImages, image)
	}

	return uploadedImages
}
