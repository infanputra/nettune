package adapter

import "go.uber.org/zap"

// SystemAdapter aggregates all system adapters
type SystemAdapter struct {
	Sysctl  *SysctlManager
	Qdisc   *QdiscManager
	Systemd *SystemdManager
	SysInfo *SystemInfoManager
	logger  *zap.Logger
}

// NewSystemAdapter creates a new SystemAdapter with all managers
func NewSystemAdapter(logger *zap.Logger) *SystemAdapter {
	return &SystemAdapter{
		Sysctl:  NewSysctlManager(logger),
		Qdisc:   NewQdiscManager(logger),
		Systemd: NewSystemdManager(logger),
		SysInfo: NewSystemInfoManager(logger),
		logger:  logger,
	}
}
