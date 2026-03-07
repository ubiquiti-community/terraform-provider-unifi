package unifi

import (
	"context"
	"fmt"
	"net/http"
)

// just to fix compile issues with the import.
var (
	_ fmt.Formatter
	_ context.Context
)

type ClientInfoFingerprint struct {
	ComputedDevId  *int64 `json:"computed_dev_id,omitempty"`
	ComputedEngine *int64 `json:"computed_engine,omitempty"`
	Confidence     *int64 `json:"confidence,omitempty"`
	DevCat         *int64 `json:"dev_cat,omitempty"`
	DevFamily      *int64 `json:"dev_family,omitempty"`
	DevId          *int64 `json:"dev_id,omitempty"`
	DevIdOverride  *int64 `json:"dev_id_override,omitempty"`
	DevVendor      *int64 `json:"dev_vendor,omitempty"`
	HasOverride    bool   `json:"has_override,omitempty"`
	OsName         *int64 `json:"os_name,omitempty"`
}

type ClientInfoDeviceInfo struct {
	IconFilename      string     `json:"icon_filename,omitempty"`
	IconResolutions   [][]*int64 `json:"icon_resolutions,omitempty"`
	ViewInApplication bool       `json:"view_in_application,omitempty"`
}

type ClientInfoDetailedStates struct {
	UplinkNearPowerLimit bool `json:"uplink_near_power_limit,omitempty"`
}

