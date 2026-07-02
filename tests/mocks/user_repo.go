package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"

	"skintrader-go/internal/domain"
)

// MockUserRepository implements domain.UserRepository for testing.
type MockUserRepository struct {
	CreateFn                 func(ctx context.Context, user *domain.User) error
	FindByIDFn               func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByGoogleIDFn         func(ctx context.Context, googleID string) (*domain.User, error)
	FindByAppleIDFn          func(ctx context.Context, appleID string) (*domain.User, error)
	FindByEmailFn            func(ctx context.Context, email string) (*domain.User, error)
	UpdateFn                 func(ctx context.Context, user *domain.User) error
	UpdateKYCStatusFn        func(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, reviewedBy uuid.UUID, reason string) error
	UpdateSubscriptionStatusFn func(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error
	UpdateLocationFn         func(ctx context.Context, userID uuid.UUID, latitude, longitude float64) error
	DeleteFn                 func(ctx context.Context, id uuid.UUID) error
	FindNearbyFn             func(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*domain.User, error)
	SearchByNameFn           func(ctx context.Context, query string, limit, offset int) ([]*domain.User, int, error)
	CountByStatusFn          func(ctx context.Context, status domain.UserStatus) (int64, error)
	ListWithFiltersFn        func(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error)
	SaveKYCDocumentFn        func(ctx context.Context, doc *domain.KYCDocument) error
	GetKYCDocumentsFn        func(ctx context.Context, userID uuid.UUID) ([]*domain.KYCDocument, error)
	GetKYCDocumentByTypeFn   func(ctx context.Context, userID uuid.UUID, docType domain.KYCDocumentType) (*domain.KYCDocument, error)
	UpdateFaceMatchScoreFn   func(ctx context.Context, userID uuid.UUID, score float32) error
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) FindByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	if m.FindByGoogleIDFn != nil {
		return m.FindByGoogleIDFn(ctx, googleID)
	}
	return nil, nil
}

func (m *MockUserRepository) FindByAppleID(ctx context.Context, appleID string) (*domain.User, error) {
	if m.FindByAppleIDFn != nil {
		return m.FindByAppleIDFn(ctx, appleID)
	}
	return nil, nil
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.FindByEmailFn != nil {
		return m.FindByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) UpdateKYCStatus(ctx context.Context, userID uuid.UUID, status domain.KYCStatus, reviewedBy uuid.UUID, reason string) error {
	if m.UpdateKYCStatusFn != nil {
		return m.UpdateKYCStatusFn(ctx, userID, status, reviewedBy, reason)
	}
	return nil
}

func (m *MockUserRepository) UpdateSubscriptionStatus(ctx context.Context, userID uuid.UUID, status domain.SubscriptionStatus, subscriptionID *uuid.UUID, expiresAt *time.Time) error {
	if m.UpdateSubscriptionStatusFn != nil {
		return m.UpdateSubscriptionStatusFn(ctx, userID, status, subscriptionID, expiresAt)
	}
	return nil
}

func (m *MockUserRepository) UpdateLocation(ctx context.Context, userID uuid.UUID, latitude, longitude float64) error {
	if m.UpdateLocationFn != nil {
		return m.UpdateLocationFn(ctx, userID, latitude, longitude)
	}
	return nil
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *MockUserRepository) FindNearby(ctx context.Context, latitude, longitude, radiusKM float64, limit int) ([]*domain.User, error) {
	if m.FindNearbyFn != nil {
		return m.FindNearbyFn(ctx, latitude, longitude, radiusKM, limit)
	}
	return nil, nil
}

func (m *MockUserRepository) SearchByName(ctx context.Context, query string, limit, offset int) ([]*domain.User, int, error) {
	if m.SearchByNameFn != nil {
		return m.SearchByNameFn(ctx, query, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockUserRepository) CountByStatus(ctx context.Context, status domain.UserStatus) (int64, error) {
	if m.CountByStatusFn != nil {
		return m.CountByStatusFn(ctx, status)
	}
	return 0, nil
}

func (m *MockUserRepository) ListWithFilters(ctx context.Context, filter domain.UserListFilter) ([]*domain.User, int64, error) {
	if m.ListWithFiltersFn != nil {
		return m.ListWithFiltersFn(ctx, filter)
	}
	return nil, 0, nil
}

func (m *MockUserRepository) SaveKYCDocument(ctx context.Context, doc *domain.KYCDocument) error {
	if m.SaveKYCDocumentFn != nil {
		return m.SaveKYCDocumentFn(ctx, doc)
	}
	return nil
}

func (m *MockUserRepository) GetKYCDocuments(ctx context.Context, userID uuid.UUID) ([]*domain.KYCDocument, error) {
	if m.GetKYCDocumentsFn != nil {
		return m.GetKYCDocumentsFn(ctx, userID)
	}
	return nil, nil
}

func (m *MockUserRepository) GetKYCDocumentByType(ctx context.Context, userID uuid.UUID, docType domain.KYCDocumentType) (*domain.KYCDocument, error) {
	if m.GetKYCDocumentByTypeFn != nil {
		return m.GetKYCDocumentByTypeFn(ctx, userID, docType)
	}
	return nil, nil
}

func (m *MockUserRepository) UpdateFaceMatchScore(ctx context.Context, userID uuid.UUID, score float32) error {
	if m.UpdateFaceMatchScoreFn != nil {
		return m.UpdateFaceMatchScoreFn(ctx, userID, score)
	}
	return nil
}

func (m *MockUserRepository) CountByKYCStatus(ctx context.Context, status domain.KYCStatus) (int64, error) {
	return 0, nil
}

func (m *MockUserRepository) FindRecent(ctx context.Context, limit int) ([]*domain.User, error) {
	return nil, nil
}
