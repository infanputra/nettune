// Package adapter provides system adapters for Linux operations
package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// SysctlManager handles sysctl operations
type SysctlManager struct {
	logger *zap.Logger
}

// NewSysctlManager creates a new SysctlManager
func NewSysctlManager(logger *zap.Logger) *SysctlManager {
	return &SysctlManager{logger: logger}
}

// Get reads a sysctl value
func (m *SysctlManager) Get(key string) (string, error) {
	// Convert dot notation to path: net.core.rmem_max -> /proc/sys/net/core/rmem_max
	path := m.keyToPath(key)

	data, err := os.ReadFile(path)
	if err != nil {
		// Fallback to sysctl command
		return m.getViaSysctl(key)
	}

	return strings.TrimSpace(string(data)), nil
}

// Set writes a sysctl value
func (m *SysctlManager) Set(key, value string) error {
	path := m.keyToPath(key)

	// Try writing to /proc/sys first
	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		m.logger.Debug("failed to write to proc, falling back to sysctl command",
			zap.String("key", key),
			zap.Error(err))
		// Fallback to sysctl command
		return m.setViaSysctl(key, value)
	}

	m.logger.Debug("sysctl set successfully",
		zap.String("key", key),
		zap.String("value", value))
	return nil
}

// GetMultiple reads multiple sysctl values
func (m *SysctlManager) GetMultiple(keys []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		value, err := m.Get(key)
		if err != nil {
			m.logger.Warn("failed to read sysctl", zap.String("key", key), zap.Error(err))
			continue
		}
		result[key] = value
	}
	return result, nil
}

// SetMultiple writes multiple sysctl values
func (m *SysctlManager) SetMultiple(values map[string]string) error {
	var errors []string
	for key, value := range values {
		if err := m.Set(key, value); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", key, err))
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("failed to set some sysctl values: %s", strings.Join(errors, "; "))
	}
	return nil
}

// WriteToFile writes sysctl configuration to a file
func (m *SysctlManager) WriteToFile(path string, kvs map[string]string) error {
	var lines []string
	lines = append(lines, "# Managed by nettune - DO NOT EDIT")
	lines = append(lines, "")

	for key, value := range kvs {
		lines = append(lines, fmt.Sprintf("%s = %s", key, value))
	}

	content := strings.Join(lines, "\n") + "\n"

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Atomic write
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, path); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	m.logger.Info("wrote sysctl configuration file", zap.String("path", path))
	return nil
}

// LoadFromFile loads sysctl settings from a file using sysctl -p
func (m *SysctlManager) LoadFromFile(path string) error {
	cmd := exec.Command("sysctl", "-p", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load sysctl from %s: %w\noutput: %s", path, err, string(output))
	}
	m.logger.Info("loaded sysctl configuration from file", zap.String("path", path))
	return nil
}

// ReadFile reads a sysctl configuration file content
func (m *SysctlManager) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// keyToPath converts sysctl key to /proc/sys path
func (m *SysctlManager) keyToPath(key string) string {
	// Replace dots with slashes
	pathKey := strings.ReplaceAll(key, ".", "/")
	return filepath.Join("/proc/sys", pathKey)
}

// getViaSysctl reads sysctl value using sysctl command
func (m *SysctlManager) getViaSysctl(key string) (string, error) {
	cmd := exec.Command("sysctl", "-n", key)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("sysctl -n %s failed: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// setViaSysctl writes sysctl value using sysctl command
func (m *SysctlManager) setViaSysctl(key, value string) error {
	cmd := exec.Command("sysctl", "-w", fmt.Sprintf("%s=%s", key, value))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sysctl -w %s=%s failed: %w\noutput: %s", key, value, err, string(output))
	}
	return nil
}

// NetworkSysctlKeys returns a list of common network-related sysctl keys
func NetworkSysctlKeys() []string {
	return []string{
		"net.core.default_qdisc",
		"net.ipv4.tcp_congestion_control",
		"net.core.rmem_max",
		"net.core.wmem_max",
		"net.core.rmem_default",
		"net.core.wmem_default",
		"net.ipv4.tcp_rmem",
		"net.ipv4.tcp_wmem",
		"net.ipv4.tcp_mtu_probing",
		"net.ipv4.tcp_slow_start_after_idle",
		"net.ipv4.tcp_no_metrics_save",
		"net.ipv4.tcp_timestamps",
		"net.ipv4.tcp_sack",
		"net.ipv4.tcp_window_scaling",
	}
}
