// Code generated from ace.jar fields *.json files
// DO NOT EDIT.

package unifi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ubiquiti-community/go-unifi/unifi/types"
)

// just to fix compile issues with the import.
var (
	_ context.Context
	_ fmt.Formatter
	_ json.Marshaler
	_ types.Number
	_ strconv.NumError
	_ strings.Builder
)

type Device struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	MAC string `json:"mac,omitempty"`

	Adopted                     bool                      `json:"adopted"`
	AfcEnabled                  bool                      `json:"afc_enabled,omitempty"`
	AtfEnabled                  bool                      `json:"atf_enabled,omitempty"`
	BandsteeringMode            string                    `json:"bandsteering_mode,omitempty"` // off|equal|prefer_5g
	BaresipAuthUser             string                    `json:"baresip_auth_user,omitempty"` // ^\+?[a-zA-Z0-9_.\-!~*'()]*
	BaresipEnabled              bool                      `json:"baresip_enabled,omitempty"`
	BaresipExtension            string                    `json:"baresip_extension,omitempty"` // ^\+?[a-zA-Z0-9_.\-!~*'()]*
	ConfigNetwork               *DeviceConfigNetwork      `json:"config_network,omitempty"`
	DPIEnabled                  bool                      `json:"dpi_enabled,omitempty"`
	Disabled                    bool                      `json:"disabled,omitempty"`
	Dot1XFallbackNetworkID      string                    `json:"dot1x_fallback_networkconf_id,omitempty"` // [\d\w]+|
	Dot1XPortctrlEnabled        bool                      `json:"dot1x_portctrl_enabled,omitempty"`
	EtherLighting               *DeviceEtherLighting      `json:"ether_lighting,omitempty"`
	EthernetOverrides           []DeviceEthernetOverrides `json:"ethernet_overrides,omitempty"`
	FanModeOverride             string                    `json:"fan_mode_override,omitempty"` // default|quiet
	FlowctrlEnabled             bool                      `json:"flowctrl_enabled,omitempty"`
	GatewayVrrpMode             string                    `json:"gateway_vrrp_mode,omitempty"`     // primary|secondary
	GatewayVrrpPriority         *int64                    `json:"gateway_vrrp_priority,omitempty"` // [1-9][0-9]|[1-9][0-9][0-9]
	GreenApEnabled              bool                      `json:"green_ap_enabled,omitempty"`
	HardwareOffload             bool                      `json:"hardware_offload,omitempty"`
	HeightInMeters              float64                   `json:"heightInMeters,omitempty"`
	Hostname                    string                    `json:"hostname,omitempty"` // .{1,128}
	IP                          string                    `json:"ip,omitempty"`
	InformIP                    string                    `json:"inform_ip,omitempty"`
	JumboframeEnabled           bool                      `json:"jumboframe_enabled,omitempty"`
	LcmBrightness               *int64                    `json:"lcm_brightness,omitempty"` // [1-9]|[1-9][0-9]|100
	LcmBrightnessOverride       bool                      `json:"lcm_brightness_override,omitempty"`
	LcmIDleTimeout              *int64                    `json:"lcm_idle_timeout,omitempty"` // [1-9][0-9]|[1-9][0-9][0-9]|[1-2][0-9][0-9][0-9]|3[0-5][0-9][0-9]|3600
	LcmIDleTimeoutOverride      bool                      `json:"lcm_idle_timeout_override,omitempty"`
	LcmNightModeBegins          string                    `json:"lcm_night_mode_begins,omitempty"`    // (^$)|(^(0[1-9])|(1[0-9])|(2[0-3])):([0-5][0-9]$)
	LcmNightModeEnds            string                    `json:"lcm_night_mode_ends,omitempty"`      // (^$)|(^(0[1-9])|(1[0-9])|(2[0-3])):([0-5][0-9]$)
	LcmOrientationOverride      *int64                    `json:"lcm_orientation_override,omitempty"` // 0|90|180|270
	LcmSettingsRestrictedAccess bool                      `json:"lcm_settings_restricted_access,omitempty"`
	LcmTrackerEnabled           bool                      `json:"lcm_tracker_enabled,omitempty"`
	LcmTrackerSeed              string                    `json:"lcm_tracker_seed,omitempty"`              // .{0,50}
	LedOverride                 string                    `json:"led_override,omitempty"`                  // default|on|off
	LedOverrideColor            string                    `json:"led_override_color,omitempty"`            // ^#(?:[0-9a-fA-F]{3}){1,2}$
	LedOverrideColorBrightness  *int64                    `json:"led_override_color_brightness,omitempty"` // ^[0-9][0-9]?$|^100$
	Locked                      bool                      `json:"locked,omitempty"`
	LowpfmodeOverride           bool                      `json:"lowpfmode_override,omitempty"`
	LteApn                      string                    `json:"lte_apn,omitempty"`       // .{1,128}
	LteAuthType                 string                    `json:"lte_auth_type,omitempty"` // PAP|CHAP|PAP-CHAP|NONE
	LteDataLimitEnabled         bool                      `json:"lte_data_limit_enabled,omitempty"`
	LteDataWarningEnabled       bool                      `json:"lte_data_warning_enabled,omitempty"`
	LteExtAnt                   bool                      `json:"lte_ext_ant,omitempty"`
	LteHardLimit                *int64                    `json:"lte_hard_limit,omitempty"`
	LtePassword                 string                    `json:"lte_password,omitempty"`
	LtePoe                      bool                      `json:"lte_poe,omitempty"`
	LteRoamingAllowed           bool                      `json:"lte_roaming_allowed,omitempty"`
	LteSimPin                   *int64                    `json:"lte_sim_pin,omitempty"`
	LteSoftLimit                *int64                    `json:"lte_soft_limit,omitempty"`
	LteUsername                 string                    `json:"lte_username,omitempty"`
	MapID                       string                    `json:"map_id,omitempty"`
	MbbOverrides                *DeviceMbbOverrides       `json:"mbb_overrides,omitempty"`
	MeshStaVapEnabled           bool                      `json:"mesh_sta_vap_enabled,omitempty"`
	MgmtNetworkID               string                    `json:"mgmt_network_id,omitempty"` // [\d\w]+
	Model                       string                    `json:"model,omitempty"`
	Name                        string                    `json:"name,omitempty"` // .{0,128}
	NutServer                   *DeviceNutServer          `json:"nut_server,omitempty"`
	OutdoorModeOverride         string                    `json:"outdoor_mode_override,omitempty"` // default|on|off
	OutletEnabled               bool                      `json:"outlet_enabled,omitempty"`
	OutletOverrides             []DeviceOutletOverrides   `json:"outlet_overrides,omitempty"`
	OutletPowerCycleEnabled     bool                      `json:"outlet_power_cycle_enabled,omitempty"`
	PeerToPeerMode              string                    `json:"peer_to_peer_mode,omitempty"` // ap|sta
	PoeMode                     string                    `json:"poe_mode,omitempty"`          // auto|pasv24|passthrough|off
	PortOverrides               []DevicePortOverrides     `json:"port_overrides"`
	PortTable                   []DevicePortTable         `json:"port_table,omitempty"`
	PowerSourceCtrl             string                    `json:"power_source_ctrl,omitempty"`        // auto|8023af|8023at|8023bt-type3|8023bt-type4|pasv24|poe-injector|ac|adapter|dc|rps
	PowerSourceCtrlBudget       *int64                    `json:"power_source_ctrl_budget,omitempty"` // [0-9]|[1-9][0-9]|[1-9][0-9][0-9]
	PowerSourceCtrlEnabled      bool                      `json:"power_source_ctrl_enabled,omitempty"`
	PtmpApMAC                   string                    `json:"ptmp_ap_mac,omitempty"` // ^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$
	PtpApMAC                    string                    `json:"ptp_ap_mac,omitempty"`  // ^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$
	RADIUSProfileID             string                    `json:"radiusprofile_id,omitempty"`
	RadioTable                  []DeviceRadioTable        `json:"radio_table,omitempty"`
	ResetbtnEnabled             string                    `json:"resetbtn_enabled,omitempty"` // on|off
	RpsOverride                 *DeviceRpsOverride        `json:"rps_override,omitempty"`
	SnmpContact                 string                    `json:"snmp_contact,omitempty"`  // .{0,255}
	SnmpLocation                string                    `json:"snmp_location,omitempty"` // .{0,255}
	State                       DeviceState               `json:"state"`
	StationMode                 string                    `json:"station_mode,omitempty"` // ptp|ptmp|wifi
	StpPriority                 *int64                    `json:"stp_priority,omitempty"` // 0|4096|8192|12288|16384|20480|24576|28672|32768|36864|40960|45056|49152|53248|57344|61440
	StpVersion                  string                    `json:"stp_version,omitempty"`  // stp|rstp|disabled
	SwitchVLANEnabled           bool                      `json:"switch_vlan_enabled,omitempty"`
	Type                        string                    `json:"type,omitempty"`
	UbbPairName                 string                    `json:"ubb_pair_name,omitempty"` // .{1,128}
	Volume                      *int64                    `json:"volume,omitempty"`        // [0-9]|[1-9][0-9]|100
	X                           float64                   `json:"x,omitempty"`
	XBaresipPassword            string                    `json:"x_baresip_password,omitempty"` // ^[a-zA-Z0-9_.\-!~*'()]*
	Y                           float64                   `json:"y,omitempty"`
}

