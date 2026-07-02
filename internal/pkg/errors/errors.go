package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Details    any    `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code string, message string, statusCode int) *AppError {
	return &AppError{Code: code, Message: message, StatusCode: statusCode}
}

func WithDetails(err *AppError, details any) *AppError {
	return &AppError{Code: err.Code, Message: err.Message, StatusCode: err.StatusCode, Details: details}
}

func Wrap(err error, code string, statusCode int) *AppError {
	return &AppError{Code: code, Message: err.Error(), StatusCode: statusCode}
}

// Common error codes
const (
	CodeInvalidToken        = "INVALID_TOKEN"
	CodeTokenExpired        = "TOKEN_EXPIRED"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeBadRequest          = "BAD_REQUEST"
	CodeValidation          = "VALIDATION_ERROR"
	CodeConflict            = "CONFLICT"
	CodeTooManyRequests     = "TOO_MANY_REQUESTS"
	CodeInternal            = "INTERNAL_ERROR"
	CodeKYCRequired         = "KYC_REQUIRED"
	CodeSubscriptionReq     = "SUBSCRIPTION_REQUIRED"
	CodeAccountSuspended    = "ACCOUNT_SUSPENDED"
	CodeAccountBanned       = "ACCOUNT_BANNED"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeEmailExists         = "EMAIL_EXISTS"
	CodeEmailNotVerified    = "EMAIL_NOT_VERIFIED"
	CodeInvalidPayment      = "INVALID_PAYMENT"
	CodePaymentFailed       = "PAYMENT_FAILED"
	CodeDuplicateReport     = "DUPLICATE_REPORT"
	CodeReportLimitExceeded = "REPORT_LIMIT_EXCEEDED"
	CodeFileTooLarge        = "FILE_TOO_LARGE"
	CodeInvalidFileType     = "INVALID_FILE_TYPE"
	CodeMaxFilesExceeded    = "MAX_FILES_EXCEEDED"
)

// Pre-built errors
var (
	ErrInvalidToken       = New(CodeInvalidToken, "Invalid or malformed token", http.StatusUnauthorized)
	ErrTokenExpired       = New(CodeTokenExpired, "Token has expired", http.StatusUnauthorized)
	ErrUnauthorized       = New(CodeUnauthorized, "Authentication required", http.StatusUnauthorized)
	ErrForbidden          = New(CodeForbidden, "You do not have permission to perform this action", http.StatusForbidden)
	ErrNotFound           = New(CodeNotFound, "Resource not found", http.StatusNotFound)
	ErrBadRequest         = New(CodeBadRequest, "Invalid request", http.StatusBadRequest)
	ErrInternal           = New(CodeInternal, "Internal server error", http.StatusInternalServerError)
	ErrKYCRequired        = New(CodeKYCRequired, "KYC verification is required", http.StatusForbidden)
	ErrSubscriptionReq    = New(CodeSubscriptionReq, "Active subscription is required", http.StatusForbidden)
	ErrAccountSuspended   = New(CodeAccountSuspended, "Account is suspended", http.StatusForbidden)
	ErrAccountBanned      = New(CodeAccountBanned, "Account is banned", http.StatusForbidden)
	ErrInvalidCredentials = New(CodeInvalidCredentials, "Invalid email or password", http.StatusUnauthorized)
	ErrEmailExists        = New(CodeEmailExists, "Email is already registered", http.StatusConflict)
	ErrEmailNotVerified   = New(CodeEmailNotVerified, "Email is not verified", http.StatusForbidden)
	ErrDuplicateReport    = New(CodeDuplicateReport, "You have already reported this", http.StatusConflict)
	ErrTooManyRequests    = New(CodeTooManyRequests, "Too many requests, please try again later", http.StatusTooManyRequests)
)

func NotFound(resource string) *AppError {
	return New(CodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

func BadRequest(message string) *AppError {
	return New(CodeBadRequest, message, http.StatusBadRequest)
}

func Forbidden(message string) *AppError {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func Conflict(message string) *AppError {
	return New(CodeConflict, message, http.StatusConflict)
}

func Validation(details any) *AppError {
	return &AppError{Code: CodeValidation, Message: "Validation failed", StatusCode: http.StatusBadRequest, Details: details}
}

func Internal(message string) *AppError {
	return New(CodeInternal, message, http.StatusInternalServerError)
}
