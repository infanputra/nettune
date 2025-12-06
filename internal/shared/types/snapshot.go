package types

import "time"

// Snapshot represents a system state snapshot for rollback
type Snapshot struct {
	ID        string                 `json:"id"`
	CreatedAt time.Time              `json:"created_at"`
	State     *SystemState           `json:"state"`
	Backups   map[string]string      `json:"backups"` // file path -> backup content
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SystemState represents the current system configuration state
type SystemState struct {
	Sysctl       map[string]string     `json:"sysctl"`
	Qdisc        map[string]*QdiscInfo `json:"qdisc"`         // interface name -> qdisc info
	SystemdUnits map[string]bool       `json:"systemd_units"` // unit name -> is active
	FileHashes   map[string]string     `json:"file_hashes"`
}

// QdiscInfo represents qdisc information for an interface
type QdiscInfo struct {
	Type   string                 `json:"type"`
	Handle string                 `json:"handle"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// SnapshotMeta represents snapshot metadata for listing
type SnapshotMeta struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"` // snapshot size in bytes
}

// ToMeta converts a Snapshot to SnapshotMeta
func (s *Snapshot) ToMeta() *SnapshotMeta {
	return &SnapshotMeta{
		ID:        s.ID,
		CreatedAt: s.CreatedAt,
	}
}
