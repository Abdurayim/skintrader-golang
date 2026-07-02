package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"skintrader-go/internal/domain"
)

// GameRepo implements domain.GameRepository using PostgreSQL.
type GameRepo struct {
	pool *pgxpool.Pool
}

// NewGameRepo creates a new GameRepo.
func NewGameRepo(pool *pgxpool.Pool) *GameRepo {
	return &GameRepo{pool: pool}
}

// slugify converts a name into a URL-safe slug.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	// Replace non-alphanumeric characters (except hyphens) with hyphens.
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// scanGame scans a single game row into a domain.Game struct.
func scanGame(row pgx.Row) (*domain.Game, error) {
	var g domain.Game
	err := row.Scan(
		&g.ID,
		&g.Name,
		&g.Slug,
		&g.Icon,
		&g.Genres,
		&g.PostsCount,
		&g.IsActive,
		&g.CreatedBy,
		&g.UpdatedBy,
		&g.CreatedAt,
		&g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if g.Genres == nil {
		g.Genres = []string{}
	}
	return &g, nil
}

// scanGames scans multiple game rows into a slice of domain.Game pointers.
func scanGames(rows pgx.Rows) ([]*domain.Game, error) {
	defer rows.Close()
	var games []*domain.Game
	for rows.Next() {
		var g domain.Game
		err := rows.Scan(
			&g.ID,
			&g.Name,
			&g.Slug,
			&g.Icon,
			&g.Genres,
			&g.PostsCount,
			&g.IsActive,
			&g.CreatedBy,
			&g.UpdatedBy,
			&g.CreatedAt,
			&g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning game row: %w", err)
		}
		if g.Genres == nil {
			g.Genres = []string{}
		}
		games = append(games, &g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating game rows: %w", err)
	}
	if games == nil {
		games = []*domain.Game{}
	}
	return games, nil
}

const gameColumns = `id, name, slug, icon, genres, posts_count, is_active, created_by, updated_by, created_at, updated_at`

func (r *GameRepo) Create(ctx context.Context, game *domain.Game) error {
	if game.ID == uuid.Nil {
		game.ID = uuid.New()
	}
	if game.Slug == "" {
		game.Slug = slugify(game.Name)
	}

	query := `
		INSERT INTO games (id, name, slug, icon, genres, posts_count, is_active, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		game.ID,
		game.Name,
		game.Slug,
		game.Icon,
		game.Genres,
		game.PostsCount,
		game.IsActive,
		game.CreatedBy,
		game.UpdatedBy,
	).Scan(&game.CreatedAt, &game.UpdatedAt)
}

func (r *GameRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Game, error) {
	query := `SELECT ` + gameColumns + ` FROM games WHERE id = $1`
	game, err := scanGame(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("finding game by id: %w", err)
	}
	return game, nil
}

func (r *GameRepo) FindBySlug(ctx context.Context, slug string) (*domain.Game, error) {
	query := `SELECT ` + gameColumns + ` FROM games WHERE slug = $1`
	game, err := scanGame(r.pool.QueryRow(ctx, query, slug))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("game not found")
		}
		return nil, fmt.Errorf("finding game by slug: %w", err)
	}
	return game, nil
}

func (r *GameRepo) FindActive(ctx context.Context) ([]*domain.Game, error) {
	query := `SELECT ` + gameColumns + ` FROM games WHERE is_active = true ORDER BY name`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying active games: %w", err)
	}
	return scanGames(rows)
}

func (r *GameRepo) SearchByName(ctx context.Context, query string, limit, offset int) ([]*domain.Game, int, error) {
	// Count total matching results.
	countQuery := `SELECT COUNT(*) FROM games WHERE name ILIKE '%' || $1 || '%' AND is_active = true`
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, query).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting search results: %w", err)
	}

	// Fetch paginated results ordered by trigram similarity.
	searchQuery := `
		SELECT ` + gameColumns + `
		FROM games
		WHERE name ILIKE '%' || $1 || '%' AND is_active = true
		ORDER BY similarity(name, $1) DESC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, searchQuery, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("searching games by name: %w", err)
	}
	games, err := scanGames(rows)
	if err != nil {
		return nil, 0, err
	}
	return games, total, nil
}

func (r *GameRepo) GetPopular(ctx context.Context, limit int) ([]*domain.Game, error) {
	query := `
		SELECT ` + gameColumns + `
		FROM games
		WHERE is_active = true
		ORDER BY posts_count DESC
		LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("querying popular games: %w", err)
	}
	return scanGames(rows)
}

func (r *GameRepo) FindByGenre(ctx context.Context, genre string) ([]*domain.Game, error) {
	query := `
		SELECT ` + gameColumns + `
		FROM games
		WHERE $1 = ANY(genres) AND is_active = true
		ORDER BY name`
	rows, err := r.pool.Query(ctx, query, genre)
	if err != nil {
		return nil, fmt.Errorf("querying games by genre: %w", err)
	}
	return scanGames(rows)
}

func (r *GameRepo) Update(ctx context.Context, game *domain.Game) error {
	query := `
		UPDATE games
		SET name = $2, slug = $3, icon = $4, genres = $5,
		    is_active = $6, updated_by = $7
		WHERE id = $1
		RETURNING updated_at`

	if game.Slug == "" {
		game.Slug = slugify(game.Name)
	}

	err := r.pool.QueryRow(ctx, query,
		game.ID,
		game.Name,
		game.Slug,
		game.Icon,
		game.Genres,
		game.IsActive,
		game.UpdatedBy,
	).Scan(&game.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("game not found")
		}
		return fmt.Errorf("updating game: %w", err)
	}
	return nil
}

func (r *GameRepo) IncrementPostsCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE games SET posts_count = posts_count + 1 WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("incrementing posts count: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("game not found")
	}
	return nil
}

func (r *GameRepo) DecrementPostsCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE games SET posts_count = GREATEST(posts_count - 1, 0) WHERE id = $1`
	ct, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("decrementing posts count: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("game not found")
	}
	return nil
}
