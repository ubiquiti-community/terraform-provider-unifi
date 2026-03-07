// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
)

type GlobalSwitch struct {
	BaseSetting

	AclDeviceIsolation             []string                            `json:"acl_device_isolation,omitempty"`
	AclL3Isolation                 []SettingGlobalSwitchAclL3Isolation `json:"acl_l3_isolation,omitempty"`
	DHCPSnoop                      bool                                `json:"dhcp_snoop"`
	Dot1XFallbackNetworkID         string                              `json:"dot1x_fallback_networkconf_id,omitempty"` // [\d\w]+|
	Dot1XPortctrlEnabled           bool                                `json:"dot1x_portctrl_enabled"`
	FloodKnownProtocols            bool                                `json:"flood_known_protocols"`
	FlowctrlEnabled                bool                                `json:"flowctrl_enabled"`
	ForwardUnknownMcastRouterPorts bool                                `json:"forward_unknown_mcast_router_ports"`
	JumboframeEnabled              bool                                `json:"jumboframe_enabled"`
	RADIUSProfileID                string                              `json:"radiusprofile_id,omitempty"`
	StpVersion                     string                              `json:"stp_version,omitempty"`       // stp|rstp|disabled
	SwitchExclusions               []string                            `json:"switch_exclusions,omitempty"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
}

func (dst *GlobalSwitch) UnmarshalJSON(b []byte) error {
	type Alias GlobalSwitch
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	// First unmarshal base setting
	if err := json.Unmarshal(b, &dst.BaseSetting); err != nil {
		return fmt.Errorf("unable to unmarshal base setting: %w", err)
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}

type SettingGlobalSwitchAclL3Isolation struct {
	DestinationNetworks []string `json:"destination_networks,omitempty"`
	SourceNetwork       string   `json:"source_network,omitempty"`
}

func (dst *SettingGlobalSwitchAclL3Isolation) UnmarshalJSON(b []byte) error {
	type Alias SettingGlobalSwitchAclL3Isolation
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}

	return nil
}
