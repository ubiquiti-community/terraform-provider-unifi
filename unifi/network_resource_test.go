package unifi

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/querycheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func strPtr(s string) *string { return &s }

func TestAccNetworkFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"name",
						"Test VLAN",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"subnet",
						"192.168.10.1/24",
					),
					resource.TestCheckResourceAttr("unifi_network.test", "vlan", "10"),
					resource.TestCheckResourceAttr("unifi_network.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_network.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test VLAN",
				// Ignore dhcp_server and dhcp_relay since they're not configured in the test
				// but will be populated by the API with default values during import
				ImportStateVerifyIgnore: []string{
					"dhcp_server",
					"dhcp_relay",
				},
			},
		},
	})
}

// TestAccNetworkFramework_ipAliases guards #413 end-to-end: a network with
// ip_aliases configured must apply cleanly (no "provider produced inconsistent
// result after apply"), round-trip the values, survive updates, and allow
// removing the aliases again.
func TestAccNetworkFramework_ipAliases(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_ipAliases(
					`["192.168.111.1/24", "192.168.112.1/24"]`,
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_aliases",
						"ip_aliases.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_aliases",
						"ip_aliases.0",
						"192.168.111.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_aliases",
						"ip_aliases.1",
						"192.168.112.1/24",
					),
				),
			},
			{
				Config: testAccNetworkFrameworkConfig_ipAliases(`["192.168.113.1/24"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_aliases",
						"ip_aliases.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_aliases",
						"ip_aliases.0",
						"192.168.113.1/24",
					),
				),
			},
			{
				Config: testAccNetworkFrameworkConfig_basicAliasNetwork(),
				Check: resource.TestCheckNoResourceAttr(
					"unifi_network.test_aliases",
					"ip_aliases",
				),
			},
		},
	})
}

func testAccNetworkFrameworkConfig_ipAliases(aliases string) string {
	return `
resource "unifi_network" "test_aliases" {
	name       = "Test Alias VLAN"
	subnet     = "192.168.11.1/24"
	vlan       = 11
	enabled    = true
	ip_aliases = ` + aliases + `
}
`
}

func testAccNetworkFrameworkConfig_basicAliasNetwork() string {
	return `
resource "unifi_network" "test_aliases" {
	name       = "Test Alias VLAN"
	subnet     = "192.168.11.1/24"
	vlan       = 11
	enabled    = true
}
`
}

func TestAccNetworkFramework_dhcp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_dhcp(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"name",
						"Test DHCP Network",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"subnet",
						"192.168.20.1/24",
					),
					resource.TestCheckResourceAttr("unifi_network.test_dhcp", "vlan", "20"),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"dhcp_server.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"dhcp_server.start",
						"192.168.20.10",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"dhcp_server.stop",
						"192.168.20.254",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcp",
						"dhcp_server.leasetime",
						"24h0m0s",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_dhcp",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test DHCP Network",
			},
			// String import by bare controller ObjectID (the 24-hex format).
			{
				ResourceName:      "unifi_network.test_dhcp",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccNetworkFrameworkImportStateIDFunc(
					"unifi_network.test_dhcp",
				),
			},
			// Identity-based import (import block with identity, Terraform 1.12+).
			{
				ResourceName:    "unifi_network.test_dhcp",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

// testAccNetworkFrameworkImportStateIDFunc returns the bare controller
// ObjectID of the named resource for use as a string import ID.
func testAccNetworkFrameworkImportStateIDFunc(
	resourceName string,
) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceName)
		}
		return rs.Primary.ID, nil
	}
}

func TestAccNetworkFramework_guest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_guest(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_guest",
						"name",
						"Guest Network",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_guest",
						"subnet",
						"192.168.30.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_guest",
						"vlan",
						"30",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_guest",
						"internet_access",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_guest",
						"network_isolation",
						"true",
					),
				),
			},
		},
	})
}

func testAccNetworkFrameworkConfig_basic() string {
	return `
resource "unifi_network" "test" {
	name      = "Test VLAN"
	subnet    = "192.168.10.1/24"
	vlan      = 10
	enabled   = true
}
`
}

func testAccNetworkFrameworkConfig_dhcp() string {
	return `
resource "unifi_network" "test_dhcp" {
	name      = "Test DHCP Network"
	subnet    = "192.168.20.1/24"
	vlan      = 20

	dhcp_server = {
		enabled   = true
		start     = "192.168.20.10"
		stop      = "192.168.20.254"
		leasetime = "24h0m0s"
	}
}
`
}

func testAccNetworkFrameworkConfig_guest() string {
	return `
resource "unifi_network" "test_guest" {
	name              = "Guest Network"
	subnet            = "192.168.30.1/24"
	vlan              = 30
	internet_access   = true
	network_isolation = true
}
`
}

func TestAccNetworkFramework_thirdPartyGateway(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_thirdPartyGateway(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"name",
						"Test Third Party",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"vlan",
						"3",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"third_party_gateway",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"dhcp_guarding.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"dhcp_guarding.servers.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"dhcp_guarding.servers.0",
						"192.168.20.20",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party",
						"dhcp_guarding.servers.1",
						"192.168.20.21",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_third_party",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test Third Party",
				// These fields are not relevant to vlan-only networks and are not
				// returned by the API, so they cannot be recovered during import.
				ImportStateVerifyIgnore: []string{
					"subnet",
					"auto_scale",
					"gateway_type",
					"setting_preference",
					"multicast_dns",
					"ipv6_interface_type",
					"ipv6_static_subnet",
					"ipv6_ra",
					"ipv6_ra_priority",
					"ipv6_ra_preferred_lifetime",
					"ipv6_ra_valid_lifetime",
					"ipv6_pd_interface",
					"ipv6_pd_prefixid",
					"ipv6_pd_start",
					"ipv6_pd_stop",
					"ipv6_pd_auto_prefixid_enabled",
					"lte_lan",
					"internet_access",
				},
			},
		},
	})
}

func TestAccNetworkFramework_thirdPartyGatewayMinimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_thirdPartyGatewayMinimal(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party_min",
						"name",
						"Test Third Party Minimal",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party_min",
						"vlan",
						"4",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_third_party_min",
						"third_party_gateway",
						"true",
					),
				),
			},
		},
	})
}

func testAccNetworkFrameworkConfig_thirdPartyGateway() string {
	return `
resource "unifi_network" "test_third_party" {
	name                = "Test Third Party"
	subnet              = "192.168.20.1/24"
	vlan                = 3
	third_party_gateway = true

	dhcp_guarding = {
		enabled = true
		servers = ["192.168.20.20", "192.168.20.21"]
	}
}
`
}

func testAccNetworkFrameworkConfig_thirdPartyGatewayMinimal() string {
	return `
resource "unifi_network" "test_third_party_min" {
	name                = "Test Third Party Minimal"
	subnet              = "192.168.20.1/24"
	vlan                = 4
	third_party_gateway = true
}
`
}

func TestAccNetworkFramework_dhcpRelay(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			preCheck(t)
			// Without zone-based firewall support the controller never
			// assigns firewall_zone_id, so it stays "(known after apply)"
			// and the re-plan step fails on a perpetual diff.
			testAccFirewallZonePreCheck(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_dhcpRelay(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_relay",
						"name",
						"Test DHCP Relay",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_relay",
						"vlan",
						"50",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_relay",
						"dhcp_relay.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_relay",
						"dhcp_relay.servers.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_relay",
						"dhcp_relay.servers.0",
						"192.168.50.1",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_relay",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test DHCP Relay",
				ImportStateVerifyIgnore: []string{
					"auto_scale",
					"gateway_type",
					"setting_preference",
					"multicast_dns",
					"ipv6_interface_type",
					"ipv6_static_subnet",
					"ipv6_ra",
					"ipv6_ra_priority",
					"ipv6_ra_preferred_lifetime",
					"ipv6_ra_valid_lifetime",
					"ipv6_pd_interface",
					"ipv6_pd_prefixid",
					"ipv6_pd_start",
					"ipv6_pd_stop",
					"ipv6_pd_auto_prefixid_enabled",
					"lte_lan",
					"internet_access",
				},
			},
		},
	})
}

func testAccNetworkFrameworkConfig_dhcpRelay() string {
	return `
resource "unifi_network" "test_relay" {
	name   = "Test DHCP Relay"
	subnet = "192.168.50.1/24"
	vlan   = 50

	dhcp_relay = {
		enabled = true
		servers = ["192.168.50.1"]
	}
}
`
}

func TestAccNetworkFramework_ipv6Static(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_ipv6Static(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"name",
						"Test IPv6 Static",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_interface_type",
						"static",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_static_subnet",
						"fd00::1/64",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_ra",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_ra_priority",
						"high",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_ra_valid_lifetime",
						"24h0m0s",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_ra_preferred_lifetime",
						"4h0m0s",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_ipv6_static",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test IPv6 Static",
				ImportStateVerifyIgnore: []string{
					"dhcp_server",
					"dhcp_relay",
					"dhcp_v6_server",
				},
			},
		},
	})
}

func TestAccNetworkFramework_dhcpV6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_dhcpV6(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"name",
						"Test DHCPv6",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"ipv6_interface_type",
						"static",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.dns_auto",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.dns_servers.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.dns_servers.0",
						"2001:4860:4860::8888",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.dns_servers.1",
						"2001:4860:4860::8844",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.start",
						"::2",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.stop",
						"::7d1",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_dhcpv6",
						"dhcp_v6_server.lease",
						"86400",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_dhcpv6",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test DHCPv6",
				ImportStateVerifyIgnore: []string{
					"dhcp_server",
					"dhcp_relay",
				},
			},
		},
	})
}

func testAccNetworkFrameworkConfig_ipv6Static() string {
	return `
resource "unifi_network" "test_ipv6_static" {
	name                    = "Test IPv6 Static"
	subnet                  = "192.168.40.1/24"
	vlan                    = 40
	ipv6_interface_type     = "static"
	ipv6_static_subnet      = "fd00::1/64"
	ipv6_ra                 = true
	ipv6_ra_priority        = "high"
	ipv6_ra_valid_lifetime  = "24h0m0s"
	ipv6_ra_preferred_lifetime = "4h0m0s"
}
`
}

func testAccNetworkFrameworkConfig_dhcpV6() string {
	return `
resource "unifi_network" "test_dhcpv6" {
	name                = "Test DHCPv6"
	subnet              = "192.168.60.1/24"
	vlan                = 60
	ipv6_interface_type = "static"
	ipv6_static_subnet  = "fd01::1/64"
	ipv6_ra             = true

	dhcp_v6_server = {
		enabled     = true
		dns_auto    = false
		dns_servers = ["2001:4860:4860::8888", "2001:4860:4860::8844"]
		start       = "::2"
		stop        = "::7d1"
		lease       = 86400
	}
}
`
}

func TestNewNetworkResource(t *testing.T) {
	got := NewNetworkResource()
	if got == nil {
		t.Fatal("NewNetworkResource() returned nil")
	}
}

func TestNewNetworkListResource(t *testing.T) {
	got := NewNetworkListResource()
	if got == nil {
		t.Fatal("NewNetworkListResource() returned nil")
	}
}

func Test_dhcpBootModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpBootModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    dhcpBootModel{},
			want: map[string]attr.Type{
				"enabled":  types.BoolType,
				"server":   types.StringType,
				"filename": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpBootModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_winsModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    winsModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    winsModel{},
			want: map[string]attr.Type{
				"enabled":   types.BoolType,
				"addresses": types.ListType{ElemType: types.StringType},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("winsModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpServerModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpServerModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    dhcpServerModel{},
			want: map[string]attr.Type{
				"boot": types.ObjectType{
					AttrTypes: dhcpBootModel{}.AttributeTypes(),
				},
				"enabled":             types.BoolType,
				"start":               types.StringType,
				"stop":                types.StringType,
				"gateway_enabled":     types.BoolType,
				"conflict_checking":   types.BoolType,
				"ntp_enabled":         types.BoolType,
				"ntp_servers":         types.ListType{ElemType: types.StringType},
				"time_offset_enabled": types.BoolType,
				"dns_enabled":         types.BoolType,
				"leasetime":           timetypes.GoDurationType{},
				"wins":                types.ObjectType{AttrTypes: winsModel{}.AttributeTypes()},
				"wpad_url":            types.StringType,
				"tftp_server":         types.StringType,
				"unifi_controller":    types.StringType,
				"dns_servers":         types.ListType{ElemType: types.StringType},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpServerModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_natOutboundIPAddressesModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		d    natOutboundIPAddressesModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			d:    natOutboundIPAddressesModel{},
			want: map[string]attr.Type{
				"ip_address":        types.StringType,
				"ip_address_pool":   types.ListType{ElemType: types.StringType},
				"mode":              types.StringType,
				"wan_network_group": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("natOutboundIPAddressesModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_natOutboundIPAddresses(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		{
			name: "returns correct type map",
			want: map[string]attr.Type{
				"ip_address":        types.StringType,
				"ip_address_pool":   types.ListType{ElemType: types.StringType},
				"mode":              types.StringType,
				"wan_network_group": types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := natOutboundIPAddresses(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("natOutboundIPAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpGuardingModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpGuardingModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    dhcpGuardingModel{},
			want: map[string]attr.Type{
				"enabled": types.BoolType,
				"servers": types.ListType{ElemType: types.StringType},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpGuardingModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpRelayModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		d    dhcpRelayModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			d:    dhcpRelayModel{},
			want: map[string]attr.Type{
				"enabled": types.BoolType,
				"servers": types.ListType{ElemType: types.StringType},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpRelayModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dhcpV6ServerModel_AttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		m    dhcpV6ServerModel
		want map[string]attr.Type
	}{
		{
			name: "returns correct attribute types",
			m:    dhcpV6ServerModel{},
			want: map[string]attr.Type{
				"enabled":     types.BoolType,
				"dns_auto":    types.BoolType,
				"dns_servers": types.ListType{ElemType: types.StringType},
				"lease":       types.Int64Type,
				"start":       types.StringType,
				"stop":        types.StringType,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("dhcpV6ServerModel.AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_networkResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
	}{
		{
			name: "sets correct type name",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != "unifi_network" {
				t.Errorf("Metadata() TypeName = %v, want unifi_network", tt.args.resp.TypeName)
			}
		})
	}
}

func Test_networkResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
	}{
		{
			name: "returns identity schema with id",
			r:    &networkResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwresource.IdentitySchemaRequest{},
				resp: &fwresource.IdentitySchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.IdentitySchema.Attributes["id"]; !ok {
				t.Error("IdentitySchema() missing 'id' attribute")
			}
		})
	}
}

func Test_networkResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
	}{
		{
			name: "returns schema with key attributes",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			for _, key := range []string{"id", "name", "subnet"} {
				if _, ok := tt.args.resp.Schema.Attributes[key]; !ok {
					t.Errorf("Schema() missing attribute %q", key)
				}
			}
		})
	}
}

func Test_networkResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{
		{
			name: "returns non-nil map",
			r:    &networkResource{},
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.UpgradeState(tt.args.ctx)
			if got == nil {
				t.Error("UpgradeState() returned nil")
			}
		})
	}
}

func Test_networkResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
	}{
		{
			name: "nil provider data does not error",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: nil},
				resp: &fwresource.ConfigureResponse{},
			},
		},
		{
			name: "wrong type produces error",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: "wrong"},
				resp: &fwresource.ConfigureResponse{},
			},
		},
		{
			name: "correct type sets client",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: &Client{}},
				resp: &fwresource.ConfigureResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			switch tt.name {
			case "nil provider data does not error":
				if tt.args.resp.Diagnostics.HasError() {
					t.Errorf("Configure() unexpected error: %v", tt.args.resp.Diagnostics)
				}
			case "wrong type produces error":
				if !tt.args.resp.Diagnostics.HasError() {
					t.Error("Configure() expected error for wrong type")
				}
			case "correct type sets client":
				if tt.args.resp.Diagnostics.HasError() {
					t.Errorf("Configure() unexpected error: %v", tt.args.resp.Diagnostics)
				}
				if tt.r.client == nil {
					t.Error("Configure() client not set")
				}
			}
		})
	}
}

func Test_networkResource_modelToNetwork(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *networkResourceModel
	}
	tests := []struct {
		name  string
		r     *networkResource
		args  args
		want  *unifi.Network
		want1 diag.Diagnostics
	}{
		{
			name: "minimal model conversion",
			r:    &networkResource{},
			args: args{
				ctx: context.Background(),
				model: &networkResourceModel{
					Name:                        types.StringValue("test-net"),
					Enabled:                     types.BoolValue(true),
					Subnet:                      cidrtypes.NewIPv4PrefixValue("10.0.0.0/24"),
					AutoScale:                   types.BoolValue(false),
					NetworkIsolation:            types.BoolValue(false),
					SettingPreference:           types.StringNull(),
					InternetAccess:              types.BoolValue(false),
					MulticastDNS:                types.BoolValue(false),
					GatewayType:                 types.StringNull(),
					IPv6InterfaceType:           types.StringNull(),
					IPv6ClientAddressAssignment: types.StringNull(),
					IPv6StaticSubnet:            types.StringNull(),
					IPv6RA:                      types.BoolValue(false),
					IPv6RAPriority:              types.StringNull(),
					IPv6RAPreferredLifetime:     timetypes.NewGoDurationNull(),
					IPv6RAValidLifetime:         timetypes.NewGoDurationNull(),
					IPv6PDInterface:             types.StringNull(),
					IPv6PDPrefixID:              types.StringNull(),
					IPv6PDStart:                 types.StringNull(),
					IPv6PDStop:                  types.StringNull(),
					IPv6PDAutoPrefixidEnabled:   types.BoolValue(false),
					LteLan:                      types.BoolValue(false),
					ThirdPartyGateway:           types.BoolValue(false),
					IgmpSnooping:                types.BoolValue(false),
					Vlan:                        types.Int64Null(),
					NatOutboundIPAddresses: types.ListNull(
						types.ObjectType{AttrTypes: natOutboundIPAddresses()},
					),
					IPAliases:   types.ListNull(types.StringType),
					IPv6Aliases: types.ListNull(types.StringType),
					DhcpServer: types.ObjectNull(
						dhcpServerModel{}.AttributeTypes(),
					),
					DhcpRelay: types.ObjectNull(
						dhcpRelayModel{}.AttributeTypes(),
					),
					DhcpV6Server: types.ObjectNull(
						dhcpV6ServerModel{}.AttributeTypes(),
					),
					DhcpGuarding: types.ObjectNull(
						dhcpGuardingModel{}.AttributeTypes(),
					),
				},
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToNetwork(tt.args.ctx, tt.args.model)
			if got == nil {
				t.Fatal("modelToNetwork() returned nil network")
			}
			if *got.Name != "test-net" {
				t.Errorf("modelToNetwork() Name = %v, want test-net", *got.Name)
			}
			if got.Purpose != unifi.PurposeCorporate {
				t.Errorf(
					"modelToNetwork() Purpose = %v, want %v",
					got.Purpose,
					unifi.PurposeCorporate,
				)
			}
			if got1 != nil && got1.HasError() {
				t.Errorf("modelToNetwork() diagnostics has errors: %v", got1)
			}
		})
	}
}

func Test_networkResource_networkToModel(t *testing.T) {
	type args struct {
		ctx           context.Context
		network       *unifi.Network
		model         *networkResourceModel
		site          string
		previousModel *networkResourceModel
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
		want diag.Diagnostics
	}{
		{
			name: "minimal network to model",
			r:    &networkResource{},
			args: args{
				ctx: context.Background(),
				network: &unifi.Network{
					ID:      "net-123",
					Name:    strPtr("test-net"),
					Purpose: unifi.PurposeCorporate,
					Enabled: true,
				},
				model: &networkResourceModel{},
				site:  "default",
				previousModel: &networkResourceModel{
					DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
					DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
					DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
					DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
					NatOutboundIPAddresses: types.ListNull(
						types.ObjectType{AttrTypes: natOutboundIPAddresses()},
					),
					IPAliases:   types.ListNull(types.StringType),
					IPv6Aliases: types.ListNull(types.StringType),
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.networkToModel(
				tt.args.ctx,
				tt.args.network,
				tt.args.model,
				tt.args.site,
				tt.args.previousModel,
			)
			if got != nil && got.HasError() {
				t.Errorf("networkToModel() diagnostics has errors: %v", got)
			}
			if tt.args.model.ID.ValueString() != "net-123" {
				t.Errorf("networkToModel() ID = %v, want net-123", tt.args.model.ID.ValueString())
			}
			if tt.args.model.Site.ValueString() != "default" {
				t.Errorf(
					"networkToModel() Site = %v, want default",
					tt.args.model.Site.ValueString(),
				)
			}
			if tt.args.model.Name.ValueString() != "test-net" {
				t.Errorf(
					"networkToModel() Name = %v, want test-net",
					tt.args.model.Name.ValueString(),
				)
			}
		})
	}
}

func Test_networkResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *networkResource
		args args
	}{
		{
			name: "returns schema without panic",
			r:    &networkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.Schema.Attributes == nil {
				t.Error("ListResourceConfigSchema() returned nil attributes")
			}
		})
	}
}

func TestAccNetworkList_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_14_0),
		},
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_basic(),
			},
			{
				Query: true,
				Config: `
					provider "unifi" {}
					list "unifi_network" "test" {
						provider = unifi
						config {
							filter {
								name  = "name"
								value = "Test VLAN"
							}
						}
					}
				`,
				QueryResultChecks: []querycheck.QueryResultCheck{
					querycheck.ExpectLengthAtLeast("unifi_network.test", 1),
				},
			},
		},
	})
}

// Test_networkResource_networkToModel_multicastDNS guards #282: a corporate
// network's multicast_dns is overridden to false server-side by some controllers
// (UniFi OS gateways), so a user-configured true would fail the consistency
// check. The configured/known value must be preserved; an unset (unknown) value
// falls back to the controller's value.
func Test_networkResource_networkToModel_multicastDNS(t *testing.T) {
	r := &networkResource{}
	base := func() *networkResourceModel {
		return &networkResourceModel{
			DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:   types.ListNull(types.StringType),
			IPv6Aliases: types.ListNull(types.StringType),
		}
	}
	// Corporate network (not vlan-only); controller forces mdns false.
	network := &unifi.Network{
		ID:          "net-1",
		Name:        strPtr("IoT"),
		Purpose:     unifi.PurposeCorporate,
		Enabled:     true,
		IPSubnet:    strPtr("10.0.2.1/24"),
		MdnsEnabled: false,
	}

	t.Run("configured true is preserved", func(t *testing.T) {
		prev := base()
		prev.MulticastDNS = types.BoolValue(true)
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prev)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if !model.MulticastDNS.ValueBool() {
			t.Errorf("configured multicast_dns=true not preserved: %v", model.MulticastDNS)
		}
	})

	t.Run("unset falls back to controller value", func(t *testing.T) {
		prev := base()
		prev.MulticastDNS = types.BoolUnknown()
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prev)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if model.MulticastDNS.ValueBool() {
			t.Errorf("unset multicast_dns should reflect controller false, got true")
		}
	})
}

// Test_networkResource_networkToModel_ipAliases guards #413: ip_aliases must
// round-trip whatever the controller reports. nat_outbound_ip_addresses also
// round-trips but only when it was already non-null in the previous state
// (i.e., the user manages it); for unmanaged configurations the field stays
// null even when the controller returns data.
func Test_networkResource_networkToModel_ipAliases(t *testing.T) {
	r := &networkResource{}

	// prevManaged has NatOutboundIPAddresses non-null: the user has configured it.
	prevManaged := &networkResourceModel{
		DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
		DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
		DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
		DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
		NatOutboundIPAddresses: types.ListValueMust(
			types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			[]attr.Value{},
		),
		IPAliases:   types.ListNull(types.StringType),
		IPv6Aliases: types.ListNull(types.StringType),
	}

	// prevUnmanaged has NatOutboundIPAddresses null: the user did not configure it.
	prevUnmanaged := &networkResourceModel{
		DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
		DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
		DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
		DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
		NatOutboundIPAddresses: types.ListNull(
			types.ObjectType{AttrTypes: natOutboundIPAddresses()},
		),
		IPAliases:   types.ListNull(types.StringType),
		IPv6Aliases: types.ListNull(types.StringType),
	}

	t.Run("non-empty values round-trip (managed)", func(t *testing.T) {
		network := &unifi.Network{
			ID:        "net-1",
			Name:      strPtr("aliased"),
			Purpose:   unifi.PurposeCorporate,
			Enabled:   true,
			IPSubnet:  strPtr("10.0.2.1/24"),
			IPAliases: []string{"10.0.2.5", "10.0.2.6"},
			NATOutboundIPAddresses: []unifi.NetworkNATOutboundIPAddresses{
				{
					IPAddress:       "203.0.113.5",
					Mode:            strPtr("ip_address"),
					WANNetworkGroup: strPtr("WAN"),
				},
			},
		}
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prevManaged)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}

		if model.IPAliases.IsNull() {
			t.Fatal("IPAliases: got null, want the two configured aliases")
		}
		var ipAliases []string
		if diags := model.IPAliases.ElementsAs(
			context.Background(),
			&ipAliases,
			false,
		); diags.HasError() {
			t.Fatalf("ElementsAs: %v", diags)
		}
		if len(ipAliases) != 2 || ipAliases[0] != "10.0.2.5" || ipAliases[1] != "10.0.2.6" {
			t.Errorf("IPAliases = %v, want [10.0.2.5 10.0.2.6]", ipAliases)
		}

		if model.NatOutboundIPAddresses.IsNull() {
			t.Fatal("NatOutboundIPAddresses: got null, want the configured entry")
		}
		var natEntries []natOutboundIPAddressesModel
		if diags := model.NatOutboundIPAddresses.ElementsAs(
			context.Background(),
			&natEntries,
			false,
		); diags.HasError() {
			t.Fatalf("ElementsAs: %v", diags)
		}
		if len(natEntries) != 1 || natEntries[0].IPAddress.ValueString() != "203.0.113.5" ||
			natEntries[0].Mode.ValueString() != "ip_address" ||
			natEntries[0].WANNetworkGroup.ValueString() != "WAN" {
			t.Errorf(
				"NatOutboundIPAddresses = %+v, want ip_address=203.0.113.5 mode=ip_address wan_network_group=WAN",
				natEntries,
			)
		}
	})

	t.Run("unmanaged nat stays null when API returns data", func(t *testing.T) {
		network := &unifi.Network{
			ID:       "net-3",
			Name:     strPtr("nat-unmanaged"),
			Purpose:  unifi.PurposeCorporate,
			Enabled:  true,
			IPSubnet: strPtr("10.0.4.1/24"),
			NATOutboundIPAddresses: []unifi.NetworkNATOutboundIPAddresses{
				{
					IPAddress:       "203.0.113.1",
					Mode:            strPtr("ip_address"),
					WANNetworkGroup: strPtr("WAN"),
				},
			},
		}
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prevUnmanaged)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if !model.NatOutboundIPAddresses.IsNull() {
			t.Errorf(
				"NatOutboundIPAddresses = %v, want null (unmanaged)",
				model.NatOutboundIPAddresses,
			)
		}
	})

	t.Run("empty values stay null", func(t *testing.T) {
		network := &unifi.Network{
			ID:       "net-2",
			Name:     strPtr("no-aliases"),
			Purpose:  unifi.PurposeCorporate,
			Enabled:  true,
			IPSubnet: strPtr("10.0.3.1/24"),
		}
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prevUnmanaged)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if !model.IPAliases.IsNull() {
			t.Errorf("IPAliases = %v, want null", model.IPAliases)
		}
		if !model.NatOutboundIPAddresses.IsNull() {
			t.Errorf("NatOutboundIPAddresses = %v, want null", model.NatOutboundIPAddresses)
		}
		if !model.IPv6Aliases.IsNull() {
			t.Errorf("IPv6Aliases = %v, want null", model.IPv6Aliases)
		}
	})

	t.Run("managed empty ip_aliases stays a known empty list", func(t *testing.T) {
		// A configured `ip_aliases = []` plans a known empty list; the readback
		// must echo an empty list, not null, or apply fails with an
		// inconsistent-result error.
		prevEmptyAliases := &networkResourceModel{
			DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:   types.ListValueMust(types.StringType, []attr.Value{}),
			IPv6Aliases: types.ListNull(types.StringType),
		}
		network := &unifi.Network{
			ID:       "net-4",
			Name:     strPtr("empty-aliases"),
			Purpose:  unifi.PurposeCorporate,
			Enabled:  true,
			IPSubnet: strPtr("10.0.5.1/24"),
		}
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prevEmptyAliases)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if model.IPAliases.IsNull() || model.IPAliases.IsUnknown() {
			t.Fatalf("IPAliases = %v, want known empty list", model.IPAliases)
		}
		if n := len(model.IPAliases.Elements()); n != 0 {
			t.Errorf("IPAliases has %d elements, want 0", n)
		}
	})
}

// Test_networkResource_networkToModel_dnsServersEmptyList guards #429:
// dhcp_server.dns_servers and dhcp_server.wins.addresses are Optional but not
// Computed, so a config of `[]` plans an empty (non-null) list. Read must
// mirror an empty API response as an empty list when the previous plan/state
// held an empty list, rather than always collapsing to null - otherwise
// Terraform reports "provider produced inconsistent result after apply" and
// the empty value can never be expressed. A previous null (never configured)
// must still read back as null.
func Test_networkResource_networkToModel_dnsServersEmptyList(t *testing.T) {
	r := &networkResource{}
	ctx := context.Background()

	// The controller holds no DNS servers and no WINS addresses.
	network := &unifi.Network{
		ID:      "net-1",
		Name:    strPtr("test-net"),
		Purpose: unifi.PurposeCorporate,
		Enabled: true,
	}

	base := func() *networkResourceModel {
		return &networkResourceModel{
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:   types.ListNull(types.StringType),
			IPv6Aliases: types.ListNull(types.StringType),
		}
	}

	dhcpServerObj := func(dnsServers, winsAddresses, ntpServers types.List) types.Object {
		wins := types.ObjectValueMust(winsModel{}.AttributeTypes(), map[string]attr.Value{
			"enabled":   types.BoolValue(false),
			"addresses": winsAddresses,
		})
		return types.ObjectValueMust(dhcpServerModel{}.AttributeTypes(), map[string]attr.Value{
			"boot":                types.ObjectNull(dhcpBootModel{}.AttributeTypes()),
			"enabled":             types.BoolValue(true),
			"start":               types.StringNull(),
			"stop":                types.StringNull(),
			"gateway_enabled":     types.BoolValue(false),
			"conflict_checking":   types.BoolValue(true),
			"ntp_enabled":         types.BoolValue(false),
			"ntp_servers":         ntpServers,
			"time_offset_enabled": types.BoolValue(false),
			"dns_enabled":         types.BoolValue(false),
			"leasetime":           timetypes.NewGoDurationNull(),
			"wins":                wins,
			"wpad_url":            types.StringNull(),
			"tftp_server":         types.StringNull(),
			"unifi_controller":    types.StringNull(),
			"dns_servers":         dnsServers,
		})
	}

	emptyList := types.ListValueMust(types.StringType, []attr.Value{})
	nullList := types.ListNull(types.StringType)

	t.Run("empty config list round-trips as empty list, not null", func(t *testing.T) {
		prev := base()
		prev.DhcpServer = dhcpServerObj(emptyList, emptyList, emptyList)

		var model networkResourceModel
		d := r.networkToModel(ctx, network, &model, "default", prev)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}

		var got dhcpServerModel
		d = model.DhcpServer.As(ctx, &got, basetypes.ObjectAsOptions{})
		if d.HasError() {
			t.Fatalf("extracting dhcp_server: %v", d)
		}
		if got.DnsServers.IsNull() {
			t.Errorf("dns_servers = null, want empty list")
		}
		if len(got.DnsServers.Elements()) != 0 {
			t.Errorf("dns_servers = %v, want 0 elements", got.DnsServers.Elements())
		}
		if got.NtpServers.IsNull() {
			t.Errorf("ntp_servers = null, want empty list")
		}
		if len(got.NtpServers.Elements()) != 0 {
			t.Errorf("ntp_servers = %v, want 0 elements", got.NtpServers.Elements())
		}

		var gotWins winsModel
		d = got.Wins.As(ctx, &gotWins, basetypes.ObjectAsOptions{})
		if d.HasError() {
			t.Fatalf("extracting wins: %v", d)
		}
		if gotWins.Addresses.IsNull() {
			t.Errorf("wins.addresses = null, want empty list")
		}
	})

	t.Run("never-configured stays null", func(t *testing.T) {
		prev := base()
		prev.DhcpServer = dhcpServerObj(nullList, nullList, nullList)

		var model networkResourceModel
		d := r.networkToModel(ctx, network, &model, "default", prev)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}

		var got dhcpServerModel
		d = model.DhcpServer.As(ctx, &got, basetypes.ObjectAsOptions{})
		if d.HasError() {
			t.Fatalf("extracting dhcp_server: %v", d)
		}
		if !got.DnsServers.IsNull() {
			t.Errorf("dns_servers = %v, want null", got.DnsServers)
		}
		if !got.NtpServers.IsNull() {
			t.Errorf("ntp_servers = %v, want null", got.NtpServers)
		}

		var gotWins winsModel
		d = got.Wins.As(ctx, &gotWins, basetypes.ObjectAsOptions{})
		if d.HasError() {
			t.Fatalf("extracting wins: %v", d)
		}
		if !gotWins.Addresses.IsNull() {
			t.Errorf("wins.addresses = %v, want null", gotWins.Addresses)
		}
	})
}

// Test_networkResource_networkToModel_normalizesControllerDefaults guards #414:
// UniFi may omit gateway_type and ipv6_interface_type when they have their
// implicit defaults. Import must write the provider defaults into state instead
// of null, otherwise every subsequent plan proposes null -> default/none.
func Test_networkResource_networkToModel_normalizesControllerDefaults(t *testing.T) {
	r := &networkResource{}
	base := func() *networkResourceModel {
		return &networkResourceModel{
			// Import seeds identity only, leaving computed attributes null.
			NetworkIsolation: types.BoolNull(),
			DhcpServer:       types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:        types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server:     types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding:     types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:   types.ListNull(types.StringType),
			IPv6Aliases: types.ListNull(types.StringType),
		}
	}

	tests := []struct {
		name                  string
		gatewayType           *string
		ipv6InterfaceType     *string
		wantGatewayType       string
		wantIPv6InterfaceType string
	}{
		{
			name:                  "nil API values use schema defaults",
			wantGatewayType:       "default",
			wantIPv6InterfaceType: "none",
		},
		{
			name:                  "empty API values use schema defaults",
			gatewayType:           strPtr(""),
			ipv6InterfaceType:     strPtr(""),
			wantGatewayType:       "default",
			wantIPv6InterfaceType: "none",
		},
		{
			name:                  "explicit API values are preserved",
			gatewayType:           strPtr("switch"),
			ipv6InterfaceType:     strPtr("static"),
			wantGatewayType:       "switch",
			wantIPv6InterfaceType: "static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := &unifi.Network{
				ID:                "net-imported",
				Name:              strPtr("Imported Network"),
				Purpose:           unifi.PurposeCorporate,
				Enabled:           true,
				IPSubnet:          strPtr("10.25.0.1/24"),
				GatewayType:       tt.gatewayType,
				IPV6InterfaceType: tt.ipv6InterfaceType,
			}
			var model networkResourceModel
			d := r.networkToModel(context.Background(), network, &model, "default", base())
			if d.HasError() {
				t.Fatalf("networkToModel: %v", d)
			}
			if model.GatewayType.IsNull() || model.GatewayType.IsUnknown() {
				t.Fatalf("gateway_type should be known, got %v", model.GatewayType)
			}
			if got := model.GatewayType.ValueString(); got != tt.wantGatewayType {
				t.Errorf("gateway_type = %q, want %q", got, tt.wantGatewayType)
			}
			if model.IPv6InterfaceType.IsNull() || model.IPv6InterfaceType.IsUnknown() {
				t.Fatalf("ipv6_interface_type should be known, got %v", model.IPv6InterfaceType)
			}
			if got := model.IPv6InterfaceType.ValueString(); got != tt.wantIPv6InterfaceType {
				t.Errorf(
					"ipv6_interface_type = %q, want %q",
					got,
					tt.wantIPv6InterfaceType,
				)
			}
		})
	}
}

func Test_networkResource_networkToModel_normalizesVLANOnlyDefaults(t *testing.T) {
	r := &networkResource{}
	base := func() *networkResourceModel {
		return &networkResourceModel{
			// Import seeds identity only, leaving computed attributes null.
			NetworkIsolation: types.BoolNull(),
			DhcpServer:       types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:        types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server:     types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding:     types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:   types.ListNull(types.StringType),
			IPv6Aliases: types.ListNull(types.StringType),
		}
	}

	tests := []struct {
		name                  string
		previousGatewayType   types.String
		previousIPv6Type      types.String
		wantGatewayType       string
		wantIPv6InterfaceType string
	}{
		{
			name:                  "import nulls use schema defaults",
			previousGatewayType:   types.StringNull(),
			previousIPv6Type:      types.StringNull(),
			wantGatewayType:       "default",
			wantIPv6InterfaceType: "none",
		},
		{
			name:                  "unknown plan values use schema defaults",
			previousGatewayType:   types.StringUnknown(),
			previousIPv6Type:      types.StringUnknown(),
			wantGatewayType:       "default",
			wantIPv6InterfaceType: "none",
		},
		{
			name:                  "empty prior values use schema defaults",
			previousGatewayType:   types.StringValue(""),
			previousIPv6Type:      types.StringValue(""),
			wantGatewayType:       "default",
			wantIPv6InterfaceType: "none",
		},
		{
			name:                  "explicit prior values are preserved",
			previousGatewayType:   types.StringValue("switch"),
			previousIPv6Type:      types.StringValue("static"),
			wantGatewayType:       "switch",
			wantIPv6InterfaceType: "static",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previous := base()
			previous.GatewayType = tt.previousGatewayType
			previous.IPv6InterfaceType = tt.previousIPv6Type
			network := &unifi.Network{
				ID:      "net-vlan-only-imported",
				Name:    strPtr("Imported VLAN Only"),
				Purpose: unifi.PurposeVLANOnly,
				Enabled: true,
			}
			var model networkResourceModel
			d := r.networkToModel(context.Background(), network, &model, "default", previous)
			if d.HasError() {
				t.Fatalf("networkToModel: %v", d)
			}
			if model.GatewayType.IsNull() || model.GatewayType.IsUnknown() {
				t.Fatalf("gateway_type should be known, got %v", model.GatewayType)
			}
			if got := model.GatewayType.ValueString(); got != tt.wantGatewayType {
				t.Errorf("gateway_type = %q, want %q", got, tt.wantGatewayType)
			}
			if model.IPv6InterfaceType.IsNull() || model.IPv6InterfaceType.IsUnknown() {
				t.Fatalf("ipv6_interface_type should be known, got %v", model.IPv6InterfaceType)
			}
			if got := model.IPv6InterfaceType.ValueString(); got != tt.wantIPv6InterfaceType {
				t.Errorf(
					"ipv6_interface_type = %q, want %q",
					got,
					tt.wantIPv6InterfaceType,
				)
			}
		})
	}
}

// Test_networkResource_purpose covers #276: purpose must be author-settable
// (guest/vlan-only/corporate) on write and reflected from the controller on read.
func Test_networkResource_purpose(t *testing.T) {
	r := &networkResource{}

	baseModel := func() *networkResourceModel {
		return &networkResourceModel{
			Name:              types.StringValue("test-net"),
			Subnet:            cidrtypes.NewIPv4PrefixValue("10.0.0.0/24"),
			ThirdPartyGateway: types.BoolValue(false),
			Purpose:           types.StringNull(),
			NatOutboundIPAddresses: types.ListNull(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
			),
			IPAliases:    types.ListNull(types.StringType),
			IPv6Aliases:  types.ListNull(types.StringType),
			DhcpServer:   types.ObjectNull(dhcpServerModel{}.AttributeTypes()),
			DhcpRelay:    types.ObjectNull(dhcpRelayModel{}.AttributeTypes()),
			DhcpV6Server: types.ObjectNull(dhcpV6ServerModel{}.AttributeTypes()),
			DhcpGuarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
		}
	}

	t.Run("write: unset defaults to corporate", func(t *testing.T) {
		got, d := r.modelToNetwork(context.Background(), baseModel())
		if d.HasError() {
			t.Fatalf("modelToNetwork: %v", d)
		}
		if got.Purpose != unifi.PurposeCorporate {
			t.Errorf("Purpose = %q, want %q", got.Purpose, unifi.PurposeCorporate)
		}
	})

	t.Run("write: configured guest is sent", func(t *testing.T) {
		m := baseModel()
		m.Purpose = types.StringValue(unifi.PurposeGuest)
		got, d := r.modelToNetwork(context.Background(), m)
		if d.HasError() {
			t.Fatalf("modelToNetwork: %v", d)
		}
		if got.Purpose != unifi.PurposeGuest {
			t.Errorf("Purpose = %q, want %q", got.Purpose, unifi.PurposeGuest)
		}
	})

	t.Run("write: third_party_gateway forces vlan-only over purpose", func(t *testing.T) {
		m := baseModel()
		m.Purpose = types.StringValue(unifi.PurposeGuest)
		m.ThirdPartyGateway = types.BoolValue(true)
		got, d := r.modelToNetwork(context.Background(), m)
		if d.HasError() {
			t.Fatalf("modelToNetwork: %v", d)
		}
		if got.Purpose != unifi.PurposeVLANOnly {
			t.Errorf(
				"Purpose = %q, want %q (third_party_gateway precedence)",
				got.Purpose,
				unifi.PurposeVLANOnly,
			)
		}
	})

	t.Run("read: controller guest is reflected", func(t *testing.T) {
		network := &unifi.Network{
			ID:       "net-guest",
			Name:     strPtr("Guest"),
			Purpose:  unifi.PurposeGuest,
			Enabled:  true,
			IPSubnet: strPtr("10.0.9.1/24"),
		}
		prev := baseModel()
		prev.Purpose = types.StringValue(unifi.PurposeGuest)
		var model networkResourceModel
		d := r.networkToModel(context.Background(), network, &model, "default", prev)
		if d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if model.Purpose.ValueString() != unifi.PurposeGuest {
			t.Errorf("Purpose = %q, want %q", model.Purpose.ValueString(), unifi.PurposeGuest)
		}
	})
}

// TestNetworkFirewallZoneIDModelRoundTrip validates the model <-> go-unifi struct
// conversion for the firewall_zone_id attribute on the unifi_network resource.
// It is a unit test rather than an acceptance test because zone-based firewall
// is not available in the dockerized acceptance controller.
func TestNetworkFirewallZoneIDModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	// Write path: a configured firewall_zone_id must reach the API struct.
	model := &networkResourceModel{
		Name:           types.StringValue("Test Zone Network"),
		FirewallZoneID: types.StringValue("60b7c25e0cf2732bbdf1e102"),
	}

	network, diags := r.modelToNetwork(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToNetwork failed: %v", diags)
	}

	if network.FirewallZoneID == nil {
		t.Fatalf("modelToNetwork: FirewallZoneID pointer is nil, want a value")
	}
	if *network.FirewallZoneID != "60b7c25e0cf2732bbdf1e102" {
		t.Errorf(
			"modelToNetwork: FirewallZoneID = %q, want 60b7c25e0cf2732bbdf1e102",
			*network.FirewallZoneID,
		)
	}

	// Read path: the controller-side value must land in the model for drift
	// detection.
	apiNetwork := &unifi.Network{
		ID:             "net-123",
		Name:           stringPtr("Test Zone Network"),
		FirewallZoneID: stringPtr("60b7c25e0cf2732bbdf1e102"),
	}

	var out networkResourceModel
	var planData networkResourceModel

	readDiags := r.networkToModel(ctx, apiNetwork, &out, "default", &planData)
	if readDiags.HasError() {
		t.Fatalf("networkToModel failed: %v", readDiags)
	}

	if out.FirewallZoneID.ValueString() != "60b7c25e0cf2732bbdf1e102" {
		t.Errorf(
			"networkToModel: FirewallZoneID = %q, want 60b7c25e0cf2732bbdf1e102",
			out.FirewallZoneID.ValueString(),
		)
	}
}

func stringPtr(s string) *string {
	return &s
}

func Test_preserveUnmanagedDhcpGuarding(t *testing.T) {
	t.Parallel()

	current := &unifi.Network{
		DHCPguardEnabled: true,
		DHCPDIP1:         "192.168.1.1",
		DHCPDIP2:         "192.168.1.2",
	}

	t.Run("null block preserves controller values", func(t *testing.T) {
		t.Parallel()

		network := &unifi.Network{}
		preserved := preserveUnmanagedDhcpGuarding(
			types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			network,
			current,
		)

		if !preserved {
			t.Fatal("expected preservation for a null dhcp_guarding block")
		}
		if !network.DHCPguardEnabled {
			t.Error("expected DHCPguardEnabled to be preserved as true")
		}
		if network.DHCPDIP1 != "192.168.1.1" || network.DHCPDIP2 != "192.168.1.2" {
			t.Errorf(
				"expected trusted servers to be preserved, got %q, %q",
				network.DHCPDIP1,
				network.DHCPDIP2,
			)
		}
	})

	t.Run("managed block is left alone", func(t *testing.T) {
		t.Parallel()

		planned, diags := types.ObjectValueFrom(
			t.Context(),
			dhcpGuardingModel{}.AttributeTypes(),
			dhcpGuardingModel{
				Enabled: types.BoolValue(false),
				Servers: types.ListNull(types.StringType),
			},
		)
		if diags.HasError() {
			t.Fatalf("failed to build planned object: %v", diags)
		}

		network := &unifi.Network{}
		if preserveUnmanagedDhcpGuarding(planned, network, current) {
			t.Fatal("expected no preservation for a managed dhcp_guarding block")
		}
		if network.DHCPguardEnabled || network.DHCPDIP1 != "" {
			t.Error("expected the outgoing network to be untouched")
		}
	})
}

// minimalNetworkPlan returns a tfsdk.Plan with every attribute at its null/zero
// value so that individual attributes can be overridden via SetAttribute before
// being passed to ModifyPlan in unit tests.
func minimalNetworkPlan(ctx context.Context, t *testing.T) (tfsdk.Plan, tfsdk.Config) {
	t.Helper()
	r := &networkResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build a null (destroying) plan — callers will SetAttribute to make it
	// non-null where needed, or replace Raw with a non-null tftypes value.
	// For "normal" (non-destroy) plan tests we need a non-null Raw value, so
	// we use tftypes.UnknownValue to indicate "unknown object" instead of null.
	planRaw := tftypes.NewValue(schemaType, tftypes.UnknownValue)
	plan := tfsdk.Plan{
		Schema: schemaResp.Schema,
		Raw:    planRaw,
	}
	config := tfsdk.Config{
		Schema: schemaResp.Schema,
		Raw:    tftypes.NewValue(schemaType, nil),
	}
	return plan, config
}

// Test_networkResource_ModifyPlan_ipv6Aliases guards that ModifyPlan rejects
// any non-null ipv6_aliases value (known, unknown, or empty list) with a clear
// error instead of letting Create/Update silently drop it and produce a
// confusing "provider produced inconsistent result after apply" failure (#413).
func Test_networkResource_ModifyPlan_ipv6Aliases(t *testing.T) {
	r := &networkResource{}
	ctx := context.Background()

	tests := []struct {
		name        string
		ipv6Aliases types.List
		wantError   bool
	}{
		{
			name:        "null list: no error",
			ipv6Aliases: types.ListNull(types.StringType),
			wantError:   false,
		},
		{
			name: "non-empty known list: error",
			ipv6Aliases: types.ListValueMust(
				types.StringType,
				[]attr.Value{types.StringValue("2001:db8::1")},
			),
			wantError: true,
		},
		{
			name:        "unknown list: error",
			ipv6Aliases: types.ListUnknown(types.StringType),
			wantError:   true,
		},
		{
			name:        "empty known list: error",
			ipv6Aliases: types.ListValueMust(types.StringType, []attr.Value{}),
			wantError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, config := minimalNetworkPlan(ctx, t)
			diags := plan.SetAttribute(ctx, path.Root("ipv6_aliases"), tt.ipv6Aliases)
			if diags.HasError() {
				t.Fatalf("SetAttribute(ipv6_aliases): %v", diags)
			}

			req := fwresource.ModifyPlanRequest{
				Plan:   plan,
				Config: config,
			}
			resp := &fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, req, resp)

			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error diagnostic, got none")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

// Test_networkResource_ModifyPlan_ipAddressPool guards that ModifyPlan rejects
// a non-null ip_address_pool inside nat_outbound_ip_addresses entries, since
// the field is not yet wired to the API request side.
func Test_networkResource_ModifyPlan_ipAddressPool(t *testing.T) {
	r := &networkResource{}
	ctx := context.Background()

	tests := []struct {
		name      string
		pool      types.List
		wantError bool
	}{
		{
			name:      "null pool: no error",
			pool:      types.ListNull(types.StringType),
			wantError: false,
		},
		{
			name: "non-empty pool: error",
			pool: types.ListValueMust(
				types.StringType,
				[]attr.Value{types.StringValue("203.0.113.10")},
			),
			wantError: true,
		},
		{
			name:      "empty pool list: error",
			pool:      types.ListValueMust(types.StringType, []attr.Value{}),
			wantError: true,
		},
		{
			name:      "unknown pool (from data source): error",
			pool:      types.ListUnknown(types.StringType),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, config := minimalNetworkPlan(ctx, t)

			// Ensure ipv6_aliases is null so it doesn't trigger its own error
			// before we reach the ip_address_pool check.
			diags := plan.SetAttribute(
				ctx,
				path.Root("ipv6_aliases"),
				types.ListNull(types.StringType),
			)
			if diags.HasError() {
				t.Fatalf("SetAttribute(ipv6_aliases): %v", diags)
			}

			// Build a nat_outbound_ip_addresses list with one entry whose
			// ip_address_pool is set to the test value.
			entry, d := types.ObjectValue(
				natOutboundIPAddresses(),
				map[string]attr.Value{
					"ip_address":        types.StringNull(),
					"ip_address_pool":   tt.pool,
					"mode":              types.StringNull(),
					"wan_network_group": types.StringNull(),
				},
			)
			if d.HasError() {
				t.Fatalf("types.ObjectValue: %v", d)
			}
			natList, d := types.ListValue(
				types.ObjectType{AttrTypes: natOutboundIPAddresses()},
				[]attr.Value{entry},
			)
			if d.HasError() {
				t.Fatalf("types.ListValue: %v", d)
			}
			diags = plan.SetAttribute(ctx, path.Root("nat_outbound_ip_addresses"), natList)
			if diags.HasError() {
				t.Fatalf("SetAttribute(nat_outbound_ip_addresses): %v", diags)
			}

			req := fwresource.ModifyPlanRequest{
				Plan:   plan,
				Config: config,
			}
			resp := &fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, req, resp)

			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error diagnostic, got none")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}

// Test_networkResource_ModifyPlan_dhcpGuardingForcesManual guards #419: the
// controller force-resets dhcpguard_enabled to false on any write to a
// corporate or guest network whose setting_preference is "auto" (the provider
// default), so ModifyPlan must pin setting_preference to "manual" whenever the
// plan enables dhcp_guarding on those purposes and the practitioner has not
// chosen a preference explicitly. vlan-only networks keep guarding under
// "auto" and must be left alone, and an explicit choice is never overridden
// (an explicit "auto" with guarding enabled gets a warning instead).
func Test_networkResource_ModifyPlan_dhcpGuardingForcesManual(t *testing.T) {
	r := &networkResource{}
	ctx := context.Background()

	guardingObj := func(enabled bool) types.Object {
		return types.ObjectValueMust(
			dhcpGuardingModel{}.AttributeTypes(),
			map[string]attr.Value{
				"enabled": types.BoolValue(enabled),
				"servers": types.ListNull(types.StringType),
			},
		)
	}

	// explicitAutoConfig builds a Config whose only non-null attribute is
	// setting_preference = "auto", i.e. the practitioner set it explicitly.
	explicitAutoConfig := func(t *testing.T) tfsdk.Config {
		t.Helper()
		var schemaResp fwresource.SchemaResponse
		r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
		schemaType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
		if !ok {
			t.Fatal("schema type is not tftypes.Object")
		}
		attrVals := make(map[string]tftypes.Value, len(schemaType.AttributeTypes))
		for name, typ := range schemaType.AttributeTypes {
			attrVals[name] = tftypes.NewValue(typ, nil)
		}
		attrVals["setting_preference"] = tftypes.NewValue(tftypes.String, "auto")
		return tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaType, attrVals),
		}
	}

	tests := []struct {
		name         string
		purpose      types.String
		tpg          types.Bool
		guarding     types.Object
		explicitAuto bool
		wantPref     string
		wantWarn     bool
	}{
		{
			name:     "guard enabled, default purpose: forces manual",
			purpose:  types.StringNull(),
			tpg:      types.BoolNull(),
			guarding: guardingObj(true),
			wantPref: "manual",
		},
		{
			name:     "guard enabled, corporate: forces manual",
			purpose:  types.StringValue("corporate"),
			tpg:      types.BoolValue(false),
			guarding: guardingObj(true),
			wantPref: "manual",
		},
		{
			name:     "guard enabled, guest: forces manual",
			purpose:  types.StringValue("guest"),
			tpg:      types.BoolValue(false),
			guarding: guardingObj(true),
			wantPref: "manual",
		},
		{
			name:     "guard enabled, vlan-only purpose: left alone",
			purpose:  types.StringValue("vlan-only"),
			tpg:      types.BoolValue(false),
			guarding: guardingObj(true),
			wantPref: "auto",
		},
		{
			name:     "guard enabled, third_party_gateway: left alone",
			purpose:  types.StringNull(),
			tpg:      types.BoolValue(true),
			guarding: guardingObj(true),
			wantPref: "auto",
		},
		{
			name:     "guard disabled: left alone",
			purpose:  types.StringNull(),
			tpg:      types.BoolNull(),
			guarding: guardingObj(false),
			wantPref: "auto",
		},
		{
			name:     "guard block absent: left alone",
			purpose:  types.StringNull(),
			tpg:      types.BoolNull(),
			guarding: types.ObjectNull(dhcpGuardingModel{}.AttributeTypes()),
			wantPref: "auto",
		},
		{
			name:         "explicit auto + guard enabled: warned, not overridden",
			purpose:      types.StringNull(),
			tpg:          types.BoolNull(),
			guarding:     guardingObj(true),
			explicitAuto: true,
			wantPref:     "auto",
			wantWarn:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, config := minimalNetworkPlan(ctx, t)
			if tt.explicitAuto {
				config = explicitAutoConfig(t)
			}

			for _, set := range []struct {
				p path.Path
				v attr.Value
			}{
				{path.Root("ipv6_aliases"), types.ListNull(types.StringType)},
				{path.Root("nat_outbound_ip_addresses"), types.ListNull(
					types.ObjectType{AttrTypes: natOutboundIPAddresses()},
				)},
				{path.Root("dhcp_relay"), types.ObjectNull(dhcpRelayModel{}.AttributeTypes())},
				{path.Root("purpose"), tt.purpose},
				{path.Root("third_party_gateway"), tt.tpg},
				{path.Root("dhcp_guarding"), tt.guarding},
				{path.Root("setting_preference"), types.StringValue("auto")},
			} {
				if d := plan.SetAttribute(ctx, set.p, set.v); d.HasError() {
					t.Fatalf("SetAttribute(%s): %v", set.p, d)
				}
			}

			req := fwresource.ModifyPlanRequest{Plan: plan, Config: config}
			resp := &fwresource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected error: %v", resp.Diagnostics)
			}

			var gotPref types.String
			if d := resp.Plan.GetAttribute(
				ctx,
				path.Root("setting_preference"),
				&gotPref,
			); d.HasError() {
				t.Fatalf("GetAttribute(setting_preference): %v", d)
			}
			if gotPref.ValueString() != tt.wantPref {
				t.Errorf("setting_preference = %q, want %q", gotPref.ValueString(), tt.wantPref)
			}

			gotWarn := resp.Diagnostics.WarningsCount() > 0
			if gotWarn != tt.wantWarn {
				t.Errorf("warnings = %v (%d), want warning: %v",
					resp.Diagnostics, resp.Diagnostics.WarningsCount(), tt.wantWarn)
			}
		})
	}
}

// TestAccNetworkFramework_dhcpGuardingCorporate guards #419 end-to-end: DHCP
// guarding on corporate and guest purpose networks must survive the write —
// the controller resets dhcpguard_enabled under setting_preference "auto", so
// the provider pins "manual" and the applied state must read back enabled with
// the configured servers (a failure here historically surfaced as "Provider
// produced inconsistent result after apply: .dhcp_guarding.enabled").
func TestAccNetworkFramework_dhcpGuardingCorporate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "unifi_network" "guard_corp" {
	name    = "Guard Corporate"
	purpose = "corporate"
	subnet  = "10.0.53.1/24"
	vlan    = 53

	dhcp_guarding = {
		enabled = true
		servers = ["10.0.53.5", "10.0.53.6"]
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.servers.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.servers.0",
						"10.0.53.5",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"setting_preference",
						"manual",
					),
				),
			},
			{
				Config: `
resource "unifi_network" "guard_corp" {
	name    = "Guard Corporate"
	purpose = "guest"
	subnet  = "10.0.53.1/24"
	vlan    = 53

	dhcp_guarding = {
		enabled = true
		servers = ["10.0.53.7"]
	}
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.servers.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"dhcp_guarding.servers.0",
						"10.0.53.7",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.guard_corp",
						"setting_preference",
						"manual",
					),
				),
			},
		},
	})
}
