package config

import (
	"os"
	"strings"
	"testing"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	if cfg.Listen != "0.0.0.0:9876" {
		t.Errorf("Listen = %q, want %q", cfg.Listen, "0.0.0.0:9876")
	}

	if cfg.ReadTimeout != 30 {
		t.Errorf("ReadTimeout = %d, want %d", cfg.ReadTimeout, 30)
	}

	if cfg.WriteTimeout != 60 {
		t.Errorf("WriteTimeout = %d, want %d", cfg.WriteTimeout, 60)
	}

	if cfg.MaxBodyBytes != 100*1024*1024 {
		t.Errorf("MaxBodyBytes = %d, want %d", cfg.MaxBodyBytes, 100*1024*1024)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()

	if cfg.Server != "http://127.0.0.1:9876" {
		t.Errorf("Server = %q, want %q", cfg.Server, "http://127.0.0.1:9876")
	}

	if cfg.Timeout != 60 {
		t.Errorf("Timeout = %d, want %d", cfg.Timeout, 60)
	}
}

func TestGetStateDir(t *testing.T) {
	// Clear env vars for predictable testing
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("HOME", origHome)
	}()

	// Test with XDG_CONFIG_HOME set
	os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	os.Setenv("HOME", "/home/testuser")

	stateDir := GetDefaultStateDir()
	if !strings.Contains(stateDir, "nettune") {
		t.Errorf("StateDir should contain 'nettune', got %q", stateDir)
	}
}

func TestServerConfigGetDirs(t *testing.T) {
	cfg := &ServerConfig{
		StateDir: "/tmp/nettune-test",
	}

	profilesDir := cfg.GetProfilesDir()
	if profilesDir != "/tmp/nettune-test/profiles" {
		t.Errorf("ProfilesDir = %q, want %q", profilesDir, "/tmp/nettune-test/profiles")
	}

	snapshotsDir := cfg.GetSnapshotsDir()
	if snapshotsDir != "/tmp/nettune-test/snapshots" {
		t.Errorf("SnapshotsDir = %q, want %q", snapshotsDir, "/tmp/nettune-test/snapshots")
	}

	historyDir := cfg.GetHistoryDir()
	if historyDir != "/tmp/nettune-test/history" {
		t.Errorf("HistoryDir = %q, want %q", historyDir, "/tmp/nettune-test/history")
	}
}
