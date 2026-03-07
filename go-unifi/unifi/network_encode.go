package unifi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/netip"
)

const (
	PurposeCorporate = "corporate"
	PurposeWAN       = "wan"
	PurposeSiteVPN   = "site-vpn"
	PurposeVPNClient = "vpn-client"
	PurposeUserVPN   = "remote-user-vpn"
)

// MarshalJSON implements custom JSON marshaling that only includes fields relevant to the network's Purpose.
func (n *Network) MarshalJSON() ([]byte, error) {
	switch n.Purpose {
	case PurposeWAN:
		return n.marshalWAN()
	case PurposeCorporate:
		return n.marshalCorporate()
	case PurposeSiteVPN:
		return n.marshalSiteVPN()
	case PurposeVPNClient:
		return n.marshalVPNClient()
	case PurposeUserVPN:
		return n.marshalUserVPN()
	default:
		return nil, fmt.Errorf("unknown network purpose: %s", n.Purpose)
	}
}

// marshalCorporate marshals a Corporate/LAN network using the alias pattern.
func (n *Network) marshalCorporate() ([]byte, error) {
	// Calculate DHCP range defaults if needed
	var defaultStart, defaultEnd string
	if n.IPSubnet != nil {
		var err error
		defaultStart, defaultEnd, err = dhcpRange(*n.IPSubnet)
		if err != nil {
			log.Default().Printf("error calculating DHCP range: %s", err)
		}
	}

	// Use anonymous struct with explicit field selection
	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name                    *string                         `json:"name,omitempty"`
		Purpose                 string                          `json:"purpose"`
		Enabled                 bool                            `json:"enabled"`
		NetworkGroup            *string                         `json:"networkgroup,omitempty"`
		IPSubnet                *string                         `json:"ip_subnet,omitempty"`
		VLAN                    *int64                          `json:"vlan,omitempty"`
		VLANEnabled             bool                            `json:"vlan_enabled"`
		DomainName              *string                         `json:"domain_name,omitempty"`
		AutoScaleEnabled        bool                            `json:"auto_scale_enabled"`
		GatewayType             *string                         `json:"gateway_type,omitempty"`
		InternetAccessEnabled   bool                            `json:"internet_access_enabled"`
		NetworkIsolationEnabled bool                            `json:"network_isolation_enabled"`
		SettingPreference       *string                         `json:"setting_preference,omitempty"`
		IGMPSnooping            bool                            `json:"igmp_snooping"`
		DHCPguardEnabled        bool                            `json:"dhcpguard_enabled"`
		MdnsEnabled             bool                            `json:"mdns_enabled"`
		LteLanEnabled           bool                            `json:"lte_lan_enabled"`
		IPAliases               []string                        `json:"ip_aliases"`
		NATOutboundIPAddresses  []NetworkNATOutboundIPAddresses `json:"nat_outbound_ip_addresses"`
		MACOverride             string                          `json:"mac_override,omitempty"`

		// DHCP Server
		DHCPDEnabled           bool    `json:"dhcpd_enabled"`
		DHCPDStart             *string `json:"dhcpd_start,omitempty"`
		DHCPDStop              *string `json:"dhcpd_stop,omitempty"`
		DHCPDLeaseTime         *int64  `json:"dhcpd_leasetime,omitempty"`
		DHCPDDNSEnabled        bool    `json:"dhcpd_dns_enabled"`
		DHCPDDNS1              string  `json:"dhcpd_dns_1,omitempty"`
		DHCPDDNS2              string  `json:"dhcpd_dns_2,omitempty"`
		DHCPDDNS3              string  `json:"dhcpd_dns_3,omitempty"`
		DHCPDDNS4              string  `json:"dhcpd_dns_4,omitempty"`
		DHCPDGatewayEnabled    bool    `json:"dhcpd_gateway_enabled"`
		DHCPDGateway           *string `json:"dhcpd_gateway,omitempty"`
		DHCPDNtpEnabled        bool    `json:"dhcpd_ntp_enabled"`
		DHCPDNtp1              *string `json:"dhcpd_ntp_1,omitempty"`
		DHCPDNtp2              *string `json:"dhcpd_ntp_2,omitempty"`
		DHCPDWinsEnabled       bool    `json:"dhcpd_wins_enabled"`
		DHCPDWins1             *string `json:"dhcpd_wins_1,omitempty"`
		DHCPDWins2             *string `json:"dhcpd_wins_2,omitempty"`
		DHCPDTimeOffsetEnabled bool    `json:"dhcpd_time_offset_enabled"`
		DHCPDConflictChecking  bool    `json:"dhcpd_conflict_checking"`
		DHCPDBootEnabled       bool    `json:"dhcpd_boot_enabled"`
		DHCPDBootServer        string  `json:"dhcpd_boot_server,omitempty"`
		DHCPDBootFilename      string  `json:"dhcpd_boot_filename,omitempty"`
		DHCPDTFTPServer        *string `json:"dhcpd_tftp_server,omitempty"`
		DHCPDWPAdUrl           *string `json:"dhcpd_wpad_url,omitempty"`
		DHCPDUnifiController   *string `json:"dhcpd_unifi_controller,omitempty"`

		// DHCP Relay
		DHCPRelayEnabled bool     `json:"dhcp_relay_enabled"`
		DHCPRelayServers []string `json:"dhcp_relay_servers"`

		// IPv6
		IPV6InterfaceType     *string `json:"ipv6_interface_type,omitempty"`
		IPV6SettingPreference *string `json:"ipv6_setting_preference,omitempty"`
		IPV6RaPriority        *string `json:"ipv6_ra_priority,omitempty"`

		// DHCPv6
		DHCPDV6DNSAuto    bool    `json:"dhcpdv6_dns_auto,omitempty"`
		DHCPDV6AllowSlaac bool    `json:"dhcpdv6_allow_slaac,omitempty"`
		DHCPDV6Start      *string `json:"dhcpdv6_start,omitempty"`
		DHCPDV6Stop       *string `json:"dhcpdv6_stop,omitempty"`
		DHCPDV6LeaseTime  *int64  `json:"dhcpdv6_leasetime,omitempty"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:                    n.Name,
		Purpose:                 n.Purpose,
		Enabled:                 n.Enabled,
		NetworkGroup:            valueOrDefault(n.NetworkGroup, "LAN"),
		IPSubnet:                valueOrDefault(n.IPSubnet, ""),
		VLAN:                    n.VLAN,
		VLANEnabled:             n.VLANEnabled,
		DomainName:              valueOrDefault(n.DomainName, ""),
		AutoScaleEnabled:        n.AutoScaleEnabled,
		GatewayType:             valueOrDefault(n.GatewayType, "default"),
		InternetAccessEnabled:   n.InternetAccessEnabled,
		NetworkIsolationEnabled: n.NetworkIsolationEnabled,
		SettingPreference:       valueOrDefault(n.SettingPreference, "auto"),
		IGMPSnooping:            n.IGMPSnooping,
		DHCPguardEnabled:        n.DHCPguardEnabled,
		MdnsEnabled:             n.MdnsEnabled,
		LteLanEnabled:           n.LteLanEnabled,
		IPAliases:               orEmptySlice(n.IPAliases),
		NATOutboundIPAddresses:  orEmptyNATSlice(n.NATOutboundIPAddresses),
		MACOverride:             n.MACOverride,

		// DHCP Server with defaults
		DHCPDEnabled:           n.DHCPDEnabled,
		DHCPDStart:             valueOrDefault(n.DHCPDStart, defaultStart),
		DHCPDStop:              valueOrDefault(n.DHCPDStop, defaultEnd),
		DHCPDLeaseTime:         valueOrDefault(n.DHCPDLeaseTime, 86400),
		DHCPDDNSEnabled:        n.DHCPDDNSEnabled,
		DHCPDDNS1:              n.DHCPDDNS1,
		DHCPDDNS2:              n.DHCPDDNS2,
		DHCPDDNS3:              n.DHCPDDNS3,
		DHCPDDNS4:              n.DHCPDDNS4,
		DHCPDGatewayEnabled:    n.DHCPDGatewayEnabled,
		DHCPDGateway:           n.DHCPDGateway,
		DHCPDNtpEnabled:        n.DHCPDNtpEnabled,
		DHCPDNtp1:              nilIfEmpty(n.DHCPDNtp1),
		DHCPDNtp2:              nilIfEmpty(n.DHCPDNtp2),
		DHCPDWinsEnabled:       n.DHCPDWinsEnabled,
		DHCPDWins1:             valueOrDefault(n.DHCPDWins1, ""),
		DHCPDWins2:             valueOrDefault(n.DHCPDWins2, ""),
		DHCPDTimeOffsetEnabled: n.DHCPDTimeOffsetEnabled,
		DHCPDConflictChecking:  n.DHCPDConflictChecking,
		DHCPDBootEnabled:       n.DHCPDBootEnabled,
		DHCPDBootServer:        n.DHCPDBootServer,
		DHCPDBootFilename:      derefOrEmpty(n.DHCPDBootFilename),
		DHCPDTFTPServer:        n.DHCPDTFTPServer,
		DHCPDWPAdUrl:           n.DHCPDWPAdUrl,
		DHCPDUnifiController:   valueOrDefault(n.DHCPDUnifiController, ""),

		// DHCP Relay
		DHCPRelayEnabled: n.DHCPRelayEnabled,
		DHCPRelayServers: orEmptySlice(n.RemoteVPNSubnets),

		// IPv6
		IPV6InterfaceType:     valueOrDefault(n.IPV6InterfaceType, "none"),
		IPV6SettingPreference: n.IPV6SettingPreference,
		IPV6RaPriority:        n.IPV6RaPriority,

		// DHCPv6
		DHCPDV6DNSAuto:    n.DHCPDV6DNSAuto,
		DHCPDV6AllowSlaac: n.DHCPDV6AllowSlaac,
		DHCPDV6Start:      n.DHCPDV6Start,
		DHCPDV6Stop:       n.DHCPDV6Stop,
		DHCPDV6LeaseTime:  n.DHCPDV6LeaseTime,
	})
}

// marshalWAN marshals a WAN network.
func (n *Network) marshalWAN() ([]byte, error) {
	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name    *string `json:"name,omitempty"`
		Purpose string  `json:"purpose"`
		Enabled bool    `json:"enabled"`

		// WAN-specific fields
		WANType             *string                 `json:"wan_type,omitempty"`
		WANNetworkGroup     *string                 `json:"wan_networkgroup,omitempty"`
		WANVLANEnabled      bool                    `json:"wan_vlan_enabled"`
		WANVLAN             *int64                  `json:"wan_vlan,omitempty"`
		WANFailoverPriority *int64                  `json:"wan_failover_priority,omitempty"`
		ReportWANEvent      bool                    `json:"report_wan_event"`
		IGMPProxyUpstream   bool                    `json:"igmp_proxy_upstream"`
		WANDHCPv6PDSizeAuto bool                    `json:"wan_dhcpv6_pd_size_auto"`
		WANIPAliases        []string                `json:"wan_ip_aliases"`
		WANDHCPOptions      []NetworkWANDHCPOptions `json:"wan_dhcp_options"`
		IPV6Enabled         bool                    `json:"ipv6_enabled"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:    n.Name,
		Purpose: n.Purpose,
		Enabled: n.Enabled,

		// WAN fields
		WANType:             n.WANType,
		WANNetworkGroup:     n.WANNetworkGroup,
		WANVLANEnabled:      n.WANVLANEnabled,
		WANVLAN:             n.WANVLAN,
		WANFailoverPriority: n.WANFailoverPriority,
		ReportWANEvent:      n.ReportWANEvent,
		IGMPProxyUpstream:   n.IGMPProxyUpstream,
		WANDHCPv6PDSizeAuto: n.WANDHCPv6PDSizeAuto,
		WANIPAliases:        orEmptySlice(n.WANIPAliases),
		WANDHCPOptions:      orEmptyWANDHCPOptions(n.WANDHCPOptions),
		IPV6Enabled:         true, // Default to true for WAN
	})
}

