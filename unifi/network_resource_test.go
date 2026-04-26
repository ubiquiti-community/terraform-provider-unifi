package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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
						"86400",
					),
				),
			},
			{
				ResourceName:      "unifi_network.test_dhcp",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     "name=Test DHCP Network",
			},
		},
	})
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
		leasetime = 86400
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
		PreCheck:                 func() { preCheck(t) },
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
						"86400",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test_ipv6_static",
						"ipv6_ra_preferred_lifetime",
						"14400",
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
	ipv6_ra_valid_lifetime  = 86400
	ipv6_ra_preferred_lifetime = 14400
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
