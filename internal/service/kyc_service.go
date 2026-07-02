package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/config"
	"skintrader-go/internal/domain"
)

type KYCService struct {
	userRepo     domain.UserRepository
	kycConfig    config.KYCConfig
	faceConfig   config.FaceServiceConfig
	uploadConfig config.UploadConfig
	logger       zerolog.Logger
}

func NewKYCService(userRepo domain.UserRepository, kycConfig config.KYCConfig, faceConfig config.FaceServiceConfig, uploadConfig config.UploadConfig, logger zerolog.Logger) *KYCService {
	return &KYCService{
		userRepo:     userRepo,
		kycConfig:    kycConfig,
		faceConfig:   faceConfig,
		uploadConfig: uploadConfig,
		logger:       logger.With().Str("service", "kyc").Logger(),
	}
}

// UploadDocument handles KYC document upload and sets status to pending.
func (s *KYCService) UploadDocument(ctx context.Context, userID uuid.UUID, filePath string, docType domain.KYCDocumentType) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if user.KYCStatus == domain.KYCStatusVerified {
		return fmt.Errorf("user is already KYC verified")
	}

	// Persist the document so admins can review it
	doc := &domain.KYCDocument{
		ID:         uuid.New(),
		UserID:     userID,
		DocType:    docType,
		FilePath:   filePath,
		UploadedAt: time.Now(),
	}
	if err := s.userRepo.SaveKYCDocument(ctx, doc); err != nil {
		return fmt.Errorf("save kyc document: %w", err)
	}

	// Update KYC status to pending (admin will review)
	emptyID := uuid.UUID{}
	if err := s.userRepo.UpdateKYCStatus(ctx, userID, domain.KYCStatusPending, emptyID, ""); err != nil {
		return fmt.Errorf("update kyc status: %w", err)
	}

	s.logger.Info().
		Str("user_id", userID.String()).
		Str("doc_type", string(docType)).
		Str("file_path", filePath).
		Msg("KYC document uploaded, status set to pending")

	return nil
}

// AutoVerify triggers automatic KYC verification using face matching via gRPC.
// If the face service is not available, it keeps the status as pending for manual review.
func (s *KYCService) AutoVerify(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	if user.KYCStatus == domain.KYCStatusVerified {
		return fmt.Errorf("user is already KYC verified")
	}

	if user.KYCStatus != domain.KYCStatusPending {
		return fmt.Errorf("user must upload KYC documents first")
	}

	// Face matching requires the gRPC face service to be running.
	// If the service is not configured, keep as pending for manual review.
	if s.faceConfig.Address == "" {
		s.logger.Warn().
			Str("user_id", userID.String()).
			Msg("face service not configured, KYC requires manual review")
		return nil
	}

	// TODO: When face-service gRPC client is available:
	// 1. Load the ID document and selfie images
	// 2. Call face service CompareFaces RPC
	// 3. If confidence >= threshold, auto-approve
	// 4. Otherwise keep pending for manual review
	//
	// For now, log and keep pending.
	s.logger.Info().
		Str("user_id", userID.String()).
		Msg("auto-verify requested, pending face service integration")

	return nil
}

// AdminApprove manually approves KYC verification.
func (s *KYCService) AdminApprove(ctx context.Context, userID uuid.UUID, adminID uuid.UUID) error {
	return s.userRepo.UpdateKYCStatus(ctx, userID, domain.KYCStatusVerified, adminID, "")
}

// AdminReject manually rejects KYC verification.
func (s *KYCService) AdminReject(ctx context.Context, userID uuid.UUID, adminID uuid.UUID, reason string) error {
	return s.userRepo.UpdateKYCStatus(ctx, userID, domain.KYCStatusRejected, adminID, reason)
}

// GetStatus returns the KYC status for a user.
func (s *KYCService) GetStatus(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.FindByID(ctx, userID)
}
