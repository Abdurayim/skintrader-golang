package domain

// UserStatus represents the status of a user account.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"
)

// KYCStatus represents the KYC verification status.
type KYCStatus string

const (
	KYCStatusNotSubmitted KYCStatus = "not_submitted"
	KYCStatusPending      KYCStatus = "pending"
	KYCStatusVerified     KYCStatus = "verified"
	KYCStatusRejected     KYCStatus = "rejected"
)

// KYCDocumentType represents the type of KYC document.
type KYCDocumentType string

const (
	KYCDocumentTypeIDCard   KYCDocumentType = "id_card"
	KYCDocumentTypePassport KYCDocumentType = "passport"
	KYCDocumentTypeSelfie   KYCDocumentType = "selfie"
)

// AuthProvider represents the authentication provider.
type AuthProvider string

const (
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderApple  AuthProvider = "apple"
	AuthProviderEmail  AuthProvider = "email"
)

// PostStatus represents the status of a post.
type PostStatus string

const (
	PostStatusActive PostStatus = "active"
	PostStatusSold   PostStatus = "sold"
	PostStatusDraft  PostStatus = "draft"
)

// PostType represents the type of post.
type PostType string

const (
	PostTypeSkin    PostType = "skin"
	PostTypeProfile PostType = "profile"
)

// Currency represents a currency code.
type Currency string

const (
	CurrencyUZS Currency = "UZS"
	CurrencyUSD Currency = "USD"
)

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	SubscriptionStatusNone        SubscriptionStatus = "none"
	SubscriptionStatusActive      SubscriptionStatus = "active"
	SubscriptionStatusExpired     SubscriptionStatus = "expired"
	SubscriptionStatusGracePeriod SubscriptionStatus = "grace_period"
	SubscriptionStatusPending     SubscriptionStatus = "pending"
	SubscriptionStatusCancelled   SubscriptionStatus = "cancelled"
)

// SubscriptionPlan represents the subscription plan type.
type SubscriptionPlan string

const (
	SubscriptionPlanMonthly SubscriptionPlan = "monthly"
)

// TransactionStatus represents the status of a payment transaction.
type TransactionStatus string

const (
	TransactionStatusPending    TransactionStatus = "pending"
	TransactionStatusProcessing TransactionStatus = "processing"
	TransactionStatusCompleted  TransactionStatus = "completed"
	TransactionStatusFailed     TransactionStatus = "failed"
	TransactionStatusCancelled  TransactionStatus = "cancelled"
	TransactionStatusRefunded   TransactionStatus = "refunded"
)

// PaymentMethod represents a supported payment method.
type PaymentMethod string

const (
	PaymentMethodPayme PaymentMethod = "payme"
	PaymentMethodClick PaymentMethod = "click"
	PaymentMethodXazna PaymentMethod = "xazna"
	PaymentMethodUzum  PaymentMethod = "uzum"
)

// MessageStatus represents the delivery status of a message.
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

// ReportType represents what kind of entity is being reported.
type ReportType string

const (
	ReportTypePost ReportType = "post"
	ReportTypeUser ReportType = "user"
)

// ReportCategory represents the reason for a report.
type ReportCategory string

const (
	// Post report categories
	ReportCategoryScam                 ReportCategory = "scam"
	ReportCategoryFakeItem             ReportCategory = "fake_item"
	ReportCategoryInappropriateContent ReportCategory = "inappropriate_content"
	ReportCategoryDuplicatePost        ReportCategory = "duplicate_post"
	ReportCategoryIncorrectPricing     ReportCategory = "incorrect_pricing"

	// User report categories
	ReportCategoryHarassment       ReportCategory = "harassment"
	ReportCategorySpam             ReportCategory = "spam"
	ReportCategoryFraud            ReportCategory = "fraud"
	ReportCategoryImpersonation    ReportCategory = "impersonation"
	ReportCategoryOffensiveProfile ReportCategory = "offensive_profile"

	// General
	ReportCategoryOther ReportCategory = "other"
)

// ReportStatus represents the review status of a report.
type ReportStatus string

const (
	ReportStatusPending     ReportStatus = "pending"
	ReportStatusUnderReview ReportStatus = "under_review"
	ReportStatusResolved    ReportStatus = "resolved"
	ReportStatusDismissed   ReportStatus = "dismissed"
)

// ReportPriority represents the priority level of a report.
type ReportPriority string

const (
	ReportPriorityLow      ReportPriority = "low"
	ReportPriorityMedium   ReportPriority = "medium"
	ReportPriorityHigh     ReportPriority = "high"
	ReportPriorityCritical ReportPriority = "critical"
)

// ReportAction represents the action taken to resolve a report.
type ReportAction string

const (
	ReportActionDismiss     ReportAction = "dismiss"
	ReportActionDeletePost  ReportAction = "delete_post"
	ReportActionWarnUser    ReportAction = "warn_user"
	ReportActionSuspendUser ReportAction = "suspend_user"
	ReportActionBanUser     ReportAction = "ban_user"
	ReportActionDeleteUser  ReportAction = "delete_user"
)

