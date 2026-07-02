package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// psql is the PostgreSQL placeholder format ($1, $2, ...).
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// UserRepo implements domain.UserRepository using PostgreSQL.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// userColumns lists all columns returned by SELECT queries (excluding the raw location column).
// Location is extracted separately via ST_Y/ST_X.
const userSelectColumns = `
	u.id, u.auth_provider, u.google_id, u.apple_id,
	u.email, u.password_hash, u.email_verified,
	u.display_name, u.phone_number, u.bio, u.avatar_url,
	u.social_media, u.language, u.status, u.status_reason,
	u.kyc_status, u.kyc_rejection_reason, u.kyc_verified_at, u.kyc_reviewed_by, u.face_match_score,
	ST_Y(u.location::geometry) AS latitude, ST_X(u.location::geometry) AS longitude, u.location_updated_at,
	u.posts_count, u.reports_received, u.reports_made,
	u.subscription_status, u.current_subscription_id, u.subscription_expires_at, u.grace_period_ends_at,
	u.last_login_at, u.last_active_at, u.created_at, u.updated_at
`

// scanUser scans a single pgx.Row (or pgx.Rows) into a domain.User.
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var socialMediaBytes []byte

	err := row.Scan(
		&u.ID, &u.AuthProvider, &u.GoogleID, &u.AppleID,
		&u.Email, &u.PasswordHash, &u.EmailVerified,
		&u.DisplayName, &u.PhoneNumber, &u.Bio, &u.AvatarURL,
		&socialMediaBytes, &u.Language, &u.Status, &u.StatusReason,
		&u.KYCStatus, &u.KYCRejectionReason, &u.KYCVerifiedAt, &u.KYCReviewedBy, &u.FaceMatchScore,
		&u.Latitude, &u.Longitude, &u.LocationUpdatedAt,
		&u.PostsCount, &u.ReportsReceived, &u.ReportsMade,
		&u.SubscriptionStatus, &u.CurrentSubscriptionID, &u.SubscriptionExpiresAt, &u.GracePeriodEndsAt,
		&u.LastLoginAt, &u.LastActiveAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if socialMediaBytes != nil {
		u.SocialMedia = json.RawMessage(socialMediaBytes)
	}

	return &u, nil
}

// Create inserts a new user into the database.
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	query := `
		INSERT INTO users (
			id, auth_provider, google_id, apple_id,
			email, password_hash, email_verified,
			display_name, phone_number, bio, avatar_url,
			social_media, language, status, status_reason,
			kyc_status, kyc_rejection_reason, kyc_verified_at, kyc_reviewed_by, face_match_score,
			location, location_updated_at,
			posts_count, reports_received, reports_made,
			subscription_status, current_subscription_id, subscription_expires_at, grace_period_ends_at,
			last_login_at, last_active_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20,
			CASE WHEN $21::float8 IS NOT NULL AND $22::float8 IS NOT NULL
				THEN ST_MakePoint($22, $21)::geography
				ELSE NULL
			END,
			$23,
			$24, $25, $26,
			$27, $28, $29, $30,
			$31, $32
		)
		RETURNING created_at, updated_at
	`

	var socialMedia []byte
	if user.SocialMedia != nil {
		socialMedia = []byte(user.SocialMedia)
	}

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.AuthProvider, user.GoogleID, user.AppleID,
		user.Email, user.PasswordHash, user.EmailVerified,
		user.DisplayName, user.PhoneNumber, user.Bio, user.AvatarURL,
		socialMedia, user.Language, user.Status, user.StatusReason,
		user.KYCStatus, user.KYCRejectionReason, user.KYCVerifiedAt, user.KYCReviewedBy, user.FaceMatchScore,
		user.Latitude, user.Longitude,
		user.LocationUpdatedAt,
		user.PostsCount, user.ReportsReceived, user.ReportsMade,
		user.SubscriptionStatus, user.CurrentSubscriptionID, user.SubscriptionExpiresAt, user.GracePeriodEndsAt,
		user.LastLoginAt, user.LastActiveAt,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	return err
}

// FindByID retrieves a user by their UUID.
func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users u WHERE u.id = $1`, userSelectColumns)
	row := r.pool.QueryRow(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// FindByGoogleID retrieves a user by their Google OAuth ID.
func (r *UserRepo) FindByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users u WHERE u.google_id = $1`, userSelectColumns)
	row := r.pool.QueryRow(ctx, query, googleID)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// FindByAppleID retrieves a user by their Apple OAuth ID.
