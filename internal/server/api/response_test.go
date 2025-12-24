package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jtsang4/nettune/internal/shared/types"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Success(c, map[string]string{"message": "hello"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Error("Response should be successful")
	}
	if resp.Error != nil {
		t.Error("Error should be nil for success response")
	}
}

func TestSuccessWithStatus(t *testing.T) {
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		SuccessWithStatus(c, http.StatusCreated, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !resp.Success {
		t.Error("Response should be successful")
	}
}

func TestError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, "INVALID_INPUT", "bad request")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Success {
		t.Error("Response should not be successful")
	}
	if resp.Error == nil {
		t.Fatal("Error should not be nil")
	}
	if resp.Error.Code != "INVALID_INPUT" {
		t.Errorf("Error code = %s, want INVALID_INPUT", resp.Error.Code)
	}
	if resp.Error.Message != "bad request" {
		t.Errorf("Error message = %s, want 'bad request'", resp.Error.Message)
	}
}

func TestErrorWithDetails(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		ErrorWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed", "field 'name' is required")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Details != "field 'name' is required" {
		t.Errorf("Details = %s, want 'field 'name' is required'", resp.Error.Details)
	}
}

func TestErrorFromAPIError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		apiErr := &types.APIError{
			Code:    "CUSTOM_ERROR",
			Message: "something went wrong",
			Details: "extra info",
		}
		ErrorFromAPIError(c, http.StatusInternalServerError, apiErr)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "CUSTOM_ERROR" {
		t.Errorf("Error code = %s, want CUSTOM_ERROR", resp.Error.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Unauthorized(c, "missing token")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestBadRequest(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		BadRequest(c, "invalid input")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestNotFound(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		NotFound(c, "resource not found")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error.Code != "NOT_FOUND" {
		t.Errorf("Error code = %s, want NOT_FOUND", resp.Error.Code)
	}
}

func TestInternalError(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		InternalError(c, "something broke")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestResponseStructure(t *testing.T) {
	resp := Response{
		Success: true,
		Data:    map[string]string{"key": "value"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Verify JSON structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if _, ok := unmarshaled["success"]; !ok {
		t.Error("Response should have 'success' field")
	}
	if _, ok := unmarshaled["data"]; !ok {
		t.Error("Response should have 'data' field")
	}
}

func TestErrorInfoStructure(t *testing.T) {
	errorInfo := ErrorInfo{
		Code:    "TEST_ERROR",
		Message: "test message",
		Details: "test details",
	}

	data, err := json.Marshal(errorInfo)
	if err != nil {
		t.Fatalf("Failed to marshal error info: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal error info: %v", err)
	}

	if unmarshaled["code"] != "TEST_ERROR" {
		t.Error("Error info should have correct 'code' field")
	}
	if unmarshaled["message"] != "test message" {
		t.Error("Error info should have correct 'message' field")
	}
}