type ClientInfo struct {
	Anomalies                           *int64                   `json:"anomalies,omitempty"`
	ApMac                               string                   `json:"ap_mac,omitempty"`
	AssocTime                           *int64                   `json:"assoc_time,omitempty"`
	Authorized                          bool                     `json:"authorized,omitempty"`
	Blocked                             bool                     `json:"blocked,omitempty"`
	Bssid                               string                   `json:"bssid,omitempty"`
	Ccq                                 *int64                   `json:"ccq,omitempty"`
	Channel                             *int64                   `json:"channel,omitempty"`
	ChannelWidth                        string                   `json:"channel_width,omitempty"`
	DetailedStates                      ClientInfoDetailedStates `json:"detailed_states"`
	DhcpendTime                         *int64                   `json:"dhcpend_time,omitempty"`
	DisplayName                         string                   `json:"display_name,omitempty"`
	Essid                               string                   `json:"essid,omitempty"`
	Fingerprint                         ClientInfoFingerprint    `json:"fingerprint"`
	FirstSeen                           *int64                   `json:"first_seen,omitempty"`
	FixedApEnabled                      bool                     `json:"fixed_ap_enabled,omitempty"`
	FixedIP                             string                   `json:"fixed_ip,omitempty"`
	GwMac                               string                   `json:"gw_mac,omitempty"`
	Hostname                            string                   `json:"hostname,omitempty"`
	Id                                  string                   `json:"id,omitempty"`
	Idletime                            *int64                   `json:"idletime,omitempty"`
	IP                                  string                   `json:"ip,omitempty"`
	Ipv4LeaseExpirationTimestampSeconds *int64                   `json:"ipv4_lease_expiration_timestamp_seconds,omitempty"`
	Ipv6Address                         []string                 `json:"ipv6_address,omitempty"`
	IsAllowedInVisualProgramming        bool                     `json:"is_allowed_in_visual_programming,omitempty"`
	IsGuest                             bool                     `json:"is_guest,omitempty"`
	IsMlo                               bool                     `json:"is_mlo,omitempty"`
	IsWired                             bool                     `json:"is_wired,omitempty"`
	LastConnectionNetworkId             string                   `json:"last_connection_network_id,omitempty"`
	LastConnectionNetworkName           string                   `json:"last_connection_network_name,omitempty"`
	LastIP                              string                   `json:"last_ip,omitempty"`
	LastIpv6                            []string                 `json:"last_ipv6,omitempty"`
	LastRadio                           string                   `json:"last_radio,omitempty"`
	LastSeen                            *int64                   `json:"last_seen,omitempty"`
	LastUplinkMac                       string                   `json:"last_uplink_mac,omitempty"`
	LastUplinkRemotePort                *int64                   `json:"last_uplink_remote_port,omitempty"`
	LastUplinkName                      string                   `json:"last_uplink_name,omitempty"`
	LatestAssocTime                     *int64                   `json:"latest_assoc_time,omitempty"`
	LocalDNSRecord                      string                   `json:"local_dns_record,omitempty"`
	LocalDNSRecordEnabled               bool                     `json:"local_dns_record_enabled,omitempty"`
	Mac                                 string                   `json:"mac,omitempty"`
	Mimo                                string                   `json:"mimo,omitempty"`
	ModelName                           string                   `json:"model_name,omitempty"`
	Name                                string                   `json:"name,omitempty"`
	NetworkId                           string                   `json:"network_id,omitempty"`
	NetworkName                         string                   `json:"network_name,omitempty"`
	Noise                               *int64                   `json:"noise,omitempty"`
	Noted                               bool                     `json:"noted,omitempty"`
	Oui                                 string                   `json:"oui,omitempty"`
	PowersaveEnabled                    bool                     `json:"powersave_enabled,omitempty"`
	Radio                               string                   `json:"radio,omitempty"`
	RadioName                           string                   `json:"radio_name,omitempty"`
	RadioProto                          string                   `json:"radio_proto,omitempty"`
	RateImbalance                       *int64                   `json:"rate_imbalance,omitempty"`
	Rssi                                *int64                   `json:"rssi,omitempty"`
	RxBytes                             *int64                   `json:"rx_bytes,omitempty"`
	RxBytesR                            *int64                   `json:"rx_bytes-r,omitempty"`
	RxPackets                           *int64                   `json:"rx_packets,omitempty"`
	RxRate                              *int64                   `json:"rx_rate,omitempty"`
	Signal                              *int64                   `json:"signal,omitempty"`
	SiteId                              string                   `json:"site_id,omitempty"`
	Status                              string                   `json:"status,omitempty"`
	SwPort                              *int64                   `json:"sw_port,omitempty"`
	Tags                                []string                 `json:"tags,omitempty"`
	TxBytes                             *int64                   `json:"tx_bytes,omitempty"`
	TxBytesR                            *int64                   `json:"tx_bytes-r,omitempty"`
	TxMcsIndex                          *int64                   `json:"tx_mcs_index,omitempty"`
	TxPackets                           *int64                   `json:"tx_packets,omitempty"`
	TxRate                              *int64                   `json:"tx_rate,omitempty"`
	Type                                string                   `json:"type,omitempty"`
	UnifiDevice                         bool                     `json:"unifi_device,omitempty"`
	UnifiDeviceInfo                     *ClientInfoDeviceInfo    `json:"unifi_device_info,omitempty"`
	UplinkMac                           string                   `json:"uplink_mac,omitempty"`
	Uptime                              *int64                   `json:"uptime,omitempty"`
	UseFixedip                          bool                     `json:"use_fixedip,omitempty"`
	UsergroupId                         string                   `json:"usergroup_id,omitempty"`
	UserId                              string                   `json:"user_id,omitempty"`
	VirtualNetworkOverrideEnabled       bool                     `json:"virtual_network_override_enabled,omitempty"`
	VirtualNetworkOverrideId            string                   `json:"virtual_network_override_id,omitempty"`
	WifiExperienceAverage               *int64                   `json:"wifi_experience_average,omitempty"`
	WifiExperienceScore                 *int64                   `json:"wifi_experience_score,omitempty"`
	WifiTxAttempts                      *int64                   `json:"wifi_tx_attempts,omitempty"`
	WifiTxRetriesPercentage             float64                  `json:"wifi_tx_retries_percentage,omitempty"`
	WiredRateMbps                       *int64                   `json:"wired_rate_mbps,omitempty"`
	WlanconfId                          string                   `json:"wlanconf_id,omitempty"`
}

type ClientList []ClientInfo

func (c *ApiClient) ListClientInfo(ctx context.Context, site string) (ClientList, error) {
	var respBody []ClientInfo

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/clients/active", site),
		nil,
		&respBody,
		map[string]string{
			"includeUnifiDevices": "true",
		},
	)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}

func (c *ApiClient) GetClientInfo(ctx context.Context, site string, mac string) (*ClientInfo, error) {
	var respBody ClientInfo

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/clients/local/%s", site, mac),
		nil,
		&respBody,
		map[string]string{
			"includeUnifiDevices": "true",
		},
	)
	if err != nil {
		return nil, err
	}

	return &respBody, nil
}

// ListClientHistory returns all historical clients, including offline devices.
// The withinHours parameter controls how far back to look (0 = all time).
func (c *ApiClient) ListClientHistory(ctx context.Context, site string, withinHours int) ([]ClientInfo, error) {
	var respBody []ClientInfo

	err := c.do(
		ctx,
		http.MethodGet,
		fmt.Sprintf("v2/api/site/%s/clients/history", site),
		nil,
		&respBody,
		map[string]string{
			"includeUnifiDevices": "true",
			"onlyNonBlocked":      "false",
			"withinHours":         fmt.Sprintf("%d", withinHours),
		},
	)
	if err != nil {
		return nil, err
	}

	return respBody, nil
}
