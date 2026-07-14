package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
)

const (
	// topupMinAmount is the smallest accepted top-up (one post).
	topupMinAmount int64 = domain.PostCostUZS
	// topupMaxAmount guards against typos (10 million so'm).
	topupMaxAmount int64 = 10_000_000
	chequeDir            = "uploads/cheques"
)

// BalanceHandler serves user balance info and manual top-up requests.
type BalanceHandler struct {
	userRepo   domain.UserRepository
	topupRepo  domain.BalanceTopupRepository
	cardNumber string
	logger     zerolog.Logger
}

func NewBalanceHandler(userRepo domain.UserRepository, topupRepo domain.BalanceTopupRepository, cardNumber string, logger zerolog.Logger) *BalanceHandler {
	return &BalanceHandler{
		userRepo:   userRepo,
		topupRepo:  topupRepo,
		cardNumber: cardNumber,
		logger:     logger.With().Str("handler", "balance").Logger(),
	}
}

// GetBalance returns the user's balance, the per-post cost, and the
// card number for manual top-ups.
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	user, err := h.userRepo.FindByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to load user for balance")
		Error(c, err)
		return
	}

	Success(c, gin.H{
		"balance":    user.Balance,
		"postCost":   domain.PostCostUZS,
		"cardNumber": h.cardNumber,
		"currency":   "UZS",
	}, "Balance retrieved")
}

// CreateTopup accepts a multipart form with an amount and a cheque
// (payment receipt) image, and creates a pending top-up request.
func (h *BalanceHandler) CreateTopup(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	amountStr := strings.TrimSpace(c.PostForm("amount"))
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount < topupMinAmount || amount > topupMaxAmount {
		BadRequest(c, fmt.Sprintf("Amount must be a whole number between %d and %d UZS", topupMinAmount, topupMaxAmount))
		return
	}

	// Accept the cheque under "cheque" or "file"
	file, header, err := c.Request.FormFile("cheque")
	if err != nil && errors.Is(err, http.ErrMissingFile) {
		file, header, err = c.Request.FormFile("file")
	}
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, ErrorResponse{
				Success: false,
				Message: "File is too large. Maximum request size is 15MB",
				Code:    "PAYLOAD_TOO_LARGE",
			})
			return
		}
		h.logger.Warn().Err(err).Msg("topup: failed to parse cheque file")
		BadRequest(c, "Cheque image is required")
		return
	}
	defer file.Close()

	// Basic image type check by extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".heic", ".heif", ".pdf":
	default:
		BadRequest(c, "Cheque must be an image (JPG, PNG, WEBP, HEIC) or PDF")
		return
	}

	if err := os.MkdirAll(chequeDir, 0755); err != nil {
		h.logger.Error().Err(err).Msg("failed to create cheque upload directory")
		Error(c, err)
		return
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	filePath := filepath.Join(chequeDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to create cheque file")
		Error(c, err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Error().Err(err).Str("path", filePath).Msg("failed to write cheque file")
		_ = os.Remove(filePath)
		Error(c, err)
		return
	}

	topup := &domain.BalanceTopup{
		UserID:     userID,
		Amount:     amount,
		ChequePath: filePath,
	}
	if err := h.topupRepo.Create(c.Request.Context(), topup); err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to create top-up request")
		_ = os.Remove(filePath)
		Error(c, err)
		return
	}

	h.logger.Info().
		Str("userID", userID.String()).
		Int64("amount", amount).
		Str("topupID", topup.ID.String()).
		Msg("balance top-up requested")

	Created(c, gin.H{"topup": topup}, "Top-up request submitted. It will be reviewed by an administrator.")
}

// GetMyTopups returns the user's top-up history.
func (h *BalanceHandler) GetMyTopups(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	topups, err := h.topupRepo.FindByUser(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to list top-ups")
		Error(c, err)
		return
	}

	Success(c, gin.H{"topups": topups}, "Top-up history retrieved")
}