// marshalSiteVPN marshals a site-to-site VPN network.
func (n *Network) marshalSiteVPN() ([]byte, error) {
	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name    *string `json:"name,omitempty"`
		Purpose string  `json:"purpose"`
		Enabled bool    `json:"enabled"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:    n.Name,
		Purpose: n.Purpose,
		Enabled: n.Enabled,
	})
}

// marshalVPNClient marshals a VPN client network (WireGuard client).
func (n *Network) marshalVPNClient() ([]byte, error) {
	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name     *string `json:"name,omitempty"`
		Purpose  string  `json:"purpose"`
		Enabled  bool    `json:"enabled"`
		IPSubnet *string `json:"ip_subnet,omitempty"`

		// VPN Type
		VPNType *string `json:"vpn_type,omitempty"`

		// VPN Client routing
		VPNClientDefaultRoute bool `json:"vpn_client_default_route"`
		VPNClientPullDNS      bool `json:"vpn_client_pull_dns"`

		// WireGuard Client Configuration
		WireguardClientMode                  *string `json:"wireguard_client_mode,omitempty"`
		WireguardClientConfigurationFile     *string `json:"wireguard_client_configuration_file,omitempty"`
		WireguardClientConfigurationFilename *string `json:"wireguard_client_configuration_filename,omitempty"`
		WireguardClientPeerIP                *string `json:"wireguard_client_peer_ip,omitempty"`
		WireguardClientPeerPort              *int64  `json:"wireguard_client_peer_port,omitempty"`
		WireguardClientPeerPublicKey         *string `json:"wireguard_client_peer_public_key,omitempty"`
		WireguardClientPresharedKeyEnabled   bool    `json:"wireguard_client_preshared_key_enabled"`
		WireguardClientPresharedKey          *string `json:"wireguard_client_preshared_key,omitempty"`
		WireguardInterface                   *string `json:"wireguard_interface,omitempty"`
		WireguardPrivateKey                  *string `json:"x_wireguard_private_key,omitempty"`

		// DNS servers for WireGuard interface
		DHCPDDNS1 string `json:"dhcpd_dns_1,omitempty"`
		DHCPDDNS2 string `json:"dhcpd_dns_2,omitempty"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:     n.Name,
		Purpose:  n.Purpose,
		Enabled:  n.Enabled,
		IPSubnet: n.IPSubnet,

		// VPN Type
		VPNType: n.VPNType,

		// VPN Client routing
		VPNClientDefaultRoute: n.VPNClientDefaultRoute,
		VPNClientPullDNS:      n.VPNClientPullDNS,

		// WireGuard configuration
		WireguardClientMode:                  n.WireguardClientMode,
		WireguardClientConfigurationFile:     n.WireguardClientConfigurationFile,
		WireguardClientConfigurationFilename: n.WireguardClientConfigurationFilename,
		WireguardClientPeerIP:                n.WireguardClientPeerIP,
		WireguardClientPeerPort:              n.WireguardClientPeerPort,
		WireguardClientPeerPublicKey:         n.WireguardClientPeerPublicKey,
		WireguardClientPresharedKeyEnabled:   n.WireguardClientPresharedKeyEnabled,
		WireguardClientPresharedKey:          n.WireguardClientPresharedKey,
		WireguardInterface:                   n.WireguardInterface,
		WireguardPrivateKey:                  n.WireguardPrivateKey,

		// DNS servers
		DHCPDDNS1: n.DHCPDDNS1,
		DHCPDDNS2: n.DHCPDDNS2,
	})
}

