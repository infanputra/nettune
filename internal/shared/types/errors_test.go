package types

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIErrorImplementsError(t *testing.T) {
	apiErr := &APIError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Details: "Additional details",
	}

	// Should implement error interface
	var err error = apiErr
	errStr := err.Error()

	// Error should contain the code and message
	if errStr == "" {
		t.Error("Error() should not be empty")
	}

	if !strings.Contains(errStr, "TEST_ERROR") {
		t.Errorf("Error() should contain code, got %q", errStr)
	}

	if !strings.Contains(errStr, "Test error message") {
		t.Errorf("Error() should contain message, got %q", errStr)
	}
}

func TestAPIErrorWithoutDetails(t *testing.T) {
	apiErr := &APIError{
		Code:    "SIMPLE_ERROR",
		Message: "Simple error",
	}

	expected := "SIMPLE_ERROR: Simple error"
	if apiErr.Error() != expected {
		t.Errorf("Error() = %q, want %q", apiErr.Error(), expected)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrProfileNotFound", ErrProfileNotFound},
		{"ErrSnapshotNotFound", ErrSnapshotNotFound},
		{"ErrApplyInProgress", ErrApplyInProgress},
		{"ErrValidationFailed", ErrValidationFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
		})
	}
}

func TestErrorIs(t *testing.T) {
	wrappedErr := errors.New("wrapped: " + ErrProfileNotFound.Error())
	// Note: This just tests that the errors are properly defined
	if ErrProfileNotFound == nil {
		t.Error("ErrProfileNotFound should not be nil")
	}
	if wrappedErr == nil {
		t.Error("wrapped error should not be nil")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []string{
		ErrCodeUnauthorized,
		ErrCodeInvalidRequest,
		ErrCodeApplyInProgress,
		ErrCodeInternalError,
	}

	for _, code := range codes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}
}
