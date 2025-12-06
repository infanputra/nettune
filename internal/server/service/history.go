package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jtsang4/nettune/internal/shared/types"
	"github.com/jtsang4/nettune/internal/shared/utils"
	"go.uber.org/zap"
)

// HistoryService manages operation history and audit logs
type HistoryService struct {
	historyDir string
	mu         sync.Mutex
	logger     *zap.Logger
	lastApply  *types.LastApplyInfo
}

// HistoryEntry represents a single history entry
type HistoryEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Action     string                 `json:"action"` // "apply", "rollback", "snapshot"
	ProfileID  string                 `json:"profile_id,omitempty"`
	SnapshotID string                 `json:"snapshot_id,omitempty"`
	Success    bool                   `json:"success"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// NewHistoryService creates a new HistoryService
func NewHistoryService(historyDir string, logger *zap.Logger) (*HistoryService, error) {
	s := &HistoryService{
		historyDir: historyDir,
		logger:     logger,
	}

	// Ensure history directory exists
	if err := utils.EnsureDir(historyDir); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// Load last apply info
	s.loadLastApply()

	return s, nil
}

// RecordApply records a profile apply operation
func (s *HistoryService) RecordApply(profileID, snapshotID string, success bool) {
	entry := &HistoryEntry{
		Timestamp:  time.Now(),
		Action:     "apply",
		ProfileID:  profileID,
		SnapshotID: snapshotID,
		Success:    success,
	}

	if err := s.appendEntry(entry); err != nil {
		s.logger.Error("failed to record apply", zap.Error(err))
	}

	if success {
		s.mu.Lock()
		s.lastApply = &types.LastApplyInfo{
			ProfileID: profileID,
			AppliedAt: entry.Timestamp,
			Success:   success,
		}
		s.mu.Unlock()
	}
}

// RecordRollback records a rollback operation
func (s *HistoryService) RecordRollback(snapshotID string, success bool) {
	entry := &HistoryEntry{
		Timestamp:  time.Now(),
		Action:     "rollback",
		SnapshotID: snapshotID,
		Success:    success,
	}

	if err := s.appendEntry(entry); err != nil {
		s.logger.Error("failed to record rollback", zap.Error(err))
	}
}

// RecordSnapshot records a snapshot creation
func (s *HistoryService) RecordSnapshot(snapshotID string) {
	entry := &HistoryEntry{
		Timestamp:  time.Now(),
		Action:     "snapshot",
		SnapshotID: snapshotID,
		Success:    true,
	}

	if err := s.appendEntry(entry); err != nil {
		s.logger.Error("failed to record snapshot", zap.Error(err))
	}
}

// GetLastApply returns the last apply info
func (s *HistoryService) GetLastApply() *types.LastApplyInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastApply
}

// GetRecentEntries returns recent history entries
func (s *HistoryService) GetRecentEntries(limit int) ([]*HistoryEntry, error) {
	journalPath := s.getJournalPath()
	file, err := os.Open(journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []*HistoryEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, &entry)
	}

	// Return last N entries
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Reverse order (newest first)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, nil
}

// appendEntry appends a history entry to the journal
func (s *HistoryService) appendEntry(entry *HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	journalPath := s.getJournalPath()
	file, err := os.OpenFile(journalPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(data) + "\n")
	return err
}

// getJournalPath returns the path to the journal file
func (s *HistoryService) getJournalPath() string {
	return filepath.Join(s.historyDir, "journal.jsonl")
}

// loadLastApply loads the last apply info from history
func (s *HistoryService) loadLastApply() {
	entries, err := s.GetRecentEntries(100)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.Action == "apply" && entry.Success {
			s.lastApply = &types.LastApplyInfo{
				ProfileID: entry.ProfileID,
				AppliedAt: entry.Timestamp,
				Success:   entry.Success,
			}
			return
		}
	}
}
