package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/jtsang4/nettune/internal/server/service"
	"github.com/jtsang4/nettune/internal/shared/types"
)

// SystemHandler handles system-related HTTP endpoints
type SystemHandler struct {
	snapshotService *service.SnapshotService
	applyService    *service.ApplyService
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(
	snapshotService *service.SnapshotService,
	applyService *service.ApplyService,
) *SystemHandler {
	return &SystemHandler{
		snapshotService: snapshotService,
		applyService:    applyService,
	}
}

// CreateSnapshot handles POST /sys/snapshot
func (h *SystemHandler) CreateSnapshot(c *gin.Context) {
	snapshot, err := h.snapshotService.Create()
	if err != nil {
		internalError(c, err.Error())
		return
	}

	success(c, gin.H{
		"snapshot_id":   snapshot.ID,
		"current_state": snapshot.State,
	})
}

// GetSnapshot handles GET /sys/snapshot/:id
func (h *SystemHandler) GetSnapshot(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		badRequest(c, "snapshot id is required")
		return
	}

	snapshot, err := h.snapshotService.Get(id)
	if err != nil {
		if errors.Is(err, types.ErrSnapshotNotFound) {
			notFound(c, "snapshot not found")
			return
		}
		internalError(c, err.Error())
		return
	}

	success(c, snapshot)
}

// ListSnapshots handles GET /sys/snapshots
func (h *SystemHandler) ListSnapshots(c *gin.Context) {
	snapshots, err := h.snapshotService.List()
	if err != nil {
		internalError(c, err.Error())
		return
	}

	success(c, gin.H{
		"snapshots": snapshots,
	})
}

// Apply handles POST /sys/apply
func (h *SystemHandler) Apply(c *gin.Context) {
	var req types.ApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	// Validate mode
	if req.Mode != "dry_run" && req.Mode != "commit" {
		badRequest(c, "mode must be 'dry_run' or 'commit'")
		return
	}

	result, err := h.applyService.Apply(&req)
	if err != nil {
		if errors.Is(err, types.ErrProfileNotFound) {
			notFound(c, "profile not found")
			return
		}
		if errors.Is(err, types.ErrApplyInProgress) {
			errorResponse(c, 409, types.ErrCodeApplyInProgress, "another apply operation is in progress")
			return
		}
		internalError(c, err.Error())
		return
	}

	success(c, result)
}

// Rollback handles POST /sys/rollback
func (h *SystemHandler) Rollback(c *gin.Context) {
	var req types.RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	var err error
	var snapshotID string

	if req.RollbackLast {
		snapshot, getErr := h.snapshotService.GetLatest()
		if getErr != nil {
			if errors.Is(getErr, types.ErrSnapshotNotFound) {
				notFound(c, "no snapshots available")
				return
			}
			internalError(c, getErr.Error())
			return
		}
		snapshotID = snapshot.ID
		err = h.applyService.Rollback(snapshotID)
	} else if req.SnapshotID != "" {
		snapshotID = req.SnapshotID
		err = h.applyService.Rollback(snapshotID)
	} else {
		badRequest(c, "either snapshot_id or rollback_last is required")
		return
	}

	if err != nil {
		if errors.Is(err, types.ErrSnapshotNotFound) {
			notFound(c, "snapshot not found")
			return
		}
		if errors.Is(err, types.ErrApplyInProgress) {
			errorResponse(c, 409, types.ErrCodeApplyInProgress, "another operation is in progress")
			return
		}
		internalError(c, err.Error())
		return
	}

	// Get current state after rollback
	currentState, _ := h.snapshotService.GetCurrentState()

	success(c, types.RollbackResult{
		SnapshotID:   snapshotID,
		Success:      true,
		CurrentState: currentState,
	})
}

// Status handles GET /sys/status
func (h *SystemHandler) Status(c *gin.Context) {
	status, err := h.applyService.GetStatus()
	if err != nil {
		internalError(c, err.Error())
		return
	}

	success(c, status)
}

func errorResponse(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, gin.H{"success": false, "error": gin.H{"code": code, "message": message}})
}
