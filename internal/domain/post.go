package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Post represents a listing for a game skin or profile.
type Post struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	UserID        uuid.UUID       `json:"userId" db:"user_id"`
	Title         string          `json:"title" db:"title"`
	Description   string          `json:"description,omitempty" db:"description"`
	Price         float64         `json:"price" db:"price"`
	Currency      Currency        `json:"currency" db:"currency"`
	GameID        uuid.UUID       `json:"gameId" db:"game_id"`
	Genre         string          `json:"genre,omitempty" db:"genre"`
	Type          PostType        `json:"type" db:"type"`
	ContactInfo   json.RawMessage `json:"contactInfo,omitempty" db:"contact_info"`
	Status        PostStatus      `json:"status" db:"status"`
	ViewsCount    int             `json:"viewsCount" db:"views_count"`
	ReportsCount  int             `json:"reportsCount" db:"reports_count"`
	ReportedAt    *time.Time      `json:"reportedAt,omitempty" db:"reported_at"`
	DeletedAt     *time.Time      `json:"deletedAt,omitempty" db:"deleted_at"`
	DeletedBy     *uuid.UUID      `json:"deletedBy,omitempty" db:"deleted_by"`
	DeletedByType *string         `json:"deletedByType,omitempty" db:"deleted_by_type"`
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time       `json:"updatedAt" db:"updated_at"`

	// Joined fields (not stored in posts table directly)
	Images []*PostImage        `json:"images,omitempty" db:"-"`
	Seller *PostSellerInfo     `json:"seller,omitempty" db:"-"`
	Game   *PostGameInfo       `json:"game,omitempty" db:"-"`
}

// PostSellerInfo is the public seller info nested in post responses.
type PostSellerInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   string    `json:"avatarUrl,omitempty"`
	KYCStatus   KYCStatus `json:"kycStatus"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PostGameInfo is the game info nested in post responses.
type PostGameInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
	Icon string    `json:"icon,omitempty"`
}

// PostImage represents an image attached to a post.
type PostImage struct {
	ID            uuid.UUID `json:"id" db:"id"`
	PostID        uuid.UUID `json:"postId" db:"post_id"`
	OriginalPath  string    `json:"originalPath" db:"original_path"`
	ThumbnailPath *string   `json:"thumbnailPath,omitempty" db:"thumbnail_path"`
	Filename      string    `json:"filename" db:"filename"`
	Size          int       `json:"size" db:"size"`
	MimeType      string    `json:"mimeType" db:"mime_type"`
	SortOrder     int16     `json:"sortOrder" db:"sort_order"`
	UploadedAt    time.Time `json:"uploadedAt" db:"uploaded_at"`
}

// PostListFilter provides filtering and pagination for post listing.
type PostListFilter struct {
	Page           int         `json:"page"`
	Limit          int         `json:"limit"`
	Search         string      `json:"search,omitempty"`
	Status         *PostStatus `json:"status,omitempty"`
	Type           *PostType   `json:"type,omitempty"`
	GameID         *uuid.UUID  `json:"gameId,omitempty"`
	UserID         *uuid.UUID  `json:"userId,omitempty"`
	Currency       *Currency   `json:"currency,omitempty"`
	MinPrice       *float64    `json:"minPrice,omitempty"`
	MaxPrice       *float64    `json:"maxPrice,omitempty"`
	SortBy         string      `json:"sortBy,omitempty"`
	SortOrder      SortOrder   `json:"sortOrder,omitempty"`
	IncludeDeleted bool        `json:"includeDeleted,omitempty"` // Admin only: include soft-deleted posts
}

// PostRepository defines the interface for post data access.
type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	FindByID(ctx context.Context, id uuid.UUID) (*Post, error)
	FindActive(ctx context.Context, limit, offset int) ([]*Post, int64, error)
	FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*Post, int64, error)
	FindByUserWithStatus(ctx context.Context, userID uuid.UUID, status *PostStatus, limit, offset int) ([]*Post, int64, error)
	Search(ctx context.Context, query string, limit, offset int) ([]*Post, int64, error)
	Update(ctx context.Context, post *Post) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID, deletedByType string) error
	IncrementViews(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status PostStatus) error
	AddImage(ctx context.Context, image *PostImage) error
	RemoveImage(ctx context.Context, imageID uuid.UUID) error
	ListWithFilters(ctx context.Context, filter PostListFilter) ([]*Post, int64, error)
	CountByStatus(ctx context.Context, status PostStatus) (int64, error)
	CountAll(ctx context.Context) (int64, error)
	FindRecent(ctx context.Context, limit int) ([]*Post, error)
}