func (r *UserRepo) FindByAppleID(ctx context.Context, appleID string) (*domain.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users u WHERE u.apple_id = $1`, userSelectColumns)
	row := r.pool.QueryRow(ctx, query, appleID)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// FindByEmail retrieves a user by their email address.
func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users u WHERE u.email = $1`, userSelectColumns)
	row := r.pool.QueryRow(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// Update updates all mutable fields of an existing user.
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	var socialMedia []byte
	if user.SocialMedia != nil {
		socialMedia = []byte(user.SocialMedia)
	}

	query := `
		UPDATE users SET
			auth_provider = $2,
			google_id = $3,
			apple_id = $4,
			email = $5,
			password_hash = $6,
			email_verified = $7,
			display_name = $8,
			phone_number = $9,
			bio = $10,
			avatar_url = $11,
			social_media = $12,
			language = $13,
			status = $14,
			status_reason = $15,
			kyc_status = $16,
			kyc_rejection_reason = $17,
			kyc_verified_at = $18,
			kyc_reviewed_by = $19,
			face_match_score = $20,
			location = CASE WHEN $21::float8 IS NOT NULL AND $22::float8 IS NOT NULL
				THEN ST_MakePoint($22, $21)::geography
				ELSE NULL
			END,
			location_updated_at = $23,
			posts_count = $24,
			reports_received = $25,
			reports_made = $26,
			subscription_status = $27,
			current_subscription_id = $28,
			subscription_expires_at = $29,
			grace_period_ends_at = $30,
			last_login_at = $31,
			last_active_at = $32
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		user.ID, user.AuthProvider, user.GoogleID, user.AppleID,
		user.Email, user.PasswordHash, user.EmailVerified,
		user.DisplayName, user.PhoneNumber, user.Bio, user.AvatarURL,
		socialMedia, user.Language, user.Status, user.StatusReason,
		user.KYCStatus, user.KYCRejectionReason, user.KYCVerifiedAt, user.KYCReviewedBy, user.FaceMatchScore,
		user.Latitude, user.Longitude,
		user.LocationUpdatedAt,
		user.PostsCount, user.ReportsReceived, user.ReportsMade,
		user.SubscriptionStatus, user.CurrentSubscriptionID, user.SubscriptionExpiresAt, user.GracePeriodEndsAt,
		user.LastLoginAt, user.LastActiveAt,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateKYCStatus updates the KYC verification status for a user.
// If the status is "verified", kyc_verified_at is set to NOW().
func (r *UserRepo) UpdateKYCStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, reviewedBy uuid.UUID, reason string) error {
	query := `
		UPDATE users SET
			kyc_status = $2,
			kyc_reviewed_by = $3,
			kyc_rejection_reason = $4,
			kyc_verified_at = CASE WHEN $2::kyc_status = 'verified' THEN NOW() ELSE kyc_verified_at END
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, userID, status, reviewedBy, reason)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateSubscriptionStatus updates the subscription fields for a user.
func (r *UserRepo) UpdateSubscriptionStatus(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error {
	query := `
		UPDATE users SET
			subscription_status = $2,
			current_subscription_id = $3,
			subscription_expires_at = $4
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, userID, status, subscriptionID, expiresAt)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateLocation updates the geographic location for a user.
// ST_MakePoint takes (longitude, latitude), so we pass $3=lng, $2=lat.
func (r *UserRepo) UpdateLocation(ctx context.Context, userID uuid.UUID, latitude, longitude float64) error {
	query := `
		UPDATE users SET
			location = ST_MakePoint($3, $2)::geography,
			location_updated_at = NOW()
		WHERE id = $1
	`

	cmdTag, err := r.pool.Exec(ctx, query, userID, latitude, longitude)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// FindNearby finds users within a given radius (in km) from a point.
// Only active users with verified KYC are returned, ordered by distance.
func (r *UserRepo) FindNearby(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*domain.User, error) {
	radiusMeters := radiusKM * 1000

	query := fmt.Sprintf(`
		SELECT %s
		FROM users u
		WHERE u.location IS NOT NULL
			AND u.status = 'active'
			AND u.kyc_status = 'verified'
			AND ST_DWithin(u.location, ST_MakePoint($1, $2)::geography, $3)
		ORDER BY ST_Distance(u.location, ST_MakePoint($1, $2)::geography) ASC
		LIMIT $4
	`, userSelectColumns)

	// ST_MakePoint(longitude, latitude)
	rows, err := r.pool.Query(ctx, query, longitude, latitude, radiusMeters, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// SearchByName performs a trigram similarity search on display_name.
// Returns matching users and the total count.
func (r *UserRepo) SearchByName(ctx context.Context, query string, limit, offset int) ([]*domain.User, int, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM users WHERE display_name % $1`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, query).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*domain.User{}, 0, nil
	}

	// Data query ordered by similarity
	dataQuery := fmt.Sprintf(`
		SELECT %s
		FROM users u
		WHERE u.display_name %% $1
		ORDER BY similarity(u.display_name, $1) DESC
		LIMIT $2 OFFSET $3
	`, userSelectColumns)

	rows, err := r.pool.Query(ctx, dataQuery, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CountByStatus returns the count of users with the given status.
func (r *UserRepo) CountByStatus(ctx context.Context, status domain.UserStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE status = $1`

	var count int64
	err := r.pool.QueryRow(ctx, query, status).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ListWithFilters returns paginated users with optional dynamic filters.
// Uses squirrel for dynamic query construction.
func (r *UserRepo) ListWithFilters(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	// Build the WHERE clause using squirrel
	where := sq.And{}

	if filter.Search != "" {
		searchPattern := "%" + strings.ToLower(filter.Search) + "%"
		where = append(where, sq.Or{
			sq.Expr("LOWER(display_name) LIKE ?", searchPattern),
			sq.Expr("LOWER(email) LIKE ?", searchPattern),
		})
	}

	if filter.Status != nil {
		where = append(where, sq.Eq{"status": *filter.Status})
	}

	if filter.KYCStatus != nil {
		where = append(where, sq.Eq{"kyc_status": *filter.KYCStatus})
	}

	if filter.SubscriptionStatus != nil {
		where = append(where, sq.Eq{"subscription_status": *filter.SubscriptionStatus})
	}

	// ---- Count query ----
	countBuilder := psql.Select("COUNT(*)").From("users")
	if len(where) > 0 {
		countBuilder = countBuilder.Where(where)
	}

	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building count query: %w", err)
	}

	var total int64
	err = r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*domain.User{}, 0, nil
	}

	// ---- Data query ----
	// We use a raw select with our custom columns (including PostGIS expressions)
	// but still use squirrel for WHERE, ORDER, LIMIT, OFFSET.
	dataBuilder := psql.
		Select(userSelectColumns).
		From("users u")

	if len(where) > 0 {
		dataBuilder = dataBuilder.Where(where)
	}

	// Sort
	sortColumn := "u.created_at"
	switch filter.SortBy {
	case "display_name":
		sortColumn = "u.display_name"
	case "email":
		sortColumn = "u.email"
	case "created_at":
		sortColumn = "u.created_at"
	}

	sortOrder := "DESC"
	if filter.SortOrder == domain.SortOrderASC {
		sortOrder = "ASC"
	}
	dataBuilder = dataBuilder.OrderBy(sortColumn + " " + sortOrder)

	// Pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	dataBuilder = dataBuilder.Limit(uint64(filter.Limit)).Offset(uint64(offset))

	dataSQL, dataArgs, err := dataBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building data query: %w", err)
	}

	rows, err := r.pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// SaveKYCDocument inserts or updates (upserts) a KYC document for a user.
// The kyc_documents table has a UNIQUE(user_id, doc_type) constraint,
// so we use ON CONFLICT to replace the file if the same doc type is uploaded again.
func (r *UserRepo) SaveKYCDocument(ctx context.Context, doc *domain.KYCDocument) error {
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}

	query := `
		INSERT INTO kyc_documents (id, user_id, doc_type, file_path, uploaded_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id, doc_type) DO UPDATE SET
			file_path = EXCLUDED.file_path,
			uploaded_at = NOW(),
			verified_at = NULL
		RETURNING uploaded_at
	`

	err := r.pool.QueryRow(ctx, query,
		doc.ID, doc.UserID, doc.DocType, doc.FilePath,
	).Scan(&doc.UploadedAt)

	return err
}

// GetKYCDocuments returns all KYC documents for a given user.
func (r *UserRepo) GetKYCDocuments(ctx context.Context, userID uuid.UUID) ([]*domain.KYCDocument, error) {
	query := `
		SELECT id, user_id, doc_type, file_path, uploaded_at, verified_at
		FROM kyc_documents
		WHERE user_id = $1
		ORDER BY uploaded_at DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*domain.KYCDocument
	for rows.Next() {
		var doc domain.KYCDocument
		if err := rows.Scan(&doc.ID, &doc.UserID, &doc.DocType, &doc.FilePath, &doc.UploadedAt, &doc.VerifiedAt); err != nil {
			return nil, err
		}
		docs = append(docs, &doc)
	}
	return docs, rows.Err()
}

// GetKYCDocumentByType returns a specific KYC document by user ID and document type.
func (r *UserRepo) GetKYCDocumentByType(ctx context.Context, userID uuid.UUID, docType domain.KYCDocumentType) (*domain.KYCDocument, error) {
	query := `
		SELECT id, user_id, doc_type, file_path, uploaded_at, verified_at
		FROM kyc_documents
		WHERE user_id = $1 AND doc_type = $2
	`

	var doc domain.KYCDocument
	err := r.pool.QueryRow(ctx, query, userID, docType).Scan(
		&doc.ID, &doc.UserID, &doc.DocType, &doc.FilePath, &doc.UploadedAt, &doc.VerifiedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("kyc document not found")
		}
		return nil, err
	}
	return &doc, nil
}

// UpdateFaceMatchScore updates only the face_match_score field for a user.
func (r *UserRepo) UpdateFaceMatchScore(ctx context.Context, userID uuid.UUID, score float32) error {
	query := `UPDATE users SET face_match_score = $2 WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, userID, score)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// CountByKYCStatus returns the number of users with a given KYC status.
func (r *UserRepo) CountByKYCStatus(ctx context.Context, status domain.KYCStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM users WHERE kyc_status = $1`
	var count int64
	err := r.pool.QueryRow(ctx, query, status).Scan(&count)
	return count, err
}

// FindRecent returns the most recently created users.
func (r *UserRepo) FindRecent(ctx context.Context, limit int) ([]*domain.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users u ORDER BY u.created_at DESC LIMIT $1`, userSelectColumns)
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
