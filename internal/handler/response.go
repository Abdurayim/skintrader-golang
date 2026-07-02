package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "skintrader-go/internal/pkg/errors"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message,omitempty"`
	Data       any    `json:"data"`
	Pagination any    `json:"pagination"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func Success(c *gin.Context, data any, message string) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c *gin.Context, data any, message string) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Paginated(c *gin.Context, data any, pagination any, message string) {
	// The frontend reads pagination from inside data (data.pagination),
	// so mirror it there in addition to the top-level field.
	if m, ok := data.(gin.H); ok {
		m["pagination"] = pagination
	}
	c.JSON(http.StatusOK, PaginatedResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	})
}

func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*apperr.AppError); ok {
		c.JSON(appErr.StatusCode, ErrorResponse{
			Success: false,
			Message: appErr.Message,
			Code:    appErr.Code,
			Details: appErr.Details,
		})
		return
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Message: "Internal server error",
		Code:    apperr.CodeInternal,
	})
}

func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Success: false,
		Message: message,
		Code:    apperr.CodeBadRequest,
	})
}

func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Success: false,
		Message: message,
		Code:    apperr.CodeUnauthorized,
	})
}

func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, ErrorResponse{
		Success: false,
		Message: message,
		Code:    apperr.CodeForbidden,
	})
}

func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Success: false,
		Message: message,
		Code:    apperr.CodeNotFound,
	})
}

func TooManyRequests(c *gin.Context, message string) {
	c.JSON(http.StatusTooManyRequests, ErrorResponse{
		Success: false,
		Message: message,
		Code:    apperr.CodeTooManyRequests,
	})
}

func ValidationError(c *gin.Context, details any) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Success: false,
		Message: "Validation failed",
		Code:    apperr.CodeValidation,
		Details: details,
	})
}
