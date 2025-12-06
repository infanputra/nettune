// Package config provides configuration management
package config

import (
	"os"
	"path/filepath"
)

// ServerConfig represents server mode configuration
type ServerConfig struct {
	APIKey          string `mapstructure:"api-key"`
	Listen          string `mapstructure:"listen"`
	StateDir        string `mapstructure:"state-dir"`
	ReadTimeout     int    `mapstructure:"read-timeout"`
	WriteTimeout    int    `mapstructure:"write-timeout"`
	MaxBodyBytes    int64  `mapstructure:"max-body-bytes"`
	AllowUnsafeHTTP bool   `mapstructure:"allow-unsafe-http"`
}

// ClientConfig represents client mode configuration
type ClientConfig struct {
	APIKey  string `mapstructure:"api-key"`
	Server  string `mapstructure:"server"`
	MCPName string `mapstructure:"mcp-name"`
	Timeout int    `mapstructure:"timeout"`
}

// DefaultServerConfig returns the default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Listen:          "0.0.0.0:9876",
		StateDir:        GetDefaultStateDir(),
		ReadTimeout:     30,
		WriteTimeout:    60,
		MaxBodyBytes:    100 * 1024 * 1024, // 100MB
		AllowUnsafeHTTP: true,
	}
}

// DefaultClientConfig returns the default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Server:  "http://127.0.0.1:9876",
		MCPName: "nettune",
		Timeout: 60,
	}
}

// GetDefaultStateDir returns the default state directory
func GetDefaultStateDir() string {
	// Priority: $XDG_CONFIG_HOME/nettune, then ~/.config/nettune
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "nettune")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback for running as root or when home is not available
		return "/var/lib/nettune"
	}

	return filepath.Join(homeDir, ".config", "nettune")
}

// GetProfilesDir returns the profiles directory path
func (c *ServerConfig) GetProfilesDir() string {
	return filepath.Join(c.StateDir, "profiles")
}

// GetSnapshotsDir returns the snapshots directory path
func (c *ServerConfig) GetSnapshotsDir() string {
	return filepath.Join(c.StateDir, "snapshots")
}

// GetHistoryDir returns the history directory path
func (c *ServerConfig) GetHistoryDir() string {
	return filepath.Join(c.StateDir, "history")
}
