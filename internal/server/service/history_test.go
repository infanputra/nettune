package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewHistoryService(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	if svc == nil {
		t.Fatal("HistoryService should not be nil")
	}

	// Check that journal directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("History directory should exist")
	}
}

func TestHistoryService_RecordApply(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	// Record a successful apply
	svc.RecordApply("bbr-fq-default", "snapshot-123", true)

	// Check last apply info
	lastApply := svc.GetLastApply()
	if lastApply == nil {
		t.Fatal("lastApply should not be nil after successful apply")
	}
	if lastApply.ProfileID != "bbr-fq-default" {
		t.Errorf("ProfileID = %s, want bbr-fq-default", lastApply.ProfileID)
	}
	if !lastApply.Success {
		t.Error("Success should be true")
	}

	// Verify journal file exists
	journalPath := filepath.Join(tmpDir, "journal.jsonl")
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		t.Error("Journal file should exist after record")
	}
}

func TestHistoryService_RecordApply_Failure(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	// Record a failed apply
	svc.RecordApply("bad-profile", "snapshot-123", false)

	// Last apply should still be nil (failed applies don't update it)
	lastApply := svc.GetLastApply()
	if lastApply != nil {
		t.Error("lastApply should be nil after failed apply")
	}
}

func TestHistoryService_RecordRollback(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	svc.RecordRollback("snapshot-123", true)

	// Verify entry was recorded
	entries, err := svc.GetRecentEntries(10)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Action != "rollback" {
		t.Errorf("Action = %s, want rollback", entries[0].Action)
	}
	if entries[0].SnapshotID != "snapshot-123" {
		t.Errorf("SnapshotID = %s, want snapshot-123", entries[0].SnapshotID)
	}
}

func TestHistoryService_RecordSnapshot(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	svc.RecordSnapshot("snapshot-456")

	entries, err := svc.GetRecentEntries(10)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Action != "snapshot" {
		t.Errorf("Action = %s, want snapshot", entries[0].Action)
	}
}

func TestHistoryService_GetRecentEntries(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	// Record multiple entries
	for i := 0; i < 5; i++ {
		svc.RecordApply("profile-"+string(rune('a'+i)), "snapshot-"+string(rune('0'+i)), true)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	t.Run("get all entries", func(t *testing.T) {
		entries, err := svc.GetRecentEntries(0)
		if err != nil {
			t.Fatalf("GetRecentEntries failed: %v", err)
		}
		if len(entries) != 5 {
			t.Errorf("Expected 5 entries, got %d", len(entries))
		}
	})

	t.Run("get limited entries", func(t *testing.T) {
		entries, err := svc.GetRecentEntries(3)
		if err != nil {
			t.Fatalf("GetRecentEntries failed: %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(entries))
		}
	})

	t.Run("entries are in reverse order", func(t *testing.T) {
		entries, err := svc.GetRecentEntries(5)
		if err != nil {
			t.Fatalf("GetRecentEntries failed: %v", err)
		}
		// First entry should be the most recent
		if len(entries) > 1 {
			if entries[0].Timestamp.Before(entries[len(entries)-1].Timestamp) {
				t.Error("Entries should be in reverse chronological order")
			}
		}
	})
}

func TestHistoryService_GetRecentEntries_NoJournal(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	// Should return nil without error when no journal exists
	entries, err := svc.GetRecentEntries(10)
	if err != nil {
		t.Fatalf("GetRecentEntries failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected empty entries, got %d entries", len(entries))
	}
}

func TestHistoryService_LoadLastApply(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	// Create first service and record apply
	svc1, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	svc1.RecordApply("test-profile", "snapshot-abc", true)

	// Create new service instance - should load last apply from history
	svc2, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	lastApply := svc2.GetLastApply()
	if lastApply == nil {
		t.Fatal("lastApply should be loaded from history")
	}
	if lastApply.ProfileID != "test-profile" {
		t.Errorf("ProfileID = %s, want test-profile", lastApply.ProfileID)
	}
}

func TestHistoryService_GetLastApply_InitiallyNil(t *testing.T) {
	tmpDir := t.TempDir()
	logger := zap.NewNop()

	svc, err := NewHistoryService(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewHistoryService failed: %v", err)
	}

	lastApply := svc.GetLastApply()
	if lastApply != nil {
		t.Error("lastApply should be nil initially")
	}
}

func TestHistoryEntry(t *testing.T) {
	entry := &HistoryEntry{
		Timestamp:  time.Now(),
		Action:     "apply",
		ProfileID:  "test-profile",
		SnapshotID: "snapshot-123",
		Success:    true,
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	if entry.Action != "apply" {
		t.Error("Action should be 'apply'")
	}
	if entry.ProfileID != "test-profile" {
		t.Error("ProfileID should be 'test-profile'")
	}
}
