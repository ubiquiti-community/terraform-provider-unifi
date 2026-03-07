# go-unifi Client Library: Network Purpose Fix

## Repository

- **Repo**: `ubiquiti-community/go-unifi`
- **File**: `unifi/network_encode.go`
- **Test File**: `unifi/network_encode_test.go`

## Problem

The `Network.MarshalJSON()` method in `unifi/network_encode.go` is missing case handlers for `vlan-only` and `guest` network purposes. This causes the error:

```
json: error calling MarshalJSON for type *unifi.Network: unknown network purpose: vlan-only
```

The `MarshalJSON` switch statement only handles: `corporate`, `wan`, `site-vpn`, `vpn-client`, `remote-user-vpn`. It needs to also handle `vlan-only` and `guest`.

## Required Changes

### 1. Add Purpose Constants

In `unifi/network_encode.go`, add `PurposeGuest` and `PurposeVLANOnly` to the constants block:

```go
const (
	PurposeCorporate = "corporate"
	PurposeGuest     = "guest"
	PurposeVLANOnly  = "vlan-only"
	PurposeWAN       = "wan"
	PurposeSiteVPN   = "site-vpn"
	PurposeVPNClient = "vpn-client"
	PurposeUserVPN   = "remote-user-vpn"
)
```

### 2. Add Switch Cases in MarshalJSON

Add cases for the new purposes in the `MarshalJSON` switch:

```go
func (n *Network) MarshalJSON() ([]byte, error) {
	switch n.Purpose {
	case PurposeWAN:
		return n.marshalWAN()
	case PurposeCorporate:
		return n.marshalCorporate()
	case PurposeGuest:
		return n.marshalGuest()
	case PurposeVLANOnly:
		return n.marshalVLANOnly()
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
```

### 3. Add marshalVLANOnly Method

VLAN-only networks are Layer 2 only (no routing). They have a minimal field set. Add this method after `marshalCorporate()`:

```go
// marshalVLANOnly marshals a VLAN-only network (Layer 2 only, no routing).
func (n *Network) marshalVLANOnly() ([]byte, error) {
	enabled := n.Enabled
	if !enabled {
		enabled = true
	}

	vlanEnabled := n.VLANEnabled
	if !vlanEnabled && n.VLAN != nil && *n.VLAN > 0 {
		vlanEnabled = true
	}

	return json.Marshal(&struct {
		ID       string `json:"_id,omitempty"`
		SiteID   string `json:"site_id,omitempty"`
		Hidden   bool   `json:"attr_hidden,omitempty"`
		HiddenID string `json:"attr_hidden_id,omitempty"`
		NoDelete bool   `json:"attr_no_delete,omitempty"`
		NoEdit   bool   `json:"attr_no_edit,omitempty"`

		Name         *string `json:"name,omitempty"`
		Purpose      string  `json:"purpose"`
		Enabled      bool    `json:"enabled"`
		NetworkGroup *string `json:"networkgroup,omitempty"`
		VLAN         *int64  `json:"vlan,omitempty"`
		VLANEnabled  bool    `json:"vlan_enabled"`
	}{
		ID:       n.ID,
		SiteID:   n.SiteID,
		Hidden:   n.Hidden,
		HiddenID: n.HiddenID,
		NoDelete: n.NoDelete,
		NoEdit:   n.NoEdit,

		Name:         n.Name,
		Purpose:      n.Purpose,
		Enabled:      enabled,
		NetworkGroup: valueOrDefault(n.NetworkGroup, "LAN"),
		VLAN:         n.VLAN,
		VLANEnabled:  vlanEnabled,
	})
}
```

### 4. Add marshalGuest Method

Guest networks have the same field structure as corporate networks (DHCP, IPv6, relay, etc.). Add this method after `marshalVLANOnly()`:

```go
// marshalGuest marshals a Guest network.
func (n *Network) marshalGuest() ([]byte, error) {
	var defaultStart, defaultEnd string
	if n.IPSubnet != nil {
		var err error
		defaultStart, defaultEnd, err = dhcpRange(*n.IPSubnet)
		if err != nil {
			log.Default().Printf("error calculating DHCP range: %s", err)
		}
	}

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
```

### 5. Add Tests

Add these tests to `unifi/network_encode_test.go`:

