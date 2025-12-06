package version

import (
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}
}

func TestInfoString(t *testing.T) {
	info := GetInfo()
	str := info.String()

	if !strings.Contains(str, "nettune") {
		t.Error("String should contain 'nettune'")
	}

	if !strings.Contains(str, info.Version) {
		t.Errorf("String should contain version %q", info.Version)
	}
}

func TestInfoContainsVersion(t *testing.T) {
	info := GetInfo()

	if info.Version == "" {
		t.Error("Version should have a value")
	}

	// Version should be set even if it's the default "dev"
	if info.Version != "dev" && info.Version == "" {
		t.Error("Version should be 'dev' or a valid version string")
	}
}
