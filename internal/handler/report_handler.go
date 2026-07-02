package handler

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"skintrader-go/internal/domain"
	"skintrader-go/internal/middleware"
	apperr "skintrader-go/internal/pkg/errors"
)

type ReportHandler struct {
	reportRepo     domain.ReportRepository
	authMiddleware *middleware.AuthMiddleware
	logger         zerolog.Logger
}

func NewReportHandler(reportService interface{}, authMiddleware *middleware.AuthMiddleware, logger zerolog.Logger) *ReportHandler {
	return &ReportHandler{
		reportRepo:     reportService.(domain.ReportRepository),
		authMiddleware: authMiddleware,
		logger:         logger.With().Str("handler", "report").Logger(),
	}
}

// CreateReport creates a new report.
func (h *ReportHandler) CreateReport(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	var req struct {
		TargetID    string `json:"targetId" binding:"required"`
		TargetType  string `json:"targetType" binding:"required"`
		Category    string `json:"category" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "target_id, target_type, and category are required")
		return
	}

	// Validate target_type
	req.TargetType = strings.TrimSpace(strings.ToLower(req.TargetType))
	var reportType domain.ReportType
	switch req.TargetType {
	case "user":
		reportType = domain.ReportTypeUser
	case "post":
		reportType = domain.ReportTypePost
	default:
		BadRequest(c, "target_type must be 'user' or 'post'")
		return
	}

	// Parse target ID
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		BadRequest(c, "Invalid target_id format")
		return
	}

	// Prevent self-reporting
	if reportType == domain.ReportTypeUser && targetID == userID {
		BadRequest(c, "You cannot report yourself")
		return
	}

	// Validate category
	category := domain.ReportCategory(strings.TrimSpace(req.Category))
	if !isValidReportCategory(category) {
		BadRequest(c, "Invalid report category")
		return
	}

	// Validate description length
	req.Description = strings.TrimSpace(req.Description)
	if len(req.Description) > 2000 {
		BadRequest(c, "Description must be 2000 characters or less")
		return
	}

	// Generate a hash to check for duplicate reports
	reportHash := generateReportHash(userID, targetID, reportType)

	// Check for duplicate
	isDuplicate, err := h.reportRepo.CheckDuplicate(c.Request.Context(), reportHash)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to check duplicate report")
		Error(c, apperr.Internal("Failed to check duplicate report"))
		return
	}
	if isDuplicate {
		Error(c, apperr.ErrDuplicateReport)
		return
	}

	report := &domain.Report{
		ID:          uuid.New(),
		ReporterID:  userID,
		ReportType:  reportType,
		TargetID:    targetID,
		TargetModel: req.TargetType,
		Category:    category,
		Description: req.Description,
		Status:      domain.ReportStatusPending,
		Priority:    domain.ReportPriorityMedium,
		ReportHash:  reportHash,
		ReportCount: 1,
	}

	// Set IP and user agent
	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	report.IPAddress = &ip
	report.UserAgent = &ua

	if err := h.reportRepo.Create(c.Request.Context(), report); err != nil {
		h.logger.Error().Err(err).Str("reporterID", userID.String()).Msg("failed to create report")
		Error(c, apperr.Internal("Failed to create report"))
		return
	}

	h.logger.Info().
		Str("reportID", report.ID.String()).
		Str("reporterID", userID.String()).
		Str("targetID", targetID.String()).
		Str("targetType", req.TargetType).
		Msg("report created")

	Created(c, gin.H{
		"report": report,
	}, "Report submitted successfully")
}

// GetMyReports returns the authenticated user's reports.
func (h *ReportHandler) GetMyReports(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	reports, total, err := h.reportRepo.FindByReporter(c.Request.Context(), userID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("userID", userID.String()).Msg("failed to get user reports")
		Error(c, apperr.Internal("Failed to retrieve reports"))
		return
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	Paginated(c, gin.H{
		"reports": reports,
	}, gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"totalPages": totalPages,
	}, "Reports retrieved successfully")
}

// GetReportByID returns a specific report by ID.
func (h *ReportHandler) GetReportByID(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		Unauthorized(c, "Authentication required")
		return
	}

	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		BadRequest(c, "Invalid report ID format")
		return
	}

	report, err := h.reportRepo.FindByID(c.Request.Context(), reportID)
	if err != nil {
		h.logger.Error().Err(err).Str("reportID", reportID.String()).Msg("failed to get report")
		Error(c, apperr.Internal("Failed to retrieve report"))
		return
	}
	if report == nil {
		NotFound(c, "Report not found")
		return
	}

	// Only the reporter can view their own report
	if report.ReporterID != userID {
		Forbidden(c, "You can only view your own reports")
		return
	}

	Success(c, gin.H{
		"report": report,
	}, "Report retrieved successfully")
}

// generateReportHash creates a unique hash for deduplication.
func generateReportHash(reporterID, targetID uuid.UUID, reportType domain.ReportType) string {
	data := fmt.Sprintf("%s:%s:%s", reporterID.String(), targetID.String(), string(reportType))
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// isValidReportCategory checks if a report category is valid.
func isValidReportCategory(cat domain.ReportCategory) bool {
	switch cat {
	case domain.ReportCategoryScam,
		domain.ReportCategoryFakeItem,
		domain.ReportCategoryInappropriateContent,
		domain.ReportCategoryDuplicatePost,
		domain.ReportCategoryIncorrectPricing,
		domain.ReportCategoryHarassment,
		domain.ReportCategorySpam,
		domain.ReportCategoryFraud,
		domain.ReportCategoryImpersonation,
		domain.ReportCategoryOffensiveProfile,
		domain.ReportCategoryOther:
		return true
	default:
		return false
	}
}
