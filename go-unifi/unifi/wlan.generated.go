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

type WLAN struct {
	ID     string `json:"_id,omitempty"`
	SiteID string `json:"site_id,omitempty"`

	Hidden   bool   `json:"attr_hidden,omitempty"`
	HiddenID string `json:"attr_hidden_id,omitempty"`
	NoDelete bool   `json:"attr_no_delete,omitempty"`
	NoEdit   bool   `json:"attr_no_edit,omitempty"`

	ApGroupIDs                  []string                   `json:"ap_group_ids,omitempty"`
	ApGroupMode                 string                     `json:"ap_group_mode,omitempty"` // all|groups|devices
	AuthCache                   bool                       `json:"auth_cache"`
	BSupported                  bool                       `json:"b_supported"`
	BroadcastFilterEnabled      bool                       `json:"bc_filter_enabled"`
	BroadcastFilterList         []string                   `json:"bc_filter_list,omitempty"` // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	BssTransition               bool                       `json:"bss_transition"`
	CountryBeacon               bool                       `json:"country_beacon"`
	DPIEnabled                  bool                       `json:"dpi_enabled"`
	DPIgroupID                  string                     `json:"dpigroup_id"`         // [\d\w]+|^$
	DTIM6E                      *int64                     `json:"dtim_6e,omitempty"`   // ^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	DTIMMode                    string                     `json:"dtim_mode,omitempty"` // default|custom
	DTIMNa                      *int64                     `json:"dtim_na,omitempty"`   // ^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	DTIMNg                      *int64                     `json:"dtim_ng,omitempty"`   // ^([1-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$|^$
	ElementAdopt                bool                       `json:"element_adopt"`
	Enabled                     bool                       `json:"enabled"`
	EnhancedIot                 bool                       `json:"enhanced_iot"`
	FastRoamingEnabled          bool                       `json:"fast_roaming_enabled"`
	GroupRekey                  *int64                     `json:"group_rekey,omitempty"` // ^(0|[6-9][0-9]|[1-9][0-9]{2,3}|[1-7][0-9]{4}|8[0-5][0-9]{3}|86[0-3][0-9][0-9]|86400)$
	HideSSID                    bool                       `json:"hide_ssid"`
	Hotspot2                    *WLANHotspot2              `json:"hotspot2,omitempty"`
	Hotspot2ConfEnabled         bool                       `json:"hotspot2conf_enabled"`
	IappEnabled                 bool                       `json:"iapp_enabled"`
	IsGuest                     bool                       `json:"is_guest"`
	L2Isolation                 bool                       `json:"l2_isolation"`
	LogLevel                    string                     `json:"log_level,omitempty"`
	MACFilterEnabled            bool                       `json:"mac_filter_enabled"`
	MACFilterList               []string                   `json:"mac_filter_list,omitempty"`   // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	MACFilterPolicy             string                     `json:"mac_filter_policy,omitempty"` // allow|deny
	MdnsProxyCustom             []WLANMdnsProxyCustom      `json:"mdns_proxy_custom,omitempty"`
	MdnsProxyMode               string                     `json:"mdns_proxy_mode,omitempty"` // off|auto|custom
	MinrateNaAdvertisingRates   bool                       `json:"minrate_na_advertising_rates"`
	MinrateNaDataRateKbps       *int64                     `json:"minrate_na_data_rate_kbps,omitempty"`
	MinrateNaEnabled            bool                       `json:"minrate_na_enabled"`
	MinrateNgAdvertisingRates   bool                       `json:"minrate_ng_advertising_rates"`
	MinrateNgDataRateKbps       *int64                     `json:"minrate_ng_data_rate_kbps,omitempty"`
	MinrateNgEnabled            bool                       `json:"minrate_ng_enabled"`
	MinrateSettingPreference    string                     `json:"minrate_setting_preference,omitempty"` // auto|manual
	MloEnabled                  bool                       `json:"mlo_enabled"`
	MulticastEnhanceEnabled     bool                       `json:"mcastenhance_enabled"`
	Name                        string                     `json:"name,omitempty"` // .{1,32}
	NameCombineEnabled          bool                       `json:"name_combine_enabled"`
	NameCombineSuffix           string                     `json:"name_combine_suffix,omitempty"` // .{0,8}
	NasIDentifier               string                     `json:"nas_identifier,omitempty"`      // .{0,48}
	NasIDentifierType           string                     `json:"nas_identifier_type,omitempty"` // ap_name|ap_mac|bssid|site_name|custom
	NetworkID                   string                     `json:"networkconf_id,omitempty"`
	No2GhzOui                   bool                       `json:"no2ghz_oui"`
	OptimizeIotWifiConnectivity bool                       `json:"optimize_iot_wifi_connectivity"`
	P2P                         bool                       `json:"p2p"`
	P2PCrossConnect             bool                       `json:"p2p_cross_connect"`
	PMFCipher                   string                     `json:"pmf_cipher,omitempty"` // auto|aes-128-cmac|bip-gmac-256
	PMFMode                     string                     `json:"pmf_mode,omitempty"`   // disabled|optional|required
	Priority                    string                     `json:"priority,omitempty"`   // medium|high|low
	PrivatePresharedKeys        []WLANPrivatePresharedKeys `json:"private_preshared_keys,omitempty"`
	PrivatePresharedKeysEnabled bool                       `json:"private_preshared_keys_enabled"`
	ProxyArp                    bool                       `json:"proxy_arp"`
	RADIUSDasEnabled            bool                       `json:"radius_das_enabled"`
	RADIUSMACAuthEnabled        bool                       `json:"radius_mac_auth_enabled"`
	RADIUSMACaclEmptyPassword   bool                       `json:"radius_macacl_empty_password"`
	RADIUSMACaclFormat          string                     `json:"radius_macacl_format,omitempty"` // none_lower|hyphen_lower|colon_lower|none_upper|hyphen_upper|colon_upper
	RADIUSProfileID             string                     `json:"radiusprofile_id,omitempty"`
	RoamClusterID               *int64                     `json:"roam_cluster_id,omitempty"` // [0-9]|[1-2][0-9]|[3][0-1]|^$
	RrmEnabled                  bool                       `json:"rrm_enabled"`
	SaeAntiClogging             *int64                     `json:"sae_anti_clogging,omitempty"`
	SaeGroups                   []int64                    `json:"sae_groups,omitempty"`
	SaePsk                      []WLANSaePsk               `json:"sae_psk,omitempty"`
	SaePskVLANRequired          bool                       `json:"sae_psk_vlan_required"`
	SaeSync                     *int64                     `json:"sae_sync,omitempty"`
	Schedule                    []string                   `json:"schedule,omitempty"` // (sun|mon|tue|wed|thu|fri|sat)(\-(sun|mon|tue|wed|thu|fri|sat))?\|([0-2][0-9][0-5][0-9])\-([0-2][0-9][0-5][0-9])
	ScheduleEnabled             bool                       `json:"schedule_enabled"`
	ScheduleReversed            bool                       `json:"schedule_reversed"`
	ScheduleWithDuration        []WLANScheduleWithDuration `json:"schedule_with_duration"`
	Security                    string                     `json:"security,omitempty"`           // open|wpapsk|wep|wpaeap|osen
	SettingPreference           string                     `json:"setting_preference,omitempty"` // auto|manual
	TdlsProhibit                bool                       `json:"tdls_prohibit"`
	UapsdEnabled                bool                       `json:"uapsd_enabled"`
	UidWorkspaceUrl             string                     `json:"uid_workspace_url,omitempty"`
	UserGroupID                 string                     `json:"usergroup_id,omitempty"`
	VLAN                        *int64                     `json:"vlan,omitempty"` // [2-9]|[1-9][0-9]{1,2}|[1-3][0-9]{3}|40[0-8][0-9]|409[0-5]|^$
	VLANEnabled                 bool                       `json:"vlan_enabled"`
	WEPIDX                      *int64                     `json:"wep_idx,omitempty"`    // [1-4]
	WLANBand                    string                     `json:"wlan_band,omitempty"`  // 2g|5g|both
	WLANBands                   []string                   `json:"wlan_bands,omitempty"` // 2g|5g|6g
	WLANGroupID                 string                     `json:"wlangroup_id"`
	WPA3Enhanced192             bool                       `json:"wpa3_enhanced_192"`
	WPA3FastRoaming             bool                       `json:"wpa3_fast_roaming"`
	WPA3Support                 bool                       `json:"wpa3_support"`
	WPA3Transition              bool                       `json:"wpa3_transition"`
	WPAEnc                      string                     `json:"wpa_enc,omitempty"`        // auto|ccmp|gcmp|ccmp-256|gcmp-256
	WPAMode                     string                     `json:"wpa_mode,omitempty"`       // auto|wpa1|wpa2
	WPAPskRADIUS                string                     `json:"wpa_psk_radius,omitempty"` // disabled|optional|required
	XIappKey                    string                     `json:"x_iapp_key,omitempty"`     // [0-9A-Fa-f]{32}
	XPassphrase                 string                     `json:"x_passphrase,omitempty"`   // [\x20-\x7E]{8,255}|[0-9a-fA-F]{64}
	XWEP                        string                     `json:"x_wep,omitempty"`
}

