// Package handlers provides HTTP request handlers
package handlers

import (
	"crypto/rand"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jtsang4/nettune/internal/server/service"
)

// MaxDownloadBytes is the maximum size for download tests (500MB)
const MaxDownloadBytes = 500 * 1024 * 1024

// ProbeHandler handles probe-related HTTP endpoints
type ProbeHandler struct {
	probeService *service.ProbeService
}

// NewProbeHandler creates a new ProbeHandler
func NewProbeHandler(probeService *service.ProbeService) *ProbeHandler {
	return &ProbeHandler{
		probeService: probeService,
	}
}

// Echo handles GET /probe/echo
func (h *ProbeHandler) Echo(c *gin.Context) {
	success(c, gin.H{
		"ts": time.Now().UnixMilli(),
		"ok": true,
	})
}

// Download handles GET /probe/download
func (h *ProbeHandler) Download(c *gin.Context) {
	bytesStr := c.DefaultQuery("bytes", "10485760") // Default 10MB
	bytes, err := strconv.ParseInt(bytesStr, 10, 64)
	if err != nil || bytes <= 0 {
		badRequest(c, "invalid bytes parameter")
		return
	}

	if bytes > MaxDownloadBytes {
		badRequest(c, "bytes exceeds maximum allowed (500MB)")
		return
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(bytes, 10))
	c.Header("Content-Encoding", "identity")
	c.Header("Cache-Control", "no-store")

	// Stream random data
	bufSize := int64(64 * 1024) // 64KB buffer
	buf := make([]byte, bufSize)
	written := int64(0)

	for written < bytes {
		toWrite := bufSize
		if bytes-written < bufSize {
			toWrite = bytes - written
		}

		// Generate random data
		if _, err := rand.Read(buf[:toWrite]); err != nil {
			// Connection might be closed
			return
		}

		n, err := c.Writer.Write(buf[:toWrite])
		if err != nil {
			// Connection closed by client
			return
		}
		written += int64(n)

		// Flush periodically
		c.Writer.Flush()
	}
}

// Upload handles POST /probe/upload
func (h *ProbeHandler) Upload(c *gin.Context) {
	startTime := time.Now()

	// Read request body
	buf := make([]byte, 64*1024)
	totalBytes := int64(0)

	for {
		n, err := c.Request.Body.Read(buf)
		totalBytes += int64(n)
		if err != nil {
			break
		}
	}

	duration := time.Since(startTime)

	success(c, gin.H{
		"received_bytes": totalBytes,
		"duration_ms":    duration.Milliseconds(),
	})
}

// Info handles GET /probe/info
func (h *ProbeHandler) Info(c *gin.Context) {
	info, err := h.probeService.GetServerInfo()
	if err != nil {
		internalError(c, err.Error())
		return
	}
	success(c, info)
}

// Helper functions for responses
func success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"success": true, "data": data})
}

func badRequest(c *gin.Context, message string) {
	c.JSON(400, gin.H{"success": false, "error": gin.H{"code": "INVALID_REQUEST", "message": message}})
}

func internalError(c *gin.Context, message string) {
	c.JSON(500, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": message}})
}
