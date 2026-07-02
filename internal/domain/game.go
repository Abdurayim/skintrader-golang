package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Game represents a game that users can list skins/profiles for.
type Game struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	Slug       string     `json:"slug" db:"slug"`
	Icon       string     `json:"icon,omitempty" db:"icon"`
	Genres     []string   `json:"genres" db:"genres"`
	PostsCount int        `json:"postsCount" db:"posts_count"`
	IsActive   bool       `json:"isActive" db:"is_active"`
	CreatedBy  *uuid.UUID `json:"createdBy,omitempty" db:"created_by"`
	UpdatedBy  *uuid.UUID `json:"updatedBy,omitempty" db:"updated_by"`
	CreatedAt  time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time  `json:"updatedAt" db:"updated_at"`
}

// GameRepository defines the interface for game data access.
type GameRepository interface {
	Create(ctx context.Context, game *Game) error
	FindByID(ctx context.Context, id uuid.UUID) (*Game, error)
	FindBySlug(ctx context.Context, slug string) (*Game, error)
	FindActive(ctx context.Context) ([]*Game, error)
	SearchByName(ctx context.Context, query string, limit, offset int) ([]*Game, int, error)
	GetPopular(ctx context.Context, limit int) ([]*Game, error)
	FindByGenre(ctx context.Context, genre string) ([]*Game, error)
	Update(ctx context.Context, game *Game) error
	IncrementPostsCount(ctx context.Context, id uuid.UUID) error
	DecrementPostsCount(ctx context.Context, id uuid.UUID) error
}
