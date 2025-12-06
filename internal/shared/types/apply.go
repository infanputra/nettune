package types

import "time"

// ApplyRequest represents a request to apply a profile
type ApplyRequest struct {
	ProfileID           string `json:"profile_id" validate:"required"`
	Mode                string `json:"mode" validate:"required,oneof=dry_run commit"`
	AutoRollbackSeconds int    `json:"auto_rollback_seconds,omitempty"`
}

// ApplyResult represents the result of an apply operation
type ApplyResult struct {
	Mode         string              `json:"mode"`
	ProfileID    string              `json:"profile_id"`
	SnapshotID   string              `json:"snapshot_id,omitempty"`
	Plan         *ApplyPlan          `json:"plan"`
	Success      bool                `json:"success"`
	AppliedAt    time.Time           `json:"applied_at,omitempty"`
	Verification *VerificationResult `json:"verification,omitempty"`
	Errors       []string            `json:"errors,omitempty"`
}

// ApplyPlan represents the planned changes
type ApplyPlan struct {
	SysctlChanges  map[string]*Change `json:"sysctl_changes"`
	QdiscChanges   map[string]*Change `json:"qdisc_changes"`
	SystemdChanges map[string]*Change `json:"systemd_changes"`
}

// Change represents a single configuration change
type Change struct {
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

// VerificationResult represents the verification after apply
type VerificationResult struct {
	SysctlOK  bool     `json:"sysctl_ok"`
	QdiscOK   bool     `json:"qdisc_ok"`
	SystemdOK bool     `json:"systemd_ok"`
	Errors    []string `json:"errors,omitempty"`
}

// RollbackRequest represents a rollback request
type RollbackRequest struct {
	SnapshotID   string `json:"snapshot_id,omitempty"`
	RollbackLast bool   `json:"rollback_last,omitempty"`
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	SnapshotID   string       `json:"snapshot_id"`
	Success      bool         `json:"success"`
	CurrentState *SystemState `json:"current_state"`
	Errors       []string     `json:"errors,omitempty"`
}

// SystemStatus represents the current system status
type SystemStatus struct {
	LastApply        *LastApplyInfo `json:"last_apply,omitempty"`
	CurrentState     *SystemState   `json:"current_state"`
	SnapshotsCount   int            `json:"snapshots_count"`
	LatestSnapshotID string         `json:"latest_snapshot_id,omitempty"`
}

// LastApplyInfo represents information about the last apply operation
type LastApplyInfo struct {
	ProfileID string    `json:"profile_id"`
	AppliedAt time.Time `json:"applied_at"`
	Success   bool      `json:"success"`
}
