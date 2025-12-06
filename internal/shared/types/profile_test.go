package types

import (
	"testing"
)

func TestProfileToMeta(t *testing.T) {
	profile := &Profile{
		ID:          "test-profile",
		Name:        "Test Profile",
		Description: "A test profile for unit testing",
		RiskLevel:   "low",
		Sysctl: map[string]interface{}{
			"net.core.default_qdisc": "fq",
		},
	}

	meta := profile.ToMeta()

	if meta.ID != profile.ID {
		t.Errorf("Meta.ID = %q, want %q", meta.ID, profile.ID)
	}

	if meta.Name != profile.Name {
		t.Errorf("Meta.Name = %q, want %q", meta.Name, profile.Name)
	}

	if meta.Description != profile.Description {
		t.Errorf("Meta.Description = %q, want %q", meta.Description, profile.Description)
	}

	if meta.RiskLevel != profile.RiskLevel {
		t.Errorf("Meta.RiskLevel = %q, want %q", meta.RiskLevel, profile.RiskLevel)
	}
}

func TestProfileWithQdisc(t *testing.T) {
	profile := &Profile{
		ID:   "bbr-fq",
		Name: "BBR with FQ",
		Qdisc: &QdiscConfig{
			Type:       "fq",
			Interfaces: "default-route",
		},
	}

	if profile.Qdisc == nil {
		t.Fatal("Qdisc should not be nil")
	}

	if profile.Qdisc.Type != "fq" {
		t.Errorf("Qdisc.Type = %q, want %q", profile.Qdisc.Type, "fq")
	}

	if profile.Qdisc.Interfaces != "default-route" {
		t.Errorf("Qdisc.Interfaces = %q, want %q", profile.Qdisc.Interfaces, "default-route")
	}
}
