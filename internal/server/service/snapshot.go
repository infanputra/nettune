package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jtsang4/nettune/internal/server/adapter"
	"github.com/jtsang4/nettune/internal/shared/types"
	"github.com/jtsang4/nettune/internal/shared/utils"
	"go.uber.org/zap"
)

// SnapshotService manages system state snapshots
type SnapshotService struct {
	snapshotsDir string
	adapter      *adapter.SystemAdapter
	mu           sync.Mutex
	logger       *zap.Logger
}

// NewSnapshotService creates a new SnapshotService
func NewSnapshotService(snapshotsDir string, adapter *adapter.SystemAdapter, logger *zap.Logger) (*SnapshotService, error) {
	s := &SnapshotService{
		snapshotsDir: snapshotsDir,
		adapter:      adapter,
		logger:       logger,
	}

	// Ensure snapshots directory exists
	if err := utils.EnsureDir(snapshotsDir); err != nil {
		return nil, fmt.Errorf("failed to create snapshots directory: %w", err)
	}

	return s, nil
}

// Create creates a new snapshot of current system state
func (s *SnapshotService) Create() (*types.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate snapshot ID
	timestamp := time.Now().UTC()
	snapshotID := fmt.Sprintf("%s_%s",
		timestamp.Format("2006-01-02T15-04-05Z"),
		utils.HashString(fmt.Sprintf("%d", timestamp.UnixNano()))[:8])

	// Create snapshot directory
	snapshotDir := filepath.Join(s.snapshotsDir, snapshotID)
	if err := utils.EnsureDir(snapshotDir); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	// Collect current state
	state, err := s.collectCurrentState()
	if err != nil {
		return nil, fmt.Errorf("failed to collect current state: %w", err)
	}

	// Create backups of managed files
	backups, err := s.createBackups(snapshotDir)
	if err != nil {
		s.logger.Warn("failed to create some backups", zap.Error(err))
	}

	snapshot := &types.Snapshot{
		ID:        snapshotID,
		CreatedAt: timestamp,
		State:     state,
		Backups:   backups,
		Metadata: map[string]interface{}{
			"created_by": "nettune",
		},
	}

	// Save snapshot metadata
	if err := s.saveSnapshot(snapshot); err != nil {
		return nil, err
	}

	s.logger.Info("created snapshot", zap.String("id", snapshotID))
	return snapshot, nil
}

// Get returns a snapshot by ID
func (s *SnapshotService) Get(id string) (*types.Snapshot, error) {
	snapshotDir := filepath.Join(s.snapshotsDir, id)
	stateFile := filepath.Join(snapshotDir, "state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, types.ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	var snapshot types.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot: %w", err)
	}

	return &snapshot, nil
}

// List returns all snapshot metadata
func (s *SnapshotService) List() ([]*types.SnapshotMeta, error) {
	entries, err := os.ReadDir(s.snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []*types.SnapshotMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		snapshot, err := s.Get(entry.Name())
		if err != nil {
			s.logger.Warn("failed to load snapshot",
				zap.String("id", entry.Name()),
				zap.Error(err))
			continue
		}

		meta := snapshot.ToMeta()
		// Calculate size
		meta.Size = s.calculateSnapshotSize(filepath.Join(s.snapshotsDir, entry.Name()))
		snapshots = append(snapshots, meta)
	}

	// Sort by creation time (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

// GetLatest returns the most recent snapshot
func (s *SnapshotService) GetLatest() (*types.Snapshot, error) {
	snapshots, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, types.ErrSnapshotNotFound
	}
	return s.Get(snapshots[0].ID)
}

// Delete removes a snapshot
func (s *SnapshotService) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotDir := filepath.Join(s.snapshotsDir, id)
	if !utils.DirExists(snapshotDir) {
		return types.ErrSnapshotNotFound
	}

	if err := os.RemoveAll(snapshotDir); err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	s.logger.Info("deleted snapshot", zap.String("id", id))
	return nil
}

// GetCurrentState returns the current system state without creating a snapshot
func (s *SnapshotService) GetCurrentState() (*types.SystemState, error) {
	return s.collectCurrentState()
}

// collectCurrentState collects the current system state
func (s *SnapshotService) collectCurrentState() (*types.SystemState, error) {
	state := &types.SystemState{
		Sysctl:       make(map[string]string),
		Qdisc:        make(map[string]*types.QdiscInfo),
		SystemdUnits: make(map[string]bool),
		FileHashes:   make(map[string]string),
	}

	// Collect sysctl values
	sysctlKeys := adapter.NetworkSysctlKeys()
	sysctlValues, err := s.adapter.Sysctl.GetMultiple(sysctlKeys)
	if err != nil {
		s.logger.Warn("failed to collect some sysctl values", zap.Error(err))
	}
	state.Sysctl = sysctlValues

	// Collect qdisc info
	qdiscInfo, err := s.adapter.Qdisc.GetAll()
	if err != nil {
		s.logger.Warn("failed to collect qdisc info", zap.Error(err))
	}
	state.Qdisc = qdiscInfo

	// Check systemd units
	units := []string{adapter.NettuneQdiscServiceName}
	for _, unit := range units {
		active, _ := s.adapter.Systemd.IsActive(unit)
		state.SystemdUnits[unit] = active
	}

	// Collect file hashes
	managedFiles := []string{
		"/etc/sysctl.d/99-nettune.conf",
	}
	for _, file := range managedFiles {
		if utils.FileExists(file) {
			hash, err := utils.HashFile(file)
			if err == nil {
				state.FileHashes[file] = hash
			}
		}
	}

	return state, nil
}

// createBackups creates backups of managed files
func (s *SnapshotService) createBackups(snapshotDir string) (map[string]string, error) {
	backups := make(map[string]string)
	backupsDir := filepath.Join(snapshotDir, "backups")
	if err := utils.EnsureDir(backupsDir); err != nil {
		return nil, err
	}

	managedFiles := []string{
		"/etc/sysctl.d/99-nettune.conf",
		adapter.NettuneQdiscScriptPath,
	}

	for _, file := range managedFiles {
		if !utils.FileExists(file) {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			s.logger.Warn("failed to read file for backup",
				zap.String("file", file),
				zap.Error(err))
			continue
		}

		// Store content in backups map
		backups[file] = string(content)

		// Also save to backup file
		backupName := strings.ReplaceAll(file, "/", "_")
		backupPath := filepath.Join(backupsDir, backupName)
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			s.logger.Warn("failed to write backup file",
				zap.String("file", file),
				zap.Error(err))
		}
	}

	return backups, nil
}

// saveSnapshot saves snapshot metadata to disk
func (s *SnapshotService) saveSnapshot(snapshot *types.Snapshot) error {
	snapshotDir := filepath.Join(s.snapshotsDir, snapshot.ID)
	stateFile := filepath.Join(snapshotDir, "state.json")

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := utils.AtomicWriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshot: %w", err)
	}

	return nil
}

// calculateSnapshotSize calculates the total size of a snapshot directory
func (s *SnapshotService) calculateSnapshotSize(dir string) int64 {
	var size int64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}
