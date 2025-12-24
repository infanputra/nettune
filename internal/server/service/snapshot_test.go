package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jtsang4/nettune/internal/shared/types"
)

func TestSnapshotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	snapshotDir := filepath.Join(tmpDir, "snapshots")

	// Verify directory doesn't exist yet
	if _, err := os.Stat(snapshotDir); !os.IsNotExist(err) {
		t.Error("Snapshot directory should not exist yet")
	}
}

func TestSnapshotToMeta(t *testing.T) {
	snapshot := &types.Snapshot{
		ID: "2024-01-01T00-00-00Z_abc123",
		State: &types.SystemState{
			Sysctl: map[string]string{
				"net.ipv4.tcp_congestion_control": "bbr",
			},
		},
	}

	meta := snapshot.ToMeta()
	if meta.ID != snapshot.ID {
		t.Errorf("Meta ID = %s, want %s", meta.ID, snapshot.ID)
	}
}

func TestApplyRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *types.ApplyRequest
		wantErr bool
	}{
		{
			name: "valid dry run",
			req: &types.ApplyRequest{
				ProfileID: "bbr-fq-default",
				Mode:      "dry_run",
			},
			wantErr: false,
		},
		{
			name: "valid commit",
			req: &types.ApplyRequest{
				ProfileID:           "bbr-fq-default",
				Mode:                "commit",
				AutoRollbackSeconds: 60,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.ProfileID == "" {
				t.Error("ProfileID should not be empty")
			}
		})
	}
}

func TestApplyResult(t *testing.T) {
	result := &types.ApplyResult{
		Mode:       "commit",
		ProfileID:  "bbr-fq-default",
		SnapshotID: "snapshot-123",
		Success:    true,
	}

	if result.Mode != "commit" {
		t.Errorf("Mode = %s, want commit", result.Mode)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
}

func TestApplyPlan(t *testing.T) {
	plan := &types.ApplyPlan{
		SysctlChanges: map[string]*types.Change{
			"net.ipv4.tcp_congestion_control": {
				From: "cubic",
				To:   "bbr",
			},
		},
	}

	if len(plan.SysctlChanges) != 1 {
		t.Errorf("Expected 1 sysctl change, got %d", len(plan.SysctlChanges))
	}

	change, ok := plan.SysctlChanges["net.ipv4.tcp_congestion_control"]
	if !ok {
		t.Fatal("change for tcp_congestion_control not found")
	}
	if change.From != "cubic" {
		t.Errorf("Change from = %v, want cubic", change.From)
	}
	if change.To != "bbr" {
		t.Errorf("Change to = %v, want bbr", change.To)
	}
}

func TestVerificationResult(t *testing.T) {
	tests := []struct {
		name    string
		result  *types.VerificationResult
		allGood bool
	}{
		{
			name: "all verified",
			result: &types.VerificationResult{
				SysctlOK:  true,
				QdiscOK:   true,
				SystemdOK: true,
			},
			allGood: true,
		},
		{
			name: "sysctl failed",
			result: &types.VerificationResult{
				SysctlOK:  false,
				QdiscOK:   true,
				SystemdOK: true,
				Errors:    []string{"sysctl not applied"},
			},
			allGood: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allGood := tt.result.SysctlOK && tt.result.QdiscOK && tt.result.SystemdOK
			if allGood != tt.allGood {
				t.Errorf("AllGood = %v, want %v", allGood, tt.allGood)
			}
		})
	}
}

func TestFormatSysctlValue(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"integer", 1, "1"},
		{"large int", 33554432, "33554432"},
		{"string", "bbr", "bbr"},
		{"float as int", float64(42), "42"},
		{"tcp buffer string", "4096 87380 16777216", "4096 87380 16777216"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSysctlValue(tt.value)
			if result != tt.expected {
				t.Errorf("formatSysctlValue(%v) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSysctlValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"single space", "4096 87380 16777216", "4096 87380 16777216"},
		{"multiple spaces", "4096  87380  16777216", "4096 87380 16777216"},
		{"tabs", "4096\t87380\t16777216", "4096 87380 16777216"},
		{"mixed whitespace", "4096 \t 87380\t\t16777216", "4096 87380 16777216"},
		{"leading/trailing spaces", "  bbr  ", "bbr"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSysctlValue(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSysctlValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChange(t *testing.T) {
	change := &types.Change{
		From: "cubic",
		To:   "bbr",
	}

	if change.From != "cubic" {
		t.Errorf("From = %v, want cubic", change.From)
	}
	if change.To != "bbr" {
		t.Errorf("To = %v, want bbr", change.To)
	}
}

func TestSystemState(t *testing.T) {
	state := &types.SystemState{
		Sysctl: map[string]string{
			"net.ipv4.tcp_congestion_control": "bbr",
			"net.core.default_qdisc":          "fq",
		},
		Qdisc: map[string]*types.QdiscInfo{
			"eth0": {
				Type:   "fq",
				Handle: "8001:",
			},
		},
	}

	if len(state.Sysctl) != 2 {
		t.Errorf("Expected 2 sysctl entries, got %d", len(state.Sysctl))
	}
	if state.Sysctl["net.ipv4.tcp_congestion_control"] != "bbr" {
		t.Error("tcp_congestion_control should be bbr")
	}
}

func TestRollbackRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *types.RollbackRequest
	}{
		{
			name: "by snapshot id",
			req: &types.RollbackRequest{
				SnapshotID: "snapshot-123",
			},
		},
		{
			name: "rollback last",
			req: &types.RollbackRequest{
				RollbackLast: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.SnapshotID == "" && !tt.req.RollbackLast {
				t.Error("Either SnapshotID or RollbackLast should be set")
			}
		})
	}
}

func TestRollbackResult(t *testing.T) {
	result := &types.RollbackResult{
		SnapshotID: "snapshot-123",
		Success:    true,
		CurrentState: &types.SystemState{
			Sysctl: map[string]string{},
		},
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.SnapshotID != "snapshot-123" {
		t.Errorf("SnapshotID = %s, want snapshot-123", result.SnapshotID)
	}
}