// AdminRole represents the role of an admin user.
type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "superadmin"
	AdminRoleModerator  AdminRole = "moderator"
	AdminRoleSupport    AdminRole = "support"
)

// AdminPermission represents a specific admin permission.
type AdminPermission string

const (
	AdminPermissionManageUsers         AdminPermission = "manage_users"
	AdminPermissionManagePosts         AdminPermission = "manage_posts"
	AdminPermissionManageAdmins        AdminPermission = "manage_admins"
	AdminPermissionViewKYC             AdminPermission = "view_kyc"
	AdminPermissionApproveKYC          AdminPermission = "approve_kyc"
	AdminPermissionViewLogs            AdminPermission = "view_logs"
	AdminPermissionViewStats           AdminPermission = "view_stats"
	AdminPermissionManageGames         AdminPermission = "manage_games"
	AdminPermissionManageSubscriptions AdminPermission = "manage_subscriptions"
	AdminPermissionManageReports       AdminPermission = "manage_reports"
)

// RolePermissions maps each admin role to its allowed permissions.
var RolePermissions = map[AdminRole][]AdminPermission{
	AdminRoleSuperAdmin: {
		AdminPermissionManageUsers,
		AdminPermissionManagePosts,
		AdminPermissionManageAdmins,
		AdminPermissionViewKYC,
		AdminPermissionApproveKYC,
		AdminPermissionViewLogs,
		AdminPermissionViewStats,
		AdminPermissionManageGames,
		AdminPermissionManageSubscriptions,
		AdminPermissionManageReports,
	},
	AdminRoleModerator: {
		AdminPermissionManageUsers,
		AdminPermissionManagePosts,
		AdminPermissionViewKYC,
		AdminPermissionApproveKYC,
		AdminPermissionViewLogs,
		AdminPermissionViewStats,
		AdminPermissionManageReports,
	},
	AdminRoleSupport: {
		AdminPermissionViewKYC,
		AdminPermissionViewStats,
	},
}

// Language represents a supported language code.
type Language string

const (
	LanguageEN Language = "en"
	LanguageRU Language = "ru"
	LanguageUZ Language = "uz"
)

// AdminAction represents the type of action logged in the admin audit trail.
type AdminAction string

const (
	// User actions
	AdminActionUserBanned    AdminAction = "user_banned"
	AdminActionUserUnbanned  AdminAction = "user_unbanned"
	AdminActionUserSuspended AdminAction = "user_suspended"
	AdminActionUserDeleted   AdminAction = "user_deleted"
	AdminActionUserWarned    AdminAction = "user_warned"

	// KYC actions
	AdminActionKYCApproved AdminAction = "kyc_approved"
	AdminActionKYCRejected AdminAction = "kyc_rejected"

	// Post actions
	AdminActionPostDeleted AdminAction = "post_deleted"
	AdminActionPostFlagged AdminAction = "post_flagged"

	// Subscription actions
	AdminActionSubscriptionGranted AdminAction = "subscription_granted"
	AdminActionSubscriptionRevoked AdminAction = "subscription_revoked"

	// Balance top-up actions
	AdminActionTopupApproved AdminAction = "topup_approved"
	AdminActionTopupRejected AdminAction = "topup_rejected"

	// Report actions
	AdminActionReportResolved  AdminAction = "report_resolved"
	AdminActionReportDismissed AdminAction = "report_dismissed"

	// Admin actions
	AdminActionAdminCreated AdminAction = "admin_created"
	AdminActionAdminUpdated AdminAction = "admin_updated"
	AdminActionAdminDeleted AdminAction = "admin_deleted"

	// Game actions
	AdminActionGameCreated AdminAction = "game_created"
	AdminActionGameUpdated AdminAction = "game_updated"
	AdminActionGameDeleted AdminAction = "game_deleted"

	// Auth actions
	AdminActionAdminLogin  AdminAction = "admin_login"
	AdminActionAdminLogout AdminAction = "admin_logout"
)

// GameGenres contains all supported game genres.
var GameGenres = []string{
	"FPS",
	"MOBA",
	"RPG",
	"Battle Royale",
	"Sports",
	"Racing",
	"Strategy",
	"MMO",
	"Fighting",
	"Survival",
	"Simulation",
	"Action",
	"Adventure",
	"Card",
	"Horror",
	"Sandbox",
	"Mobile",
	"Football",
	"Basketball",
	"Life",
	"Driving",
	"Social",
	"Party",
	"Co-op",
	"Turn-Based",
	"Other",
}

// SocialPlatforms contains all supported social media platforms.
var SocialPlatforms = []string{
	"telegram",
	"instagram",
	"facebook",
	"twitter",
	"discord",
	"vk",
	"tiktok",
	"youtube",
	"other",
}

// SortOrder represents the direction of sorting.
type SortOrder string

const (
	SortOrderASC  SortOrder = "ASC"
	SortOrderDESC SortOrder = "DESC"
)
