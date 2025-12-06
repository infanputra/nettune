// Package types provides shared data types
package types

import (
	"errors"
	"fmt"
)

// Common errors
var (
	ErrProfileNotFound   = errors.New("profile not found")
	ErrSnapshotNotFound  = errors.New("snapshot not found")
	ErrApplyInProgress   = errors.New("another apply operation is in progress")
	ErrRollbackFailed    = errors.New("rollback failed")
	ErrValidationFailed  = errors.New("validation failed")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrSystemUnavailable = errors.New("system operation unavailable")
)

// APIError represents an API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAPIError creates a new API error
func NewAPIError(code, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

// NewAPIErrorWithDetails creates a new API error with details
func NewAPIErrorWithDetails(code, message, details string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Error codes
const (
	ErrCodeProfileNotFound   = "PROFILE_NOT_FOUND"
	ErrCodeSnapshotNotFound  = "SNAPSHOT_NOT_FOUND"
	ErrCodeApplyInProgress   = "APPLY_IN_PROGRESS"
	ErrCodeRollbackFailed    = "ROLLBACK_FAILED"
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeInvalidRequest    = "INVALID_REQUEST"
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeSystemUnavailable = "SYSTEM_UNAVAILABLE"
)
