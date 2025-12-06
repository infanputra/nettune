package types

// Profile represents a configuration profile
type Profile struct {
	ID             string                 `json:"id" validate:"required,profile_id"`
	Name           string                 `json:"name" validate:"required"`
	Description    string                 `json:"description,omitempty"`
	RiskLevel      string                 `json:"risk_level" validate:"required,oneof=low medium high"`
	RequiresReboot bool                   `json:"requires_reboot"`
	Sysctl         map[string]interface{} `json:"sysctl,omitempty"`
	Qdisc          *QdiscConfig           `json:"qdisc,omitempty"`
	Systemd        *SystemdConfig         `json:"systemd,omitempty"`
}

// QdiscConfig represents qdisc configuration
type QdiscConfig struct {
	Type       string                 `json:"type" validate:"oneof=fq fq_codel cake pfifo_fast"`
	Interfaces string                 `json:"interfaces" validate:"oneof=default-route all"`
	Params     map[string]interface{} `json:"params,omitempty"`
}

// SystemdConfig represents systemd configuration
type SystemdConfig struct {
	EnsureQdiscService bool `json:"ensure_qdisc_service"`
}

// ProfileMeta represents profile metadata for listing
type ProfileMeta struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	RiskLevel      string `json:"risk_level"`
	RequiresReboot bool   `json:"requires_reboot"`
}

// ToMeta converts a Profile to ProfileMeta
func (p *Profile) ToMeta() *ProfileMeta {
	return &ProfileMeta{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		RiskLevel:      p.RiskLevel,
		RequiresReboot: p.RequiresReboot,
	}
}
