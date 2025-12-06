package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// SystemdManager handles systemd operations
type SystemdManager struct {
	logger *zap.Logger
}

// NewSystemdManager creates a new SystemdManager
func NewSystemdManager(logger *zap.Logger) *SystemdManager {
	return &SystemdManager{logger: logger}
}

// IsActive checks if a systemd unit is active
func (m *SystemdManager) IsActive(unit string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", unit)
	output, err := cmd.Output()
	if err != nil {
		// Non-zero exit code means not active
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "active", nil
}

// IsEnabled checks if a systemd unit is enabled
func (m *SystemdManager) IsEnabled(unit string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", unit)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(output)) == "enabled", nil
}

// Enable enables a systemd unit
func (m *SystemdManager) Enable(unit string) error {
	cmd := exec.Command("systemctl", "enable", unit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to enable %s: %w\noutput: %s", unit, err, string(output))
	}
	m.logger.Info("enabled systemd unit", zap.String("unit", unit))
	return nil
}

// Disable disables a systemd unit
func (m *SystemdManager) Disable(unit string) error {
	cmd := exec.Command("systemctl", "disable", unit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to disable %s: %w\noutput: %s", unit, err, string(output))
	}
	m.logger.Info("disabled systemd unit", zap.String("unit", unit))
	return nil
}

// Start starts a systemd unit
func (m *SystemdManager) Start(unit string) error {
	cmd := exec.Command("systemctl", "start", unit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start %s: %w\noutput: %s", unit, err, string(output))
	}
	m.logger.Info("started systemd unit", zap.String("unit", unit))
	return nil
}

// Stop stops a systemd unit
func (m *SystemdManager) Stop(unit string) error {
	cmd := exec.Command("systemctl", "stop", unit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop %s: %w\noutput: %s", unit, err, string(output))
	}
	m.logger.Info("stopped systemd unit", zap.String("unit", unit))
	return nil
}

// Restart restarts a systemd unit
func (m *SystemdManager) Restart(unit string) error {
	cmd := exec.Command("systemctl", "restart", unit)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart %s: %w\noutput: %s", unit, err, string(output))
	}
	m.logger.Info("restarted systemd unit", zap.String("unit", unit))
	return nil
}

// DaemonReload runs systemctl daemon-reload
func (m *SystemdManager) DaemonReload() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to reload daemon: %w\noutput: %s", err, string(output))
	}
	m.logger.Info("reloaded systemd daemon")
	return nil
}

// CreateUnit creates a systemd unit file
func (m *SystemdManager) CreateUnit(name, content string) error {
	// Determine unit path
	unitPath := filepath.Join("/etc/systemd/system", name)

	// Write unit file atomically
	tmpPath := unitPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write unit file: %w", err)
	}

	if err := os.Rename(tmpPath, unitPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename unit file: %w", err)
	}

	// Reload daemon
	if err := m.DaemonReload(); err != nil {
		return err
	}

	m.logger.Info("created systemd unit", zap.String("name", name), zap.String("path", unitPath))
	return nil
}

// RemoveUnit removes a systemd unit file
func (m *SystemdManager) RemoveUnit(name string) error {
	unitPath := filepath.Join("/etc/systemd/system", name)

	// Stop and disable first
	m.Stop(name)
	m.Disable(name)

	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}

	// Reload daemon
	if err := m.DaemonReload(); err != nil {
		return err
	}

	m.logger.Info("removed systemd unit", zap.String("name", name))
	return nil
}

// UnitExists checks if a unit file exists
func (m *SystemdManager) UnitExists(name string) bool {
	unitPath := filepath.Join("/etc/systemd/system", name)
	_, err := os.Stat(unitPath)
	return err == nil
}

// ReadUnit reads a unit file content
func (m *SystemdManager) ReadUnit(name string) (string, error) {
	unitPath := filepath.Join("/etc/systemd/system", name)
	data, err := os.ReadFile(unitPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// IsAvailable checks if systemd is available
func (m *SystemdManager) IsAvailable() bool {
	cmd := exec.Command("systemctl", "--version")
	return cmd.Run() == nil
}

// NettuneQdiscServiceName is the name of the nettune qdisc service
const NettuneQdiscServiceName = "nettune-qdisc.service"

// NettuneQdiscScriptPath is the path to the qdisc setup script
const NettuneQdiscScriptPath = "/usr/local/bin/nettune-qdisc-setup.sh"

// GenerateQdiscServiceUnit generates the qdisc persistence service unit
func GenerateQdiscServiceUnit() string {
	return `[Unit]
Description=Nettune Qdisc Persistence
After=network.target

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/nettune-qdisc-setup.sh
ExecStop=/bin/true

[Install]
WantedBy=multi-user.target
`
}

// GenerateQdiscSetupScript generates the qdisc setup script
func GenerateQdiscSetupScript(qdiscType string, iface string) string {
	if iface == "" {
		return fmt.Sprintf(`#!/bin/bash
# Managed by nettune - DO NOT EDIT
DEFAULT_IFACE=$(ip route | grep default | awk '{print $5}' | head -n1)
if [ -n "$DEFAULT_IFACE" ]; then
    tc qdisc replace dev "$DEFAULT_IFACE" root %s
fi
`, qdiscType)
	}
	return fmt.Sprintf(`#!/bin/bash
# Managed by nettune - DO NOT EDIT
tc qdisc replace dev %s root %s
`, iface, qdiscType)
}
