package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
	"skintrader-go/internal/service"
	"skintrader-go/tests/mocks"
	"skintrader-go/tests/testutil"
)

func newKYCService(userRepo *mocks.MockUserRepository, faceAddr string) *service.KYCService {
	logger := zerolog.Nop()
	return service.NewKYCService(
		userRepo,
		config.KYCConfig{},
		config.FaceServiceConfig{Address: faceAddr},
		config.UploadConfig{},
		logger,
	)
}

func TestUploadDocument_Success(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusNotSubmitted

	var updatedStatus domain.KYCStatus
	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, id uuid.UUID) (*domain.User, error) {
			return user, nil
		},
		UpdateKYCStatusFn: func(_ context.Context, _ uuid.UUID, status domain.KYCStatus, _ uuid.UUID, _ string) error {
			updatedStatus = status
			return nil
		},
	}

	svc := newKYCService(userRepo, "")
	err := svc.UploadDocument(context.Background(), user.ID, "/tmp/id.jpg", domain.KYCDocumentTypeIDCard)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if updatedStatus != domain.KYCStatusPending {
		t.Fatalf("expected status pending, got %s", updatedStatus)
	}
}

func TestUploadDocument_AlreadyVerified(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusVerified

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	svc := newKYCService(userRepo, "")
	err := svc.UploadDocument(context.Background(), user.ID, "/tmp/id.jpg", domain.KYCDocumentTypeIDCard)
	if err == nil {
		t.Fatal("expected error for already verified user")
	}
}

func TestUploadDocument_UserNotFound(t *testing.T) {
	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return nil, fmt.Errorf("user not found")
		},
	}

	svc := newKYCService(userRepo, "")
	err := svc.UploadDocument(context.Background(), uuid.New(), "/tmp/id.jpg", domain.KYCDocumentTypeIDCard)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func TestAutoVerify_NoFaceServiceConfigured(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusPending

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	svc := newKYCService(userRepo, "") // empty address = not configured
	err := svc.AutoVerify(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error (should keep pending), got: %v", err)
	}
}

func TestAutoVerify_AlreadyVerified(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusVerified

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	svc := newKYCService(userRepo, "localhost:50051")
	err := svc.AutoVerify(context.Background(), user.ID)
	if err == nil {
		t.Fatal("expected error for already verified user")
	}
}

func TestAutoVerify_NotPending(t *testing.T) {
	user := testutil.NewTestUser()
	user.KYCStatus = domain.KYCStatusNotSubmitted

	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	svc := newKYCService(userRepo, "localhost:50051")
	err := svc.AutoVerify(context.Background(), user.ID)
	if err == nil {
		t.Fatal("expected error for non-pending KYC status")
	}
}

func TestAdminApprove(t *testing.T) {
	var approvedUserID uuid.UUID
	var approvedStatus domain.KYCStatus
	userRepo := &mocks.MockUserRepository{
		UpdateKYCStatusFn: func(_ context.Context, userID uuid.UUID, status domain.KYCStatus, _ uuid.UUID, _ string) error {
			approvedUserID = userID
			approvedStatus = status
			return nil
		},
	}

	svc := newKYCService(userRepo, "")
	userID := uuid.New()
	adminID := uuid.New()
	err := svc.AdminApprove(context.Background(), userID, adminID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if approvedUserID != userID {
		t.Fatalf("expected user %s, got %s", userID, approvedUserID)
	}
	if approvedStatus != domain.KYCStatusVerified {
		t.Fatalf("expected verified status, got %s", approvedStatus)
	}
}

func TestAdminReject(t *testing.T) {
	var rejectedStatus domain.KYCStatus
	var rejectedReason string
	userRepo := &mocks.MockUserRepository{
		UpdateKYCStatusFn: func(_ context.Context, _ uuid.UUID, status domain.KYCStatus, _ uuid.UUID, reason string) error {
			rejectedStatus = status
			rejectedReason = reason
			return nil
		},
	}

	svc := newKYCService(userRepo, "")
	err := svc.AdminReject(context.Background(), uuid.New(), uuid.New(), "blurry document")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if rejectedStatus != domain.KYCStatusRejected {
		t.Fatalf("expected rejected status, got %s", rejectedStatus)
	}
	if rejectedReason != "blurry document" {
		t.Fatalf("expected reason 'blurry document', got %s", rejectedReason)
	}
}

func TestGetStatus(t *testing.T) {
	user := testutil.NewTestUser()
	userRepo := &mocks.MockUserRepository{
		FindByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return user, nil
		},
	}

	svc := newKYCService(userRepo, "")
	result, err := svc.GetStatus(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.ID != user.ID {
		t.Fatalf("expected user %s, got %s", user.ID, result.ID)
	}
}