func (dst *Device) UnmarshalJSON(b []byte) error {
	type Alias Device
	aux := &struct {
		StpPriority types.Number `json:"stp_priority"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	if val, err := aux.StpPriority.Int64(); err == nil {
		dst.StpPriority = &val
	}

	return nil
}

type DeviceConfigNetwork struct {
	BondingEnabled bool   `json:"bonding_enabled,omitempty"`
	DNS1           string `json:"dns1,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$|^$
	DNS2           string `json:"dns2,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$|^$
	DNSsuffix      string `json:"dnssuffix,omitempty"`
	Gateway        string `json:"gateway,omitempty"` // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	IP             string `json:"ip,omitempty"`      // ^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$
	Netmask        string `json:"netmask,omitempty"` // ^((128|192|224|240|248|252|254)\.0\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0\.0)|(255\.(((0|128|192|224|240|248|252|254)\.0)|255\.(0|128|192|224|240|248|252|254)))))$
	Type           string `json:"type,omitempty"`    // dhcp|static
}

func (dst *DeviceConfigNetwork) UnmarshalJSON(b []byte) error {
	type Alias DeviceConfigNetwork
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

type DeviceCurrentApn struct {
	Apn      string `json:"apn,omitempty"`
	AuthType string `json:"auth_type,omitempty"` // PAP|CHAP|PAP-CHAP|NONE
	PDpType  string `json:"pdp_type,omitempty"`  // IPv4|IPv6|IPv4v6
	Password string `json:"password,omitempty"`
	Roaming  bool   `json:"roaming,omitempty"`
	Username string `json:"username,omitempty"`
}

func (dst *DeviceCurrentApn) UnmarshalJSON(b []byte) error {
	type Alias DeviceCurrentApn
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

type DeviceEtherLighting struct {
	Behavior   string `json:"behavior,omitempty"`   // breath|steady
	Brightness *int64 `json:"brightness,omitempty"` // [1-9]|[1-9][0-9]|100
	LedMode    string `json:"led_mode,omitempty"`   // standard|etherlighting
	Mode       string `json:"mode,omitempty"`       // speed|network
}

func (dst *DeviceEtherLighting) UnmarshalJSON(b []byte) error {
	type Alias DeviceEtherLighting
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

type DeviceEthernetOverrides struct {
	Disabled     bool   `json:"disabled,omitempty"`
	Ifname       string `json:"ifname,omitempty"`       // eth[0-9]{1,2}
	NetworkGroup string `json:"networkgroup,omitempty"` // LAN[2-8]?|WAN[2-9]?
}

func (dst *DeviceEthernetOverrides) UnmarshalJSON(b []byte) error {
	type Alias DeviceEthernetOverrides
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

type DeviceMbbOverrides struct {
	PrimarySlot *int64      `json:"primary_slot,omitempty"` // 1|2
	Sim         []DeviceSim `json:"sim,omitempty"`
}

func (dst *DeviceMbbOverrides) UnmarshalJSON(b []byte) error {
	type Alias DeviceMbbOverrides
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

type DeviceNutServer struct {
	CredentialRequired bool   `json:"credential_required,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`
	ID                 string `json:"id,omitempty"`
	Password           string `json:"password,omitempty"`
	Port               *int64 `json:"port,omitempty"`
	Username           string `json:"username,omitempty"`
}

func (dst *DeviceNutServer) UnmarshalJSON(b []byte) error {
	type Alias DeviceNutServer
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

type DeviceOutletOverrides struct {
	CycleEnabled bool   `json:"cycle_enabled,omitempty"`
	Index        *int64 `json:"index,omitempty"`
	Name         string `json:"name,omitempty"` // .{0,128}
	RelayState   bool   `json:"relay_state,omitempty"`
}

func (dst *DeviceOutletOverrides) UnmarshalJSON(b []byte) error {
	type Alias DeviceOutletOverrides
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

type DevicePortOverrides struct {
	AggregateMembers             []int64           `json:"aggregate_members,omitempty"` // [1-9]|[1-4][0-9]|5[0-6]
	Autoneg                      bool              `json:"autoneg,omitempty"`
	Dot1XCtrl                    string            `json:"dot1x_ctrl,omitempty"`             // auto|force_authorized|force_unauthorized|mac_based|multi_host
	Dot1XIDleTimeout             *int64            `json:"dot1x_idle_timeout,omitempty"`     // [0-9]|[1-9][0-9]{1,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5]
	EgressRateLimitKbps          *int64            `json:"egress_rate_limit_kbps,omitempty"` // 6[4-9]|[7-9][0-9]|[1-9][0-9]{2,6}
	EgressRateLimitKbpsEnabled   bool              `json:"egress_rate_limit_kbps_enabled,omitempty"`
	ExcludedNetworkIDs           []string          `json:"excluded_networkconf_ids,omitempty"`
	FecMode                      string            `json:"fec_mode,omitempty"` // rs-fec|fc-fec|default|disabled
	FlowControlEnabled           bool              `json:"flow_control_enabled,omitempty"`
	Forward                      string            `json:"forward,omitempty"` // all|native|customize|disabled
	FullDuplex                   bool              `json:"full_duplex,omitempty"`
	Isolation                    bool              `json:"isolation,omitempty"`
	LldpmedEnabled               bool              `json:"lldpmed_enabled,omitempty"`
	LldpmedNotifyEnabled         bool              `json:"lldpmed_notify_enabled,omitempty"`
	MirrorPortIDX                *int64            `json:"mirror_port_idx,omitempty"` // [1-9]|[1-4][0-9]|5[0-6]
	MulticastRouterNetworkIDs    []string          `json:"multicast_router_networkconf_ids,omitempty"`
	NATiveNetworkID              string            `json:"native_networkconf_id,omitempty"`
	Name                         string            `json:"name,omitempty"`     // .{0,128}
	OpMode                       string            `json:"op_mode,omitempty"`  // switch|mirror|aggregate
	PoeMode                      string            `json:"poe_mode,omitempty"` // auto|pasv24|passthrough|off
	PortIDX                      *int64            `json:"port_idx,omitempty"` // [1-9]|[1-4][0-9]|5[0-6]
	PortKeepaliveEnabled         bool              `json:"port_keepalive_enabled,omitempty"`
	PortProfileID                string            `json:"portconf_id,omitempty"` // [\d\w]+
	PortSecurityEnabled          bool              `json:"port_security_enabled,omitempty"`
	PortSecurityMACAddress       []string          `json:"port_security_mac_address,omitempty"` // ^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$
	PriorityQueue1Level          *int64            `json:"priority_queue1_level,omitempty"`     // [0-9]|[1-9][0-9]|100
	PriorityQueue2Level          *int64            `json:"priority_queue2_level,omitempty"`     // [0-9]|[1-9][0-9]|100
	PriorityQueue3Level          *int64            `json:"priority_queue3_level,omitempty"`     // [0-9]|[1-9][0-9]|100
	PriorityQueue4Level          *int64            `json:"priority_queue4_level,omitempty"`     // [0-9]|[1-9][0-9]|100
	QOSProfile                   *DeviceQOSProfile `json:"qos_profile,omitempty"`
	SettingPreference            string            `json:"setting_preference,omitempty"` // auto|manual
	Speed                        *int64            `json:"speed,omitempty"`              // 10|100|1000|2500|5000|10000|20000|25000|40000|50000|100000
	StormctrlBroadcastastEnabled bool              `json:"stormctrl_bcast_enabled,omitempty"`
	StormctrlBroadcastastLevel   *int64            `json:"stormctrl_bcast_level,omitempty"` // [0-9]|[1-9][0-9]|100
	StormctrlBroadcastastRate    *int64            `json:"stormctrl_bcast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
	StormctrlMcastEnabled        bool              `json:"stormctrl_mcast_enabled,omitempty"`
	StormctrlMcastLevel          *int64            `json:"stormctrl_mcast_level,omitempty"` // [0-9]|[1-9][0-9]|100
	StormctrlMcastRate           *int64            `json:"stormctrl_mcast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
	StormctrlType                string            `json:"stormctrl_type,omitempty"`        // level|rate
	StormctrlUcastEnabled        bool              `json:"stormctrl_ucast_enabled,omitempty"`
	StormctrlUcastLevel          *int64            `json:"stormctrl_ucast_level,omitempty"` // [0-9]|[1-9][0-9]|100
	StormctrlUcastRate           *int64            `json:"stormctrl_ucast_rate,omitempty"`  // [0-9]|[1-9][0-9]{1,6}|1[0-3][0-9]{6}|14[0-7][0-9]{5}|148[0-7][0-9]{4}|14880000
	StpPortMode                  bool              `json:"stp_port_mode,omitempty"`
	TaggedVLANMgmt               string            `json:"tagged_vlan_mgmt,omitempty"` // auto|block_all|custom
	VoiceNetworkID               string            `json:"voice_networkconf_id,omitempty"`
}

func (dst *DevicePortOverrides) UnmarshalJSON(b []byte) error {
	type Alias DevicePortOverrides
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

type DeviceQOSMarking struct {
	CosCode          *int64 `json:"cos_code,omitempty"`           // [0-7]
	DscpCode         *int64 `json:"dscp_code,omitempty"`          // 0|8|16|24|32|40|48|56|10|12|14|18|20|22|26|28|30|34|36|38|44|46
	IPPrecedenceCode *int64 `json:"ip_precedence_code,omitempty"` // [0-7]
	Queue            *int64 `json:"queue,omitempty"`              // [0-7]
}

func (dst *DeviceQOSMarking) UnmarshalJSON(b []byte) error {
	type Alias DeviceQOSMarking
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

type DeviceQOSMatching struct {
	CosCode          *int64 `json:"cos_code,omitempty"`           // [0-7]
	DscpCode         *int64 `json:"dscp_code,omitempty"`          // [0-9]|[1-5][0-9]|6[0-3]
	DstPort          *int64 `json:"dst_port,omitempty"`           // [0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[0-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-4]|65535
	IPPrecedenceCode *int64 `json:"ip_precedence_code,omitempty"` // [0-7]
	Protocol         string `json:"protocol,omitempty"`           // ([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])|ah|ax.25|dccp|ddp|egp|eigrp|encap|esp|etherip|fc|ggp|gre|hip|hmp|icmp|idpr-cmtp|idrp|igmp|igp|ip|ipcomp|ipencap|ipip|ipv6|ipv6-frag|ipv6-icmp|ipv6-nonxt|ipv6-opts|ipv6-route|isis|iso-tp4|l2tp|manet|mobility-header|mpls-in-ip|ospf|pim|pup|rdp|rohc|rspf|rsvp|sctp|shim6|skip|st|tcp|udp|udplite|vmtp|vrrp|wesp|xns-idp|xtp
	SrcPort          *int64 `json:"src_port,omitempty"`           // [0-9]|[1-9][0-9]|[1-9][0-9][0-9]|[1-9][0-9][0-9][0-9]|[1-5][0-9][0-9][0-9][0-9]|6[0-4][0-9][0-9][0-9]|65[0-4][0-9][0-9]|655[0-2][0-9]|6553[0-4]|65535
}

func (dst *DeviceQOSMatching) UnmarshalJSON(b []byte) error {
	type Alias DeviceQOSMatching
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

type DeviceQOSPolicies struct {
	QOSMarking  *DeviceQOSMarking  `json:"qos_marking,omitempty"`
	QOSMatching *DeviceQOSMatching `json:"qos_matching,omitempty"`
}

func (dst *DeviceQOSPolicies) UnmarshalJSON(b []byte) error {
	type Alias DeviceQOSPolicies
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

type DeviceQOSProfile struct {
	QOSPolicies    []DeviceQOSPolicies `json:"qos_policies,omitempty"`
	QOSProfileMode string              `json:"qos_profile_mode,omitempty"` // custom|unifi_play|aes67_audio|crestron_audio_video|dante_audio|ndi_aes67_audio|ndi_dante_audio|qsys_audio_video|qsys_video_dante_audio|sdvoe_aes67_audio|sdvoe_dante_audio|shure_audio
}

func (dst *DeviceQOSProfile) UnmarshalJSON(b []byte) error {
	type Alias DeviceQOSProfile
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

type DeviceRadioIDentifiers struct {
	DeviceID  string `json:"device_id,omitempty"`
	RadioName string `json:"radio_name,omitempty"`
}

func (dst *DeviceRadioIDentifiers) UnmarshalJSON(b []byte) error {
	type Alias DeviceRadioIDentifiers
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

type DeviceRadioTable struct {
	AntennaGain            *int64                   `json:"antenna_gain,omitempty"` // ^-?([0-9]|[1-9][0-9])
	AntennaID              *int64                   `json:"antenna_id,omitempty"`   // -1|[0-9]
	AssistedRoamingEnabled bool                     `json:"assisted_roaming_enabled,omitempty"`
	AssistedRoamingRssi    *int64                   `json:"assisted_roaming_rssi,omitempty"` // ^-([6-7][0-9]|80)$
	Channel                string                   `json:"channel,omitempty"`               // [0-9]|[1][0-4]|1.5|2.5|3.5|4.5|5.5|6.5|5|16|17|21|25|29|33|34|36|37|38|40|41|42|44|45|46|48|49|52|53|56|57|60|61|64|65|69|73|77|81|85|89|93|97|100|101|104|105|108|109|112|113|117|116|120|121|124|125|128|129|132|133|136|137|140|141|144|145|149|153|157|161|165|169|173|177|181|183|184|185|187|188|189|192|193|196|197|201|205|209|213|217|221|225|229|233|auto
	Dfs                    bool                     `json:"dfs,omitempty"`
	HardNoiseFloorEnabled  bool                     `json:"hard_noise_floor_enabled,omitempty"`
	Ht                     *int64                   `json:"ht,omitempty"` // 20|40|80|160|240|320|1080|2160|4320
	LoadbalanceEnabled     bool                     `json:"loadbalance_enabled,omitempty"`
	Maxsta                 *int64                   `json:"maxsta,omitempty"`   // [1-9]|[1-9][0-9]|1[0-9]{2}|200|^$
	MinRssi                *int64                   `json:"min_rssi,omitempty"` // ^-(6[7-9]|[7-8][0-9]|90)$
	MinRssiEnabled         bool                     `json:"min_rssi_enabled,omitempty"`
	Name                   string                   `json:"name,omitempty"`
	Radio                  string                   `json:"radio,omitempty"` // ng|na|ad|6e
	RadioIDentifiers       []DeviceRadioIDentifiers `json:"radio_identifiers,omitempty"`
	SensLevel              *int64                   `json:"sens_level,omitempty"` // ^-([5-8][0-9]|90)$
	SensLevelEnabled       bool                     `json:"sens_level_enabled,omitempty"`
	TxPower                string                   `json:"tx_power,omitempty"`      // [\d]+|auto
	TxPowerMode            string                   `json:"tx_power_mode,omitempty"` // auto|medium|high|low|custom|disabled
	VwireEnabled           bool                     `json:"vwire_enabled,omitempty"`
}

func (dst *DeviceRadioTable) UnmarshalJSON(b []byte) error {
	type Alias DeviceRadioTable
	aux := &struct {
		Ht types.Number `json:"ht"`

		*Alias
	}{
		Alias: (*Alias)(dst),
	}

	err := json.Unmarshal(b, &aux)
	if err != nil {
		return fmt.Errorf("unable to unmarshal alias: %w", err)
	}
	dst.Ht = types.ToInt64Pointer(aux.Ht)

	return nil
}

type DeviceRpsOverride struct {
	PowerManagementMode string               `json:"power_management_mode,omitempty"` // dynamic|static
	RpsPortTable        []DeviceRpsPortTable `json:"rps_port_table,omitempty"`
}

func (dst *DeviceRpsOverride) UnmarshalJSON(b []byte) error {
	type Alias DeviceRpsOverride
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

type DeviceRpsPortTable struct {
	Name     string `json:"name,omitempty"`      // .{0,32}
	PortIDX  *int64 `json:"port_idx,omitempty"`  // [1-8]
	PortMode string `json:"port_mode,omitempty"` // auto|force_active|manual|disabled
}

func (dst *DeviceRpsPortTable) UnmarshalJSON(b []byte) error {
	type Alias DeviceRpsPortTable
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

type DeviceSim struct {
	CurrentApn               *DeviceCurrentApn `json:"current_apn,omitempty"`
	DataHardLimitBytes       *int64            `json:"data_hard_limit_bytes,omitempty"`
	DataLimitEnabled         bool              `json:"data_limit_enabled,omitempty"`
	DataSoftLimitBytes       *int64            `json:"data_soft_limit_bytes,omitempty"`
	DataSoftLimitDisplayUnit string            `json:"data_soft_limit_display_unit,omitempty"` // MB|GB
	DataWarningThreshold     *int64            `json:"data_warning_threshold,omitempty"`       // [0-9]|[1-9][0-9]|100
	ResetDate                *int64            `json:"reset_date,omitempty"`                   // [0-9]|[1-2][0-9]|3[0-1]
	ResetPolicy              string            `json:"reset_policy,omitempty"`                 // day|week|month
	Slot                     *int64            `json:"slot,omitempty"`                         // 1|2
	UseCustomApn             bool              `json:"use_custom_apn,omitempty"`
}

func (dst *DeviceSim) UnmarshalJSON(b []byte) error {
	type Alias DeviceSim
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

func (c *ApiClient) listDevice(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]Device, error) {
	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/stat/device", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getDevice(
	ctx context.Context,
	site string,
	id string,
) (*Device, error) {
	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/stat/device/%s", site, id),
		nil,
		&respBody,
	)
	if err != nil {
		return nil, err
	}
	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	d := respBody.Data[0]
	return &d, nil
}

func (c *ApiClient) deleteDevice(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/stat/device/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createDevice(
	ctx context.Context,
	site string,
	d *Device,
) (*Device, error) {
	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/stat/device", site),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}

func (c *ApiClient) updateDevice(
	ctx context.Context,
	site string,
	d *Device,
) (*Device, error) {
	var respBody struct {
		Meta meta     `json:"meta"`
		Data []Device `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/device/%s", site, d.ID),
		d,
		&respBody,
	)
	if err != nil {
		return nil, err
	}

	if len(respBody.Data) != 1 {
		return nil, &NotFoundError{}
	}

	res := respBody.Data[0]

	return &res, nil
}
