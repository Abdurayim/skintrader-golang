package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// User represents a user account in the system.
type User struct {
	ID            uuid.UUID        `json:"id" db:"id"`
	AuthProvider  AuthProvider     `json:"authProvider" db:"auth_provider"`
	GoogleID      *string          `json:"googleId,omitempty" db:"google_id"`
	AppleID       *string          `json:"appleId,omitempty" db:"apple_id"`
	Email         string           `json:"email" db:"email"`
	PasswordHash  *string          `json:"-" db:"password_hash"`
	EmailVerified bool             `json:"emailVerified" db:"email_verified"`
	DisplayName   string           `json:"displayName" db:"display_name"`
	PhoneNumber   string           `json:"phoneNumber,omitempty" db:"phone_number"`
	Bio           string           `json:"bio,omitempty" db:"bio"`
	AvatarURL     string           `json:"avatarUrl,omitempty" db:"avatar_url"`
	SocialMedia   json.RawMessage  `json:"socialMedia,omitempty" db:"social_media"`
	Language      Language         `json:"language" db:"language"`
	Status        UserStatus       `json:"status" db:"status"`
	StatusReason  string           `json:"statusReason,omitempty" db:"status_reason"`

	// KYC fields
	KYCStatus          KYCStatus  `json:"kycStatus" db:"kyc_status"`
	KYCRejectionReason string     `json:"kycRejectionReason,omitempty" db:"kyc_rejection_reason"`
	KYCVerifiedAt      *time.Time `json:"kycVerifiedAt,omitempty" db:"kyc_verified_at"`
	KYCReviewedBy      *uuid.UUID `json:"kycReviewedBy,omitempty" db:"kyc_reviewed_by"`
	FaceMatchScore     *float32   `json:"faceMatchScore,omitempty" db:"face_match_score"`

	// Location fields
	Latitude          *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude         *float64   `json:"longitude,omitempty" db:"longitude"`
	LocationUpdatedAt *time.Time `json:"locationUpdatedAt,omitempty" db:"location_updated_at"`

	// Balance in Uzbek so'm (UZS), spent on publishing posts
	Balance int64 `json:"balance" db:"balance"`

	// Stats
	PostsCount      int `json:"postsCount" db:"posts_count"`
	ReportsReceived int `json:"reportsReceived" db:"reports_received"`
	ReportsMade     int `json:"reportsMade" db:"reports_made"`

	// Subscription fields
	SubscriptionStatus  SubscriptionStatus `json:"subscriptionStatus" db:"subscription_status"`
	CurrentSubscriptionID *uuid.UUID       `json:"currentSubscriptionId,omitempty" db:"current_subscription_id"`
	SubscriptionExpiresAt *time.Time       `json:"subscriptionExpiresAt,omitempty" db:"subscription_expires_at"`
	GracePeriodEndsAt     *time.Time       `json:"gracePeriodEndsAt,omitempty" db:"grace_period_ends_at"`

	// Timestamps
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	LastActiveAt *time.Time `json:"lastActiveAt,omitempty" db:"last_active_at"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time  `json:"updatedAt" db:"updated_at"`

	// Computed fields (not persisted)
	IsNewUser    bool           `json:"isNewUser,omitempty" db:"-"`
	KYCDocuments []*KYCDocument `json:"kycDocuments,omitempty" db:"-"`
}

// KYCDocument represents a KYC verification document uploaded by a user.
type KYCDocument struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	UserID     uuid.UUID       `json:"userId" db:"user_id"`
	DocType    KYCDocumentType `json:"type" db:"doc_type"`
	FilePath   string          `json:"filePath" db:"file_path"`
	UploadedAt time.Time       `json:"uploadedAt" db:"uploaded_at"`
	VerifiedAt *time.Time      `json:"verifiedAt,omitempty" db:"verified_at"`
}

// UserListFilter provides filtering and pagination for admin user listing.
type UserListFilter struct {
	Page               int                `json:"page"`
	Limit              int                `json:"limit"`
	Search             string             `json:"search,omitempty"`
	Status             *UserStatus        `json:"status,omitempty"`
	KYCStatus          *KYCStatus         `json:"kycStatus,omitempty"`
	SubscriptionStatus *SubscriptionStatus `json:"subscriptionStatus,omitempty"`
	SortBy             string             `json:"sortBy,omitempty"`
	SortOrder          SortOrder          `json:"sortOrder,omitempty"`
}

// UserRepository defines the interface for user data access.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	FindByAppleID(ctx context.Context, appleID string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateKYCStatus(ctx context.Context, userID uuid.UUID, status KYCStatus, reviewedBy uuid.UUID, reason string) error
	UpdateSubscriptionStatus(ctx context.Context, userID uuid.UUID, status SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error
	UpdateLocation(ctx context.Context, userID uuid.UUID, latitude, longitude float64) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindNearby(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*User, error)
	SearchByName(ctx context.Context, query string, limit, offset int) ([]*User, int, error)
	CountByStatus(ctx context.Context, status UserStatus) (int64, error)
	CountByKYCStatus(ctx context.Context, status KYCStatus) (int64, error)
	ListWithFilters(ctx context.Context, filter UserListFilter) ([]*User, int64, error)
	FindRecent(ctx context.Context, limit int) ([]*User, error)

	// KYC document methods
	SaveKYCDocument(ctx context.Context, doc *KYCDocument) error
	GetKYCDocuments(ctx context.Context, userID uuid.UUID) ([]*KYCDocument, error)
	GetKYCDocumentByType(ctx context.Context, userID uuid.UUID, docType KYCDocumentType) (*KYCDocument, error)
	UpdateFaceMatchScore(ctx context.Context, userID uuid.UUID, score float32) error

	// Balance methods (amounts in UZS)
	// DeductBalance atomically subtracts amount if the balance is sufficient;
	// returns ErrInsufficientBalance otherwise.
	DeductBalance(ctx context.Context, userID uuid.UUID, amount int64) error
	// AddBalance atomically credits amount to the user's balance.
	AddBalance(ctx context.Context, userID uuid.UUID, amount int64) error
}

// ErrInsufficientBalance is returned when a balance deduction would go negative.
var ErrInsufficientBalance = errInsufficientBalance{}

type errInsufficientBalance struct{}

func (errInsufficientBalance) Error() string { return "insufficient balance" }
