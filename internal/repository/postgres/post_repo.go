package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// psq is a PostgreSQL-flavored squirrel statement builder using $1, $2, ... placeholders.
var psq = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// PostRepo implements domain.PostRepository using PostgreSQL.
type PostRepo struct {
	pool *pgxpool.Pool
}

// NewPostRepo creates a new PostRepo.
func NewPostRepo(pool *pgxpool.Pool) *PostRepo {
	return &PostRepo{pool: pool}
}

const postColumns = `id, user_id, title, description, price, currency, game_id, genre, type,
	contact_info, status, views_count, reports_count, reported_at,
	deleted_at, deleted_by, deleted_by_type, created_at, updated_at`

// scanPost scans a single post row into a domain.Post struct.
func scanPost(row pgx.Row) (*domain.Post, error) {
	var p domain.Post
	err := row.Scan(
		&p.ID,
		&p.UserID,
		&p.Title,
		&p.Description,
		&p.Price,
		&p.Currency,
		&p.GameID,
		&p.Genre,
		&p.Type,
		&p.ContactInfo,
		&p.Status,
		&p.ViewsCount,
		&p.ReportsCount,
		&p.ReportedAt,
		&p.DeletedAt,
		&p.DeletedBy,
		&p.DeletedByType,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// scanPosts scans multiple post rows into a slice of domain.Post pointers.
func scanPosts(rows pgx.Rows) ([]*domain.Post, error) {
	defer rows.Close()
	var posts []*domain.Post
	for rows.Next() {
		var p domain.Post
		err := rows.Scan(
			&p.ID,
			&p.UserID,
			&p.Title,
			&p.Description,
			&p.Price,
			&p.Currency,
			&p.GameID,
			&p.Genre,
			&p.Type,
			&p.ContactInfo,
			&p.Status,
			&p.ViewsCount,
			&p.ReportsCount,
			&p.ReportedAt,
			&p.DeletedAt,
			&p.DeletedBy,
			&p.DeletedByType,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning post row: %w", err)
		}
		posts = append(posts, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating post rows: %w", err)
	}
	if posts == nil {
		posts = []*domain.Post{}
	}
	return posts, nil
}

// scanPostImage scans a single post image row.
func scanPostImage(row pgx.Row) (*domain.PostImage, error) {
	var img domain.PostImage
	err := row.Scan(
		&img.ID,
		&img.PostID,
		&img.OriginalPath,
		&img.ThumbnailPath,
		&img.Filename,
		&img.Size,
		&img.MimeType,
		&img.SortOrder,
		&img.UploadedAt,
	)
	if err != nil {
		return nil, err
	}
	return &img, nil
}

// loadPostImages fetches all images for a given post ID.
func (r *PostRepo) loadPostImages(ctx context.Context, postID uuid.UUID) ([]*domain.PostImage, error) {
	query := `
		SELECT id, post_id, original_path, thumbnail_path, filename, size, mime_type, sort_order, uploaded_at
		FROM post_images
		WHERE post_id = $1
		ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, query, postID)
	if err != nil {
		return nil, fmt.Errorf("querying post images: %w", err)
	}
	defer rows.Close()

	var images []*domain.PostImage
	for rows.Next() {
		var img domain.PostImage
		err := rows.Scan(
			&img.ID,
			&img.PostID,
			&img.OriginalPath,
			&img.ThumbnailPath,
			&img.Filename,
			&img.Size,
			&img.MimeType,
			&img.SortOrder,
			&img.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning post image row: %w", err)
		}
		images = append(images, &img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating post image rows: %w", err)
	}
	if images == nil {
		images = []*domain.PostImage{}
	}
	return images, nil
}

// loadImagesForPosts batch-loads images for a set of posts in one query
// and attaches them to each post.
func (r *PostRepo) loadImagesForPosts(ctx context.Context, posts []*domain.Post) error {
	if len(posts) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(posts))
	byID := make(map[uuid.UUID]*domain.Post, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
		byID[p.ID] = p
		if p.Images == nil {
			p.Images = []*domain.PostImage{}
		}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, post_id, original_path, thumbnail_path, filename, size, mime_type, sort_order, uploaded_at
		FROM post_images
		WHERE post_id = ANY($1)
		ORDER BY post_id, sort_order`, ids)
	if err != nil {
		return fmt.Errorf("querying images for posts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		img, err := scanPostImage(rows)
		if err != nil {
			return fmt.Errorf("scanning post image row: %w", err)
		}
		if p, ok := byID[img.PostID]; ok {
			p.Images = append(p.Images, img)
		}
	}
	return rows.Err()
}

func (r *PostRepo) Create(ctx context.Context, post *domain.Post) error {
	if post.ID == uuid.Nil {
		post.ID = uuid.New()
	}

	query := `
		INSERT INTO posts (
			id, user_id, title, description, price, currency, game_id, genre, type,
			contact_info, status, views_count, reports_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		post.ID,
		post.UserID,
		post.Title,
		post.Description,
		post.Price,
		post.Currency,
		post.GameID,
		post.Genre,
		post.Type,
		post.ContactInfo,
		post.Status,
		post.ViewsCount,
		post.ReportsCount,
	).Scan(&post.CreatedAt, &post.UpdatedAt)
}

func (r *PostRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	query := `SELECT ` + postColumns + ` FROM posts WHERE id = $1`
	post, err := scanPost(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("post not found")
		}
		return nil, fmt.Errorf("finding post by id: %w", err)
	}

	// Load associated images.
	images, err := r.loadPostImages(ctx, post.ID)
	if err != nil {
		return nil, fmt.Errorf("loading post images: %w", err)
	}
	post.Images = images
	return post, nil
}

func (r *PostRepo) FindActive(ctx context.Context, limit, offset int) ([]*domain.Post, int64, error) {
	// Count total active posts.
	countQuery := `SELECT COUNT(*) FROM posts WHERE status = 'active' AND deleted_at IS NULL`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting active posts: %w", err)
	}

	query := `
		SELECT ` + postColumns + `
		FROM posts
		WHERE status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying active posts: %w", err)
	}
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadImagesForPosts(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *PostRepo) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Post, int64, error) {
	// Count total posts for the user.
	countQuery := `SELECT COUNT(*) FROM posts WHERE user_id = $1 AND deleted_at IS NULL`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting user posts: %w", err)
	}

	query := `
		SELECT ` + postColumns + `
		FROM posts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying user posts: %w", err)
	}
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadImagesForPosts(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *PostRepo) FindByUserWithStatus(ctx context.Context, userID uuid.UUID, status *domain.PostStatus, limit, offset int) ([]*domain.Post, int64, error) {
	// Build query with optional status filter
	whereClause := "WHERE user_id = $1 AND deleted_at IS NULL"
	args := []any{userID}
	argIdx := 2

	if status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*status))
		argIdx++
	}

	// Count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM posts %s", whereClause)
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting user posts with status: %w", err)
	}

	// Data
	dataQuery := fmt.Sprintf("SELECT %s FROM posts %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		postColumns, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying user posts with status: %w", err)
	}
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadImagesForPosts(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *PostRepo) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Post, int64, error) {
	// Count total matching results using full-text search.
	countSQL := `
		SELECT COUNT(*)
		FROM posts
		WHERE to_tsvector('simple', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $1)
		  AND status = 'active' AND deleted_at IS NULL`
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting search results: %w", err)
	}

	searchSQL := `
		SELECT ` + postColumns + `
		FROM posts
		WHERE to_tsvector('simple', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', $1)
		  AND status = 'active' AND deleted_at IS NULL
		ORDER BY ts_rank(to_tsvector('simple', title || ' ' || COALESCE(description, '')), plainto_tsquery('simple', $1)) DESC,
		         created_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, searchSQL, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("searching posts: %w", err)
	}
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadImagesForPosts(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

func (r *PostRepo) Update(ctx context.Context, post *domain.Post) error {
	query := `
		UPDATE posts
		SET title = $2, description = $3, price = $4, currency = $5, game_id = $6,
		    genre = $7, type = $8, contact_info = $9, status = $10
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		post.ID,
		post.Title,
		post.Description,
		post.Price,
		post.Currency,
		post.GameID,
		post.Genre,
		post.Type,
		post.ContactInfo,
		post.Status,
	).Scan(&post.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("post not found")
		}
		return fmt.Errorf("updating post: %w", err)
	}
	return nil
}

func (r *PostRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID, deletedByType string) error {
	query := `
		UPDATE posts
		SET deleted_at = NOW(), deleted_by = $2, deleted_by_type = $3
		WHERE id = $1 AND deleted_at IS NULL`
	ct, err := r.pool.Exec(ctx, query, id, deletedBy, deletedByType)
	if err != nil {
		return fmt.Errorf("soft deleting post: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("post not found or already deleted")
	}
	return nil
}

func (r *PostRepo) IncrementViews(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE posts SET views_count = views_count + 1 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("incrementing views: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("post not found")
	}
	return nil
}

func (r *PostRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PostStatus) error {
	query := `UPDATE posts SET status = $2 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("updating post status: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("post not found")
	}
	return nil
}

func (r *PostRepo) AddImage(ctx context.Context, image *domain.PostImage) error {
	if image.ID == uuid.Nil {
		image.ID = uuid.New()
	}

	query := `
		INSERT INTO post_images (id, post_id, original_path, thumbnail_path, filename, size, mime_type, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING uploaded_at`

	return r.pool.QueryRow(ctx, query,
		image.ID,
		image.PostID,
		image.OriginalPath,
		image.ThumbnailPath,
		image.Filename,
		image.Size,
		image.MimeType,
		image.SortOrder,
	).Scan(&image.UploadedAt)
}

func (r *PostRepo) RemoveImage(ctx context.Context, imageID uuid.UUID) error {
	query := `DELETE FROM post_images WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, imageID)
	if err != nil {
		return fmt.Errorf("removing post image: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("post image not found")
	}
	return nil
}

func (r *PostRepo) ListWithFilters(ctx context.Context, filter domain.PostListFilter) ([]*domain.Post, int64, error) {
	// Build the base WHERE conditions using squirrel.
	baseWhere := psq.Select("").From("posts")

	// Apply filters.
	if filter.Search != "" {
		baseWhere = baseWhere.Where(
			"to_tsvector('simple', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', ?)",
			filter.Search,
		)
	}
	if filter.Status != nil {
		baseWhere = baseWhere.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.Type != nil {
		baseWhere = baseWhere.Where(sq.Eq{"type": *filter.Type})
	}
	if filter.GameID != nil {
		baseWhere = baseWhere.Where(sq.Eq{"game_id": *filter.GameID})
	}
	if filter.UserID != nil {
		baseWhere = baseWhere.Where(sq.Eq{"user_id": *filter.UserID})
	}
	if filter.Currency != nil {
		baseWhere = baseWhere.Where(sq.Eq{"currency": *filter.Currency})
	}
	if filter.MinPrice != nil {
		baseWhere = baseWhere.Where(sq.GtOrEq{"price": *filter.MinPrice})
	}
	if filter.MaxPrice != nil {
		baseWhere = baseWhere.Where(sq.LtOrEq{"price": *filter.MaxPrice})
	}

	// Build the count query.
	countBuilder := psq.Select("COUNT(*)").From("posts")
	// Copy WHERE conditions from baseWhere by rebuilding.
	// We rebuild from scratch to get a clean count query.
	countBuilder = r.applyPostFilters(countBuilder, filter)
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building count query: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting filtered posts: %w", err)
	}

	// Build the data query.
	dataBuilder := psq.Select(
		"id", "user_id", "title", "description", "price", "currency",
		"game_id", "genre", "type", "contact_info", "status",
		"views_count", "reports_count", "reported_at",
		"deleted_at", "deleted_by", "deleted_by_type",
		"created_at", "updated_at",
	).From("posts")
	dataBuilder = r.applyPostFilters(dataBuilder, filter)

	// Sorting.
	sortBy := "created_at"
	if filter.SortBy != "" {
		// Allow only known columns to prevent SQL injection.
		switch filter.SortBy {
		case "created_at", "updated_at", "price", "views_count", "reports_count", "title":
			sortBy = filter.SortBy
		}
	}
	sortOrder := "DESC"
	if filter.SortOrder == domain.SortOrderASC {
		sortOrder = "ASC"
	}
	dataBuilder = dataBuilder.OrderBy(sortBy + " " + sortOrder)

	// Pagination.
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	dataBuilder = dataBuilder.Limit(uint64(limit)).Offset(uint64(offset))

	dataSQL, dataArgs, err := dataBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("building data query: %w", err)
	}

	rows, err := r.pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying filtered posts: %w", err)
	}
	posts, err := scanPosts(rows)
	if err != nil {
		return nil, 0, err
	}
	if err := r.loadImagesForPosts(ctx, posts); err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// applyPostFilters applies common PostListFilter conditions to a squirrel SelectBuilder.
func (r *PostRepo) applyPostFilters(builder sq.SelectBuilder, filter domain.PostListFilter) sq.SelectBuilder {
	if !filter.IncludeDeleted {
		builder = builder.Where("deleted_at IS NULL")
	}
	if filter.Search != "" {
		builder = builder.Where(
			"to_tsvector('simple', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('simple', ?)",
			filter.Search,
		)
	}
	if filter.Status != nil {
		builder = builder.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.Type != nil {
		builder = builder.Where(sq.Eq{"type": *filter.Type})
	}
	if filter.GameID != nil {
		builder = builder.Where(sq.Eq{"game_id": *filter.GameID})
	}
	if filter.UserID != nil {
		builder = builder.Where(sq.Eq{"user_id": *filter.UserID})
	}
	if filter.Currency != nil {
		builder = builder.Where(sq.Eq{"currency": *filter.Currency})
	}
	if filter.MinPrice != nil {
		builder = builder.Where(sq.GtOrEq{"price": *filter.MinPrice})
	}
	if filter.MaxPrice != nil {
		builder = builder.Where(sq.LtOrEq{"price": *filter.MaxPrice})
	}
	return builder
}

// CountByStatus returns the number of posts with a given status (excluding soft-deleted).
func (r *PostRepo) CountByStatus(ctx context.Context, status domain.PostStatus) (int64, error) {
	query := `SELECT COUNT(*) FROM posts WHERE status = $1 AND deleted_at IS NULL`
	var count int64
	err := r.pool.QueryRow(ctx, query, status).Scan(&count)
	return count, err
}

// CountAll returns the total number of posts (excluding soft-deleted).
func (r *PostRepo) CountAll(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// FindRecent returns the most recently created posts.
func (r *PostRepo) FindRecent(ctx context.Context, limit int) ([]*domain.Post, error) {
	query := fmt.Sprintf(`SELECT %s FROM posts WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT $1`, postColumns)
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*domain.Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}