```go
func TestMarshalNetworkVLANOnly(t *testing.T) {
	vlan := int64(92)

	network := &Network{
		ID:      "507f1f77bcf86cd799439017",
		SiteID:  "default",
		Name:    strPtr("VLAN_92"),
		Purpose: PurposeVLANOnly,
		VLAN:    &vlan,
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal vlan-only network: %v", err)
	}

	expectedFields := []string{
		"_id",
		"site_id",
		"name",
		"purpose",
		"enabled",
		"networkgroup",
		"vlan",
		"vlan_enabled",
	}

	unexpectedFields := []string{
		"ip_subnet",
		"dhcpd_enabled",
		"wan_type",
		"wireguard_interface",
		"igmp_snooping",
	}

	checkJSONFields(t, data, expectedFields, unexpectedFields)

	var result map[string]any
	json.Unmarshal(data, &result)

	if result["purpose"] != "vlan-only" {
		t.Errorf("Expected purpose 'vlan-only', got %q", result["purpose"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled true (default), got %v", result["enabled"])
	}
	if result["vlan_enabled"] != true {
		t.Errorf("Expected vlan_enabled true (auto-set from VLAN ID), got %v", result["vlan_enabled"])
	}
	if result["networkgroup"] != "LAN" {
		t.Errorf("Expected networkgroup 'LAN', got %v", result["networkgroup"])
	}
}

func TestMarshalNetworkVLANOnlyMinimal(t *testing.T) {
	network := &Network{
		ID:      "507f1f77bcf86cd799439018",
		Purpose: PurposeVLANOnly,
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal minimal vlan-only network: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	if result["purpose"] != "vlan-only" {
		t.Errorf("Expected purpose 'vlan-only', got %q", result["purpose"])
	}
	if result["enabled"] != true {
		t.Errorf("Expected enabled true (default), got %v", result["enabled"])
	}
	if result["vlan_enabled"] != false {
		t.Errorf("Expected vlan_enabled false (no VLAN ID), got %v", result["vlan_enabled"])
	}
}

func TestMarshalNetworkGuest(t *testing.T) {
	vlan := int64(100)
	leasetime := int64(86400)
	dhcpStart := "192.168.100.100"
	dhcpStop := "192.168.100.200"

	network := &Network{
		ID:                    "507f1f77bcf86cd799439019",
		SiteID:                "default",
		Name:                  strPtr("Guest Network"),
		Purpose:               PurposeGuest,
		Enabled:               true,
		NetworkGroup:          strPtr("LAN"),
		IPSubnet:              strPtr("192.168.100.0/24"),
		VLAN:                  &vlan,
		VLANEnabled:           true,
		InternetAccessEnabled: true,
		DHCPDEnabled:          true,
		DHCPDStart:            &dhcpStart,
		DHCPDStop:             &dhcpStop,
		DHCPDLeaseTime:        &leasetime,
		DHCPDDNSEnabled:       true,
		DHCPDDNS1:             "8.8.8.8",
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal guest network: %v", err)
	}

	expectedFields := []string{
		"_id",
		"site_id",
		"name",
		"purpose",
		"enabled",
		"networkgroup",
		"ip_subnet",
		"vlan",
		"vlan_enabled",
		"internet_access_enabled",
		"dhcpd_enabled",
		"dhcpd_start",
		"dhcpd_stop",
		"dhcpd_leasetime",
		"dhcpd_dns_enabled",
		"dhcpd_dns_1",
		"ip_aliases",
		"setting_preference",
	}

	unexpectedFields := []string{
		"wan_type",
		"wan_networkgroup",
		"wireguard_interface",
	}

	checkJSONFields(t, data, expectedFields, unexpectedFields)

	var result map[string]any
	json.Unmarshal(data, &result)

	if result["purpose"] != "guest" {
		t.Errorf("Expected purpose 'guest', got %q", result["purpose"])
	}
	if result["networkgroup"] != "LAN" {
		t.Errorf("Expected networkgroup 'LAN', got %v", result["networkgroup"])
	}
	if result["setting_preference"] != "auto" {
		t.Errorf("Expected setting_preference 'auto', got %v", result["setting_preference"])
	}
}
```

## Key Design Notes

- **`marshalVLANOnly`** uses a minimal field set (no DHCP, no IPv6, no routing) since VLAN-only networks are Layer 2 only
- **`marshalVLANOnly`** auto-enables the network (`Enabled = true`) and sets `VLANEnabled = true` when a VLAN ID is provided
- **`marshalGuest`** uses the same field structure as `marshalCorporate` since guest networks support the same Layer 3 features
- Both methods follow the existing anonymous-struct-with-alias pattern used by other marshal methods
- Both methods use existing helper functions: `valueOrDefault`, `orEmptySlice`, `orEmptyNATSlice`, `nilIfEmpty`, `derefOrEmpty`, `dhcpRange`

## After Fix

Once the fix is released as a new go-unifi version, update `terraform-provider-unifi/go.mod` to use the new version:

```bash
go get github.com/ubiquiti-community/go-unifi@<new-version>
go mod tidy
```

## Related Issues

- terraform-provider-unifi#90