func (dst *WLAN) UnmarshalJSON(b []byte) error {
	type Alias WLAN
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

type WLANCapab struct {
	Port     *int64 `json:"port,omitempty"`     // ^(0|[1-9][0-9]{0,3}|[1-5][0-9]{4}|6[0-4][0-9]{3}|65[0-4][0-9]{2}|655[0-2][0-9]|6553[0-5])|$
	Protocol string `json:"protocol,omitempty"` // icmp|tcp_udp|tcp|udp|esp
	Status   string `json:"status,omitempty"`   // closed|open|unknown
}

func (dst *WLANCapab) UnmarshalJSON(b []byte) error {
	type Alias WLANCapab
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

type WLANCellularNetworkList struct {
	CountryCode *int64 `json:"country_code,omitempty"` // [1-9]{1}[0-9]{0,3}
	Mcc         *int64 `json:"mcc,omitempty"`
	Mnc         *int64 `json:"mnc,omitempty"`
	Name        string `json:"name,omitempty"` // .{1,128}
}

func (dst *WLANCellularNetworkList) UnmarshalJSON(b []byte) error {
	type Alias WLANCellularNetworkList
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

type WLANCustomServices struct {
	Address string `json:"address,omitempty"` // ^_[a-zA-Z0-9._-]+\._(tcp|udp)(\.local)?$
	Name    string `json:"name,omitempty"`
}

func (dst *WLANCustomServices) UnmarshalJSON(b []byte) error {
	type Alias WLANCustomServices
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

type WLANFriendlyName struct {
	Language string `json:"language,omitempty"` // [a-z]{3}
	Text     string `json:"text,omitempty"`     // .{1,128}
}

func (dst *WLANFriendlyName) UnmarshalJSON(b []byte) error {
	type Alias WLANFriendlyName
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

type WLANHotspot2 struct {
	Capab                   []WLANCapab                 `json:"capab,omitempty"`
	CellularNetworkList     []WLANCellularNetworkList   `json:"cellular_network_list,omitempty"`
	DomainNameList          []string                    `json:"domain_name_list,omitempty"` // .{1,128}
	FriendlyName            []WLANFriendlyName          `json:"friendly_name,omitempty"`
	IPaddrTypeAvailV4       *int64                      `json:"ipaddr_type_avail_v4,omitempty"` // 0|1|2|3|4|5|6|7
	IPaddrTypeAvailV6       *int64                      `json:"ipaddr_type_avail_v6,omitempty"` // 0|1|2
	MetricsDownlinkLoad     *int64                      `json:"metrics_downlink_load,omitempty"`
	MetricsDownlinkLoadSet  bool                        `json:"metrics_downlink_load_set"`
	MetricsDownlinkSpeed    *int64                      `json:"metrics_downlink_speed,omitempty"`
	MetricsDownlinkSpeedSet bool                        `json:"metrics_downlink_speed_set"`
	MetricsInfoAtCapacity   bool                        `json:"metrics_info_at_capacity"`
	MetricsInfoLinkStatus   string                      `json:"metrics_info_link_status,omitempty"` // up|down|test
	MetricsInfoSymmetric    bool                        `json:"metrics_info_symmetric"`
	MetricsMeasurement      *int64                      `json:"metrics_measurement,omitempty"`
	MetricsMeasurementSet   bool                        `json:"metrics_measurement_set"`
	MetricsStatus           bool                        `json:"metrics_status"`
	MetricsUplinkLoad       *int64                      `json:"metrics_uplink_load,omitempty"`
	MetricsUplinkLoadSet    bool                        `json:"metrics_uplink_load_set"`
	MetricsUplinkSpeed      *int64                      `json:"metrics_uplink_speed,omitempty"`
	MetricsUplinkSpeedSet   bool                        `json:"metrics_uplink_speed_set"`
	NaiRealmList            []WLANNaiRealmList          `json:"nai_realm_list,omitempty"`
	NetworkType             *int64                      `json:"network_type,omitempty"` // 0|1|2|3|4|5|14|15
	RoamingConsortiumList   []WLANRoamingConsortiumList `json:"roaming_consortium_list,omitempty"`
	VenueGroup              *int64                      `json:"venue_group,omitempty"` // 0|1|2|3|4|5|6|7|8|9|10|11
	VenueName               []WLANVenueName             `json:"venue_name,omitempty"`
	VenueType               *int64                      `json:"venue_type,omitempty"` // 0|1|2|3|4|5|6|7|8|9|10|11|12|13|14|15
}

func (dst *WLANHotspot2) UnmarshalJSON(b []byte) error {
	type Alias WLANHotspot2
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

type WLANMdnsProxyCustom struct {
	ApGroupIDs         []string                 `json:"ap_group_ids,omitempty"`
	ApMACs             []string                 `json:"ap_macs,omitempty"`       // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	ApScopeMode        string                   `json:"ap_scope_mode,omitempty"` // all|specific|group
	CustomServices     []WLANCustomServices     `json:"custom_services,omitempty"`
	IsolationEnabled   bool                     `json:"isolation_enabled"`
	NetworkIDs         []string                 `json:"networkconf_ids,omitempty"`
	PredefinedServices []WLANPredefinedServices `json:"predefined_services,omitempty"`
	ServicesMode       string                   `json:"services_mode,omitempty"` // all|specific|none
}

func (dst *WLANMdnsProxyCustom) UnmarshalJSON(b []byte) error {
	type Alias WLANMdnsProxyCustom
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

type WLANNaiRealmList struct {
	AuthIDs   []int64 `json:"auth_ids,omitempty"`   // 0|1|2|3|4|5
	AuthVals  []int64 `json:"auth_vals,omitempty"`  // 0|1|2|3|4|5|6|7|8|9|10
	EapMethod *int64  `json:"eap_method,omitempty"` // 13|21|18|23|50
	Encoding  *int64  `json:"encoding,omitempty"`   // 0|1
	Name      string  `json:"name,omitempty"`       // .{1,128}
	Status    bool    `json:"status"`
}

func (dst *WLANNaiRealmList) UnmarshalJSON(b []byte) error {
	type Alias WLANNaiRealmList
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

type WLANPredefinedServices struct {
	Code string `json:"code,omitempty"` // amazon_devices|android_tv_remote|apple_airDrop|apple_airPlay|apple_file_sharing|apple_iChat|apple_iTunes|aqara|bose|dns_service_discovery|ftp_servers|google_chromecast|homeKit|matter_network|philips_hue|printers|roku|scanners|sonos|spotify_connect|ssh_servers|time_capsule|web_servers|windows_file_sharing_samba
}

func (dst *WLANPredefinedServices) UnmarshalJSON(b []byte) error {
	type Alias WLANPredefinedServices
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

type WLANPrivatePresharedKeys struct {
	NetworkID string `json:"networkconf_id,omitempty"`
	Password  string `json:"password,omitempty"` // [\x20-\x7E]{8,255}
}

func (dst *WLANPrivatePresharedKeys) UnmarshalJSON(b []byte) error {
	type Alias WLANPrivatePresharedKeys
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

type WLANRoamingConsortiumList struct {
	Name string `json:"name,omitempty"` // .{1,128}
	Oid  string `json:"oid,omitempty"`  // .{1,128}
}

func (dst *WLANRoamingConsortiumList) UnmarshalJSON(b []byte) error {
	type Alias WLANRoamingConsortiumList
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

type WLANSaePsk struct {
	ID   string `json:"id,omitempty"`   // .{0,128}
	MAC  string `json:"mac,omitempty"`  // ^([0-9A-Fa-f]{2}:){5}([0-9A-Fa-f]{2})$
	Psk  string `json:"psk,omitempty"`  // [\x20-\x7E]{8,255}
	VLAN *int64 `json:"vlan,omitempty"` // [0-9]|[1-9][0-9]{1,2}|[1-3][0-9]{3}|40[0-8][0-9]|409[0-5]|^$
}

func (dst *WLANSaePsk) UnmarshalJSON(b []byte) error {
	type Alias WLANSaePsk
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

type WLANScheduleWithDuration struct {
	DurationMinutes *int64   `json:"duration_minutes,omitempty"`   // ^[1-9][0-9]*$
	Name            string   `json:"name,omitempty"`               // .*
	StartDaysOfWeek []string `json:"start_days_of_week,omitempty"` // ^(sun|mon|tue|wed|thu|fri|sat)$
	StartHour       *int64   `json:"start_hour,omitempty"`         // ^(1?[0-9])|(2[0-3])$
	StartMinute     *int64   `json:"start_minute,omitempty"`       // ^[0-5]?[0-9]$
}

func (dst *WLANScheduleWithDuration) UnmarshalJSON(b []byte) error {
	type Alias WLANScheduleWithDuration
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

type WLANVenueName struct {
	Language string `json:"language,omitempty"` // [a-z]{0,3}
	Name     string `json:"name,omitempty"`
	Url      string `json:"url,omitempty"`
}

func (dst *WLANVenueName) UnmarshalJSON(b []byte) error {
	type Alias WLANVenueName
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

func (c *ApiClient) listWLAN(
	ctx context.Context,
	site string,
	query ...map[string]string,
) ([]WLAN, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []WLAN `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/wlanconf", site),
		nil,
		&respBody,
		query...,
	)
	if err != nil {
		return nil, err
	}
	return respBody.Data, nil
}

func (c *ApiClient) getWLAN(
	ctx context.Context,
	site string,
	id string,
) (*WLAN, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []WLAN `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("api/s/%s/rest/wlanconf/%s", site, id),
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

func (c *ApiClient) deleteWLAN(
	ctx context.Context,
	site string,
	id string,
) error {
	err := c.do(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("api/s/%s/rest/wlanconf/%s", site, id),
		struct{}{},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (c *ApiClient) createWLAN(
	ctx context.Context,
	site string,
	d *WLAN,
) (*WLAN, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []WLAN `json:"data"`
	}

	err := c.do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("api/s/%s/rest/wlanconf", site),
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

func (c *ApiClient) updateWLAN(
	ctx context.Context,
	site string,
	d *WLAN,
) (*WLAN, error) {
	var respBody struct {
		Meta meta   `json:"meta"`
		Data []WLAN `json:"data"`
	}
	err := c.do(
		ctx,
		http.MethodPut,
		fmt.Sprintf("api/s/%s/rest/wlanconf/%s", site, d.ID),
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