// marshalUserVPN marshals a remote user VPN network.
func (n *Network) marshalUserVPN() ([]byte, error) {
	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name    *string `json:"name,omitempty"`
		Purpose string  `json:"purpose"`
		Enabled bool    `json:"enabled"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:    n.Name,
		Purpose: n.Purpose,
		Enabled: n.Enabled,
	})
}

// Helper functions for field transformations

func orEmptySlice(s []string) []string {
	if len(s) > 0 {
		return s
	}
	return []string{}
}

func orEmptyNATSlice(s []NetworkNATOutboundIPAddresses) []NetworkNATOutboundIPAddresses {
	if len(s) > 0 {
		return s
	}
	return []NetworkNATOutboundIPAddresses{}
}

func orEmptyWANDHCPOptions(s []NetworkWANDHCPOptions) []NetworkWANDHCPOptions {
	if len(s) > 0 {
		return s
	}
	return []NetworkWANDHCPOptions{}
}

func nilIfEmpty(s *string) *string {
	if s != nil && *s == "" {
		return nil
	}
	return s
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func dhcpRange(cidr string) (start, end string, err error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return "", "", err
	}

	// Only support IPv4
	if !prefix.Addr().Is4() {
		return "", "", fmt.Errorf("only IPv4 supported")
	}

	networkAddr := prefix.Masked().Addr()
	bits := prefix.Bits()

	// Calculate the number of host addresses
	hostBits := 32 - bits
	numHosts := uint32(1) << hostBits

	// UniFi's rules based on subnet size:
	// /30 or smaller (4 or fewer IPs): No DHCP (too small)
	// /29 (8 IPs): Start at +2, End at -2 (gives 4 usable IPs)
	// /28 to /24: Start at +6, End at -1 (broadcast)
	// /23 and larger: Start at +6, End at -1

	if bits >= 30 {
		return "", "", fmt.Errorf("subnet too small for DHCP (/%d)", bits)
	}

	// Convert network address to uint32 for arithmetic
	ip4 := networkAddr.As4()
	baseIP := uint32(ip4[0])<<24 | uint32(ip4[1])<<16 | uint32(ip4[2])<<8 | uint32(ip4[3])

	var startOffset, endOffset uint32

	if bits == 29 {
		// /29: 8 IPs total
		// Network: .0, Gateway: .1, DHCP: .2-.5, Reserved: .6, Broadcast: .7
		startOffset = 2
		endOffset = 2
	} else {
		// /28 and larger
		// Network: .0, Gateway: .1, Reserved: .2-.5, DHCP: .6 to (broadcast-1)
		startOffset = 6
		endOffset = 1
	}

	startIP := baseIP + startOffset
	endIP := baseIP + numHosts - 1 - endOffset

	// Convert back to netip.Addr
	start = netip.AddrFrom4([4]byte{
		byte(startIP >> 24),
		byte(startIP >> 16),
		byte(startIP >> 8),
		byte(startIP),
	}).String()

	end = netip.AddrFrom4([4]byte{
		byte(endIP >> 24),
		byte(endIP >> 16),
		byte(endIP >> 8),
		byte(endIP),
	}).String()

	return start, end, nil
}

func valueOrDefault[T any](in *T, defaultValue T) *T {
	if in == nil {
		return &defaultValue
	}
	return in
}
