package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockGameRepository implements domain.GameRepository for testing.
type MockGameRepository struct {
	CreateFn              func(ctx context.Context, game *domain.Game) error
	FindByIDFn            func(ctx context.Context, id uuid.UUID) (*domain.Game, error)
	FindBySlugFn          func(ctx context.Context, slug string) (*domain.Game, error)
	FindActiveFn          func(ctx context.Context) ([]*domain.Game, error)
	SearchByNameFn        func(ctx context.Context, query string, limit, offset int) ([]*domain.Game, int, error)
	GetPopularFn          func(ctx context.Context, limit int) ([]*domain.Game, error)
	FindByGenreFn         func(ctx context.Context, genre string) ([]*domain.Game, error)
	UpdateFn              func(ctx context.Context, game *domain.Game) error
	IncrementPostsCountFn func(ctx context.Context, id uuid.UUID) error
	DecrementPostsCountFn func(ctx context.Context, id uuid.UUID) error
}

func (m *MockGameRepository) Create(ctx context.Context, game *domain.Game) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, game)
	}
	return nil
}

func (m *MockGameRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Game, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockGameRepository) FindBySlug(ctx context.Context, slug string) (*domain.Game, error) {
	if m.FindBySlugFn != nil {
		return m.FindBySlugFn(ctx, slug)
	}
	return nil, nil
}

func (m *MockGameRepository) FindActive(ctx context.Context) ([]*domain.Game, error) {
	if m.FindActiveFn != nil {
		return m.FindActiveFn(ctx)
	}
	return nil, nil
}

func (m *MockGameRepository) SearchByName(ctx context.Context, query string, limit, offset int) ([]*domain.Game, int, error) {
	if m.SearchByNameFn != nil {
		return m.SearchByNameFn(ctx, query, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockGameRepository) GetPopular(ctx context.Context, limit int) ([]*domain.Game, error) {
	if m.GetPopularFn != nil {
		return m.GetPopularFn(ctx, limit)
	}
	return nil, nil
}

func (m *MockGameRepository) FindByGenre(ctx context.Context, genre string) ([]*domain.Game, error) {
	if m.FindByGenreFn != nil {
		return m.FindByGenreFn(ctx, genre)
	}
	return nil, nil
}

func (m *MockGameRepository) Update(ctx context.Context, game *domain.Game) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, game)
	}
	return nil
}

func (m *MockGameRepository) IncrementPostsCount(ctx context.Context, id uuid.UUID) error {
	if m.IncrementPostsCountFn != nil {
		return m.IncrementPostsCountFn(ctx, id)
	}
	return nil
}

func (m *MockGameRepository) DecrementPostsCount(ctx context.Context, id uuid.UUID) error {
	if m.DecrementPostsCountFn != nil {
		return m.DecrementPostsCountFn(ctx, id)
	}
	return nil
}
