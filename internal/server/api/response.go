// Package api provides HTTP API server implementation
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jtsang4/nettune/internal/shared/types"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details in API response
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Success sends a successful response
func Success(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithStatus sends a successful response with custom status code
func SuccessWithStatus(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Response{
		Success: true,
		Data:    data,
	})
}

// Error sends an error response
func Error(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails sends an error response with details
func ErrorWithDetails(c *gin.Context, statusCode int, code, message, details string) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// ErrorFromAPIError sends an error response from an APIError
func ErrorFromAPIError(c *gin.Context, statusCode int, err *types.APIError) {
	c.JSON(statusCode, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
	})
}

// Common error responses

// Unauthorized sends a 401 response
func Unauthorized(c *gin.Context, message string) {
	Error(c, 401, types.ErrCodeUnauthorized, message)
}

// BadRequest sends a 400 response
func BadRequest(c *gin.Context, message string) {
	Error(c, 400, types.ErrCodeInvalidRequest, message)
}

// NotFound sends a 404 response
func NotFound(c *gin.Context, message string) {
	Error(c, 404, "NOT_FOUND", message)
}

// InternalError sends a 500 response
func InternalError(c *gin.Context, message string) {
	Error(c, 500, types.ErrCodeInternalError, message)
}
