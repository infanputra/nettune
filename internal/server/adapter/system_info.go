package adapter

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jtsang4/nettune/internal/shared/types"
	"go.uber.org/zap"
)

// SystemInfoManager handles system information collection
type SystemInfoManager struct {
	logger *zap.Logger
}

// NewSystemInfoManager creates a new SystemInfoManager
func NewSystemInfoManager(logger *zap.Logger) *SystemInfoManager {
	return &SystemInfoManager{logger: logger}
}

// GetServerInfo collects server information
func (m *SystemInfoManager) GetServerInfo() (*types.ServerInfo, error) {
	info := &types.ServerInfo{
		Dependencies: make(map[string]string),
	}

	// Hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	// Kernel version
	info.KernelVersion = m.getKernelVersion()

	// Distribution
	info.Distribution = m.getDistribution()

	// Current congestion control
	info.CongestionControl = m.getCurrentCongestionControl()

	// Default qdisc
	info.DefaultQdisc = m.getDefaultQdisc()

	// Available congestion control algorithms
	info.AvailableCCs = m.getAvailableCCs()

	// Default interface and MTU
	qdiscMgr := NewQdiscManager(m.logger)
	if iface, err := qdiscMgr.GetDefaultRouteInterface(); err == nil {
		info.DefaultInterface = iface
		if mtu, err := qdiscMgr.GetInterfaceMTU(iface); err == nil {
			info.InterfaceMTU = mtu
		}
		// Get interface stats
		info.InterfaceStats = m.getInterfaceStats(iface)
	}

	// Check dependencies
	info.Dependencies = m.checkDependencies()

	return info, nil
}

// getKernelVersion returns the kernel version
func (m *SystemInfoManager) getKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		// Fallback to uname
		cmd := exec.Command("uname", "-r")
		output, err := cmd.Output()
		if err != nil {
			return "unknown"
		}
		return strings.TrimSpace(string(output))
	}
	// Extract version from /proc/version
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		return parts[2]
	}
	return strings.TrimSpace(string(data))
}

// getDistribution returns the Linux distribution info
func (m *SystemInfoManager) getDistribution() string {
	if runtime.GOOS != "linux" {
		return runtime.GOOS
	}

	// Try /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var name, version string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			// Extract value between quotes
			value := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(value, "\"")
		}
		if strings.HasPrefix(line, "NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "VERSION=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		}
	}

	if name != "" {
		if version != "" {
			return name + " " + version
		}
		return name
	}
	return "Linux"
}

// getCurrentCongestionControl returns the current TCP congestion control algorithm
func (m *SystemInfoManager) getCurrentCongestionControl() string {
	data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_congestion_control")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// getDefaultQdisc returns the default qdisc
func (m *SystemInfoManager) getDefaultQdisc() string {
	data, err := os.ReadFile("/proc/sys/net/core/default_qdisc")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// getAvailableCCs returns available congestion control algorithms
func (m *SystemInfoManager) getAvailableCCs() []string {
	data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_available_congestion_control")
	if err != nil {
		return nil
	}
	return strings.Fields(string(data))
}

// getInterfaceStats returns interface statistics
func (m *SystemInfoManager) getInterfaceStats(iface string) *types.InterfaceStats {
	stats := &types.InterfaceStats{}

	// Read from /sys/class/net/<iface>/statistics/
	basePath := fmt.Sprintf("/sys/class/net/%s/statistics", iface)

	readStat := func(name string) int64 {
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", basePath, name))
		if err != nil {
			return 0
		}
		var value int64
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &value)
		return value
	}

	stats.RxPackets = readStat("rx_packets")
	stats.TxPackets = readStat("tx_packets")
	stats.RxDropped = readStat("rx_dropped")
	stats.TxDropped = readStat("tx_dropped")
	stats.RxErrors = readStat("rx_errors")
	stats.TxErrors = readStat("tx_errors")

	return stats
}

// checkDependencies checks for required system dependencies
func (m *SystemInfoManager) checkDependencies() map[string]string {
	deps := make(map[string]string)

	// Check tc
	if cmd := exec.Command("tc", "-V"); cmd.Run() == nil {
		output, _ := exec.Command("tc", "-V").Output()
		version := strings.TrimSpace(string(output))
		if idx := strings.Index(version, ","); idx > 0 {
			version = version[:idx]
		}
		deps["tc"] = version
	} else {
		deps["tc"] = "not found"
	}

	// Check systemctl
	if cmd := exec.Command("systemctl", "--version"); cmd.Run() == nil {
		output, _ := exec.Command("systemctl", "--version").Output()
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			deps["systemd"] = strings.TrimSpace(lines[0])
		}
	} else {
		deps["systemd"] = "not found"
	}

	// Check ip
	if cmd := exec.Command("ip", "-V"); cmd.Run() == nil {
		output, _ := exec.Command("ip", "-V").Output()
		deps["iproute2"] = strings.TrimSpace(string(output))
	} else {
		deps["iproute2"] = "not found"
	}

	// Check if BBR is available
	availableCCs := m.getAvailableCCs()
	hasBBR := false
	for _, cc := range availableCCs {
		if cc == "bbr" {
			hasBBR = true
			break
		}
	}
	if hasBBR {
		deps["bbr"] = "available"
	} else {
		deps["bbr"] = "not available (kernel module may need loading)"
	}

	return deps
}
