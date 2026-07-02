package mocks

import (
	"context"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockPostRepository implements domain.PostRepository for testing.
type MockPostRepository struct {
	CreateFn          func(ctx context.Context, post *domain.Post) error
	FindByIDFn        func(ctx context.Context, id uuid.UUID) (*domain.Post, error)
	FindActiveFn      func(ctx context.Context, limit, offset int) ([]*domain.Post, int64, error)
	FindByUserFn      func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Post, int64, error)
	SearchFn          func(ctx context.Context, query string, limit, offset int) ([]*domain.Post, int64, error)
	UpdateFn          func(ctx context.Context, post *domain.Post) error
	SoftDeleteFn      func(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID, deletedByType string) error
	IncrementViewsFn  func(ctx context.Context, id uuid.UUID) error
	UpdateStatusFn    func(ctx context.Context, id uuid.UUID, status domain.PostStatus) error
	AddImageFn        func(ctx context.Context, image *domain.PostImage) error
	RemoveImageFn     func(ctx context.Context, imageID uuid.UUID) error
	ListWithFiltersFn func(ctx context.Context, filter domain.PostListFilter) ([]*domain.Post, int64, error)
}

func (m *MockPostRepository) Create(ctx context.Context, post *domain.Post) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, post)
	}
	return nil
}

func (m *MockPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Post, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockPostRepository) FindActive(ctx context.Context, limit, offset int) ([]*domain.Post, int64, error) {
	if m.FindActiveFn != nil {
		return m.FindActiveFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockPostRepository) FindByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Post, int64, error) {
	if m.FindByUserFn != nil {
		return m.FindByUserFn(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockPostRepository) Search(ctx context.Context, query string, limit, offset int) ([]*domain.Post, int64, error) {
	if m.SearchFn != nil {
		return m.SearchFn(ctx, query, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockPostRepository) Update(ctx context.Context, post *domain.Post) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, post)
	}
	return nil
}

func (m *MockPostRepository) SoftDelete(ctx context.Context, id uuid.UUID, deletedBy uuid.UUID, deletedByType string) error {
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(ctx, id, deletedBy, deletedByType)
	}
	return nil
}

func (m *MockPostRepository) IncrementViews(ctx context.Context, id uuid.UUID) error {
	if m.IncrementViewsFn != nil {
		return m.IncrementViewsFn(ctx, id)
	}
	return nil
}

func (m *MockPostRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PostStatus) error {
	if m.UpdateStatusFn != nil {
		return m.UpdateStatusFn(ctx, id, status)
	}
	return nil
}

func (m *MockPostRepository) AddImage(ctx context.Context, image *domain.PostImage) error {
	if m.AddImageFn != nil {
		return m.AddImageFn(ctx, image)
	}
	return nil
}

func (m *MockPostRepository) RemoveImage(ctx context.Context, imageID uuid.UUID) error {
	if m.RemoveImageFn != nil {
		return m.RemoveImageFn(ctx, imageID)
	}
	return nil
}

func (m *MockPostRepository) ListWithFilters(ctx context.Context, filter domain.PostListFilter) ([]*domain.Post, int64, error) {
	if m.ListWithFiltersFn != nil {
		return m.ListWithFiltersFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *MockPostRepository) FindByUserWithStatus(ctx context.Context, userID uuid.UUID, status *domain.PostStatus, limit, offset int) ([]*domain.Post, int64, error) {
	return nil, 0, nil
}

func (m *MockPostRepository) CountByStatus(ctx context.Context, status domain.PostStatus) (int64, error) {
	return 0, nil
}

func (m *MockPostRepository) CountAll(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockPostRepository) FindRecent(ctx context.Context, limit int) ([]*domain.Post, error) {
	return nil, nil
}
