package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVirtualNetworkFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualNetworkFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test",
						"name",
						"Test VLAN",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test",
						"subnet",
						"192.168.10.1/24",
					),
					resource.TestCheckResourceAttr("unifi_virtual_network.test", "vlan", "10"),
					resource.TestCheckResourceAttr("unifi_virtual_network.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_virtual_network.test",
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

func TestAccVirtualNetworkFramework_dhcp(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualNetworkFrameworkConfig_dhcp(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"name",
						"Test DHCP Network",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"subnet",
						"192.168.20.1/24",
					),
					resource.TestCheckResourceAttr("unifi_virtual_network.test_dhcp", "vlan", "20"),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"dhcp_server.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"dhcp_server.start",
						"192.168.20.10",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"dhcp_server.stop",
						"192.168.20.254",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_dhcp",
						"dhcp_server.leasetime",
						"86400",
					),
				),
			},
			{
				ResourceName:      "unifi_virtual_network.test_dhcp",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test DHCP Network",
			},
		},
	})
}

func TestAccVirtualNetworkFramework_guest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualNetworkFrameworkConfig_guest(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_guest",
						"name",
						"Guest Network",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_guest",
						"subnet",
						"192.168.30.1/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_guest",
						"vlan",
						"30",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_guest",
						"internet_access_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_guest",
						"network_isolation_enabled",
						"true",
					),
				),
			},
		},
	})
}

func testAccVirtualNetworkFrameworkConfig_basic() string {
	return `
resource "unifi_virtual_network" "test" {
	name      = "Test VLAN"
	subnet    = "192.168.10.1/24"
	vlan      = 10
	enabled   = true
}
`
}

func testAccVirtualNetworkFrameworkConfig_dhcp() string {
	return `
resource "unifi_virtual_network" "test_dhcp" {
	name      = "Test DHCP Network"
	subnet    = "192.168.20.1/24"
	vlan      = 20

	dhcp_server = {
		enabled   = true
		start     = "192.168.20.10"
		stop      = "192.168.20.254"
		leasetime = 86400
	}
}
`
}

func testAccVirtualNetworkFrameworkConfig_guest() string {
	return `
resource "unifi_virtual_network" "test_guest" {
	name                      = "Guest Network"
	subnet                    = "192.168.30.1/24"
	vlan                      = 30
	internet_access_enabled   = true
	network_isolation_enabled = true
}
`
}

func TestAccVirtualNetworkFramework_thirdPartyGateway(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualNetworkFrameworkConfig_thirdPartyGateway(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"name",
						"Test Third Party",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"vlan",
						"3",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"third_party_gateway",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"dhcp_guarding.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"dhcp_guarding.servers.#",
						"2",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"dhcp_guarding.servers.0",
						"192.168.20.20",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party",
						"dhcp_guarding.servers.1",
						"192.168.20.21",
					),
				),
			},
			{
				ResourceName:      "unifi_virtual_network.test_third_party",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test Third Party",
				// These fields are not relevant to vlan-only networks and are not
				// returned by the API, so they cannot be recovered during import.
				ImportStateVerifyIgnore: []string{
					"subnet",
					"auto_scale_enabled",
					"gateway_type",
					"setting_preference",
					"mdns_enabled",
					"ipv6_interface_type",
					"lte_lan_enabled",
					"internet_access_enabled",
				},
			},
		},
	})
}

func TestAccVirtualNetworkFramework_thirdPartyGatewayMinimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualNetworkFrameworkConfig_thirdPartyGatewayMinimal(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party_min",
						"name",
						"Test Third Party Minimal",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party_min",
						"vlan",
						"4",
					),
					resource.TestCheckResourceAttr(
						"unifi_virtual_network.test_third_party_min",
						"third_party_gateway",
						"true",
					),
				),
			},
		},
	})
}

func testAccVirtualNetworkFrameworkConfig_thirdPartyGateway() string {
	return `
resource "unifi_virtual_network" "test_third_party" {
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

func testAccVirtualNetworkFrameworkConfig_thirdPartyGatewayMinimal() string {
	return `
resource "unifi_virtual_network" "test_third_party_min" {
	name                = "Test Third Party Minimal"
	subnet              = "192.168.20.1/24"
	vlan                = 4
	third_party_gateway = true
}
`
}
