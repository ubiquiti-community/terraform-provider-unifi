package unifi

import (
	"encoding/json"
	"testing"
)

// Helper function to parse JSON and check for expected/unexpected fields.
func checkJSONFields(t *testing.T, data []byte, expectedFields []string, unexpectedFields []string) {
	t.Helper()

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check expected fields are present
	for _, field := range expectedFields {
		if _, ok := result[field]; !ok {
			t.Errorf("Expected field %q not found in JSON", field)
		}
	}

	// Check unexpected fields are absent
	for _, field := range unexpectedFields {
		if _, ok := result[field]; ok {
			t.Errorf("Unexpected field %q found in JSON", field)
		}
	}
}

func TestMarshalNetworkCorporate(t *testing.T) {
	// Create a corporate network with common fields
	vlan := int64(10)
	leasetime := int64(86400)
	dhcpGateway := "192.168.1.1"
	dhcpStart := "192.168.1.100"
	dhcpStop := "192.168.1.200"

	network := &Network{
		ID:                    "507f1f77bcf86cd799439011",
		SiteID:                "default",
		Name:                  strPtr("Corporate LAN"),
		Purpose:               PurposeCorporate,
		Enabled:               true,
		AutoScaleEnabled:      false,
		NetworkGroup:          strPtr("LAN"),
		IPSubnet:              strPtr("192.168.1.0/24"),
		VLAN:                  &vlan,
		VLANEnabled:           true,
		DomainName:            strPtr("example.local"),
		GatewayType:           strPtr("default"),
		DHCPDGateway:          &dhcpGateway,
		DHCPDGatewayEnabled:   true,
		InternetAccessEnabled: true,
		MdnsEnabled:           true,
		IGMPSnooping:          false,
		DHCPDEnabled:          true,
		DHCPDStart:            &dhcpStart,
		DHCPDStop:             &dhcpStop,
		DHCPDLeaseTime:        &leasetime,
		DHCPDDNS1:             "8.8.8.8",
		DHCPDDNS2:             "8.8.4.4",
		DHCPDDNSEnabled:       true,
		IPAliases:             []string{},
	}

	// Marshal to JSON
	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal network: %v", err)
	}

	// Expected fields for corporate network
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
		"domain_name",
		"gateway_type",
		"dhcpd_gateway",
		"dhcpd_gateway_enabled",
		"internet_access_enabled",
		"mdns_enabled",
		"igmp_snooping",
		"dhcpd_enabled",
		"dhcpd_start",
		"dhcpd_stop",
		"dhcpd_leasetime",
		"dhcpd_dns_1",
		"dhcpd_dns_2",
		"dhcpd_dns_enabled",
		"ip_aliases",
		"auto_scale_enabled",
		"setting_preference",
	}

	// Unexpected fields (WAN-specific)
	unexpectedFields := []string{
		"wan_type",
		"wan_ip",
		"wan_networkgroup",
		"ipsec_key_exchange",
		"wireguard_interface",
	}

	checkJSONFields(t, data, expectedFields, unexpectedFields)

	// Verify purpose is correct
	var result map[string]any
	json.Unmarshal(data, &result)
	if result["purpose"] != string(PurposeCorporate) {
		t.Errorf("Expected purpose %q, got %q", PurposeCorporate, result["purpose"])
	}

	// Verify default values are applied
	if result["networkgroup"] != "LAN" {
		t.Errorf("Expected networkgroup 'LAN', got %q", result["networkgroup"])
	}
	if result["gateway_type"] != "default" {
		t.Errorf("Expected gateway_type 'default', got %q", result["gateway_type"])
	}
	if result["setting_preference"] != "auto" {
		t.Errorf("Expected setting_preference 'auto', got %q", result["setting_preference"])
	}
}

func TestMarshalNetworkCorporateDefaults(t *testing.T) {
	// Create a minimal corporate network to test defaults
	network := &Network{
		ID:      "507f1f77bcf86cd799439011",
		Purpose: PurposeCorporate,
		Enabled: true,
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal network: %v", err)
	}

	var result map[string]any
	json.Unmarshal(data, &result)

	// Verify defaults are applied
	if result["networkgroup"] != "LAN" {
		t.Errorf("Expected default networkgroup 'LAN', got %v", result["networkgroup"])
	}
	if result["gateway_type"] != "default" {
		t.Errorf("Expected default gateway_type 'default', got %v", result["gateway_type"])
	}
	if result["setting_preference"] != "auto" {
		t.Errorf("Expected default setting_preference 'auto', got %v", result["setting_preference"])
	}
	if result["ip_subnet"] != "" {
		t.Errorf("Expected empty ip_subnet, got %v", result["ip_subnet"])
	}

	// Verify empty arrays are empty, not nil
	if aliases, ok := result["ip_aliases"].([]any); !ok || len(aliases) != 0 {
		t.Errorf("Expected empty array for ip_aliases, got %v", result["ip_aliases"])
	}
}

func TestMarshalNetworkWAN(t *testing.T) {
	vlan := int64(20)
	failoverPriority := int64(1)

	network := &Network{
		ID:                  "507f1f77bcf86cd799439012",
		SiteID:              "default",
		Name:                strPtr("WAN"),
		Purpose:             PurposeWAN,
		Enabled:             true,
		WANType:             strPtr("dhcp"),
		WANNetworkGroup:     strPtr("WAN"),
		WANVLANEnabled:      true,
		WANVLAN:             &vlan,
		WANFailoverPriority: &failoverPriority,
		ReportWANEvent:      true,
		IGMPProxyUpstream:   false,
		WANDHCPv6PDSizeAuto: true,
		WANIPAliases:        []string{},
		WANDHCPOptions:      []NetworkWANDHCPOptions{},
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal network: %v", err)
	}

	expectedFields := []string{
		"_id",
		"site_id",
		"name",
		"purpose",
		"enabled",
		"wan_type",
		"wan_networkgroup",
		"wan_vlan_enabled",
		"wan_vlan",
		"wan_failover_priority",
		"report_wan_event",
		"igmp_proxy_upstream",
		"wan_dhcpv6_pd_size_auto",
		"wan_ip_aliases",
		"wan_dhcp_options",
		"ipv6_enabled",
	}

	unexpectedFields := []string{
		"networkgroup",
		"ip_subnet",
		"vlan",
		"dhcpd_enabled",
		"ipsec_interface",
		"wireguard_interface",
	}

	checkJSONFields(t, data, expectedFields, unexpectedFields)

	var result map[string]any
	json.Unmarshal(data, &result)

	// Verify WAN-specific values
	if result["purpose"] != string(PurposeWAN) {
		t.Errorf("Expected purpose %q, got %q", PurposeWAN, result["purpose"])
	}
	if result["ipv6_enabled"] != true {
		t.Errorf("Expected ipv6_enabled true, got %v", result["ipv6_enabled"])
	}

	// Verify empty arrays
	if aliases, ok := result["wan_ip_aliases"].([]any); !ok || len(aliases) != 0 {
		t.Errorf("Expected empty array for wan_ip_aliases, got %v", result["wan_ip_aliases"])
	}
}

func TestMarshalNetworkUnknownPurpose(t *testing.T) {
	network := &Network{
		ID:      "507f1f77bcf86cd799439016",
		Purpose: "unknown-purpose",
		Enabled: true,
	}

	_, err := json.Marshal(network)
	if err == nil {
		t.Error("Expected error for unknown purpose, got nil")
	}
}

// Helper function to create string pointers.
func strPtr(s string) *string {
	return &s
}
