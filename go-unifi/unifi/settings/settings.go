package settings

import (
	"encoding/json"
	"fmt"
)

// Setting is the interface that all setting types must implement.
type Setting interface {
	GetKey() string
	SetKey(key string)
}

// BaseSetting contains common fields for all settings.
type BaseSetting struct {
	ID       string `json:"_id,omitempty"`
	SiteID   string `json:"site_id,omitempty"`
	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`
	Key      string `json:"key"`
}

// GetKey returns the setting key.
func (b *BaseSetting) GetKey() string {
	return b.Key
}

// SetKey sets the setting key.
func (b *BaseSetting) SetKey(key string) {
	b.Key = key
}

// RawSetting represents a generic setting when the specific type is unknown.
type RawSetting struct {
	BaseSetting
	Data map[string]any `json:"-"`
}

// UnmarshalJSON implements custom unmarshaling for RawSetting.
func (r *RawSetting) UnmarshalJSON(b []byte) error {
	// First unmarshal into BaseSetting
	if err := json.Unmarshal(b, &r.BaseSetting); err != nil {
		return err
	}

	// Then unmarshal the full data
	if err := json.Unmarshal(b, &r.Data); err != nil {
		return err
	}

	return nil
}

// MarshalJSON implements custom marshaling for RawSetting.
func (r *RawSetting) MarshalJSON() ([]byte, error) {
	// Merge base setting fields into data map
	data := make(map[string]any)

	for k, v := range r.Data {
		data[k] = v
	}

	// Override with base setting fields
	baseBytes, err := json.Marshal(r.BaseSetting)
	if err != nil {
		return nil, err
	}

	var baseData map[string]any
	if err := json.Unmarshal(baseBytes, &baseData); err != nil {
		return nil, err
	}

	for k, v := range baseData {
		data[k] = v
	}

	return json.Marshal(data)
}

// GetSettingKey returns the key name for a specific setting type
// This is used internally to determine which endpoint to call.
func GetSettingKey(setting Setting) (string, error) {
	switch s := setting.(type) {
	case *AutoSpeedtest:
		return "auto_speedtest", nil
	case *Baresip:
		return "baresip", nil
	case *Broadcast:
		return "broadcast", nil
	case *Connectivity:
		return "connectivity", nil
	case *Country:
		return "country", nil
	case *Dashboard:
		return "dashboard", nil
	case *Doh:
		return "doh", nil
	case *Dpi:
		return "dpi", nil
	case *ElementAdopt:
		return "element_adopt", nil
	case *EtherLighting:
		return "ether_lighting", nil
	case *EvaluationScore:
		return "evaluation_score", nil
	case *GlobalAp:
		return "global_ap", nil
	case *GlobalNat:
		return "global_nat", nil
	case *GlobalSwitch:
		return "global_switch", nil
	case *GuestAccess:
		return "guest_access", nil
	case *Ips:
		return "ips", nil
	case *Lcm:
		return "lcm", nil
	case *Locale:
		return "locale", nil
	case *MagicSiteToSiteVpn:
		return "magic_site_to_site_vpn", nil
	case *Mdns:
		return "mdns", nil
	case *Mgmt:
		return "mgmt", nil
	case *Netflow:
		return "netflow", nil
	case *NetworkOptimization:
		return "network_optimization", nil
	case *Ntp:
		return "ntp", nil
	case *Porta:
		return "porta", nil
	case *RadioAi:
		return "radio_ai", nil
	case *Radius:
		return "radius", nil
	case *RoamingAssistant:
		return "roaming_assistant", nil
	case *Rsyslogd:
		return "rsyslogd", nil
	case *Snmp:
		return "snmp", nil
	case *SslInspection:
		return "ssl_inspection", nil
	case *SuperCloudaccess:
		return "super_cloudaccess", nil
	case *SuperEvents:
		return "super_events", nil
	case *SuperFwupdate:
		return "super_fwupdate", nil
	case *SuperIdentity:
		return "super_identity", nil
	case *SuperMail:
		return "super_mail", nil
	case *SuperMgmt:
		return "super_mgmt", nil
	case *SuperSdn:
		return "super_sdn", nil
	case *SuperSmtp:
		return "super_smtp", nil
	case *Teleport:
		return "teleport", nil
	case *TrafficFlow:
		return "traffic_flow", nil
	case *Usg:
		return "usg", nil
	case *Usw:
		return "usw", nil
	case *RawSetting:
		// For raw settings, use the key from the data
		if s.Key != "" {
			return s.Key, nil
		}
		return "", fmt.Errorf("raw setting has no key")
	default:
		return "", fmt.Errorf("unknown setting type: %T", setting)
	}
}
