package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkFrameworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"name",
						"Default",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"purpose",
						"corporate",
					),
					// Verify subnet is populated
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "subnet"),
					// Verify multicast_dns is readable
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "multicast_dns"),
				),
			},
		},
	})
}

func TestAccNetworkFrameworkDataSource_byID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkDataSourceConfig_byID(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "id"),
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "name"),
					// Verify subnet field is accessible via ID lookup
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "subnet"),
				),
			},
		},
	})
}

func TestAccNetworkFrameworkDataSource_outputFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Reproduce the exact usage pattern from the bug report
				Config: testAccNetworkFrameworkDataSourceConfig_outputFields(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "subnet"),
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "multicast_dns"),
				),
			},
		},
	})
}

func testAccNetworkFrameworkDataSourceConfig_basic() string {
	return `
data "unifi_network" "test" {
	name = "Default"
}
`
}

func testAccNetworkFrameworkDataSourceConfig_byID() string {
	return `
data "unifi_network" "default" {
	name = "Default"
}

data "unifi_network" "test" {
	id = data.unifi_network.default.id
}
`
}

func testAccNetworkFrameworkDataSourceConfig_outputFields() string {
	return `
data "unifi_network" "test" {
	name = "Default"
}

output "network_subnet" {
	value = data.unifi_network.test.subnet
}

output "network_multicast_dns" {
	value = data.unifi_network.test.multicast_dns
}
`
}

func TestAccNetworkFrameworkDataSource_dhcpGuardingServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkDataSourceConfig_dhcpGuardingServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_guarding_ds",
						"dhcp_guarding.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_guarding_ds",
						"dhcp_guarding.servers.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_guarding_ds",
						"dhcp_guarding.servers.0",
						"10.0.51.1",
					),
				),
			},
		},
	})
}

func TestAccNetworkFrameworkDataSource_dhcpRelayServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkDataSourceConfig_dhcpRelayServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_relay_ds",
						"dhcp_relay.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_relay_ds",
						"dhcp_relay.servers.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test_relay_ds",
						"dhcp_relay.servers.0",
						"192.168.52.1",
					),
				),
			},
		},
	})
}

func testAccNetworkFrameworkDataSourceConfig_dhcpGuardingServers() string {
	return `
resource "unifi_network" "test_guarding" {
	name                = "Test DHCP Guarding DS"
	subnet              = "10.0.51.1/24"
	vlan                = 51
	third_party_gateway = true

	dhcp_guarding = {
		enabled = true
		servers = ["10.0.51.1"]
	}
}

data "unifi_network" "test_guarding_ds" {
	name       = unifi_network.test_guarding.name
	depends_on = [unifi_network.test_guarding]
}
`
}

func testAccNetworkFrameworkDataSourceConfig_dhcpRelayServers() string {
	return `
resource "unifi_network" "test_relay_ds_net" {
	name   = "Test DHCP Relay DS"
	subnet = "192.168.52.1/24"
	vlan   = 52

	dhcp_relay = {
		enabled = true
		servers = ["192.168.52.1"]
	}
}

data "unifi_network" "test_relay_ds" {
	name       = unifi_network.test_relay_ds_net.name
	depends_on = [unifi_network.test_relay_ds_net]
}
`
}

func TestAccNetworkFrameworkDataSource_ipv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkDataSourceConfig_ipv6(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_interface_type",
						"static",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_static_subnet",
						"fd02::1/64",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_ra",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_ra_priority",
						"medium",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_ra_preferred_lifetime",
						"4h0m0s",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"ipv6_ra_valid_lifetime",
						"24h0m0s",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"dhcp_v6_server.enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"dhcp_v6_server.dns_auto",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"dhcp_v6_server.start",
						"::2",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"dhcp_v6_server.stop",
						"::7d1",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network.test",
						"dhcp_v6_server.lease",
						"86400",
					),
				),
			},
		},
	})
}

func testAccNetworkFrameworkDataSourceConfig_ipv6() string {
	return `
resource "unifi_network" "test_ipv6_ds" {
	name                    = "Test IPv6 DS"
	subnet                  = "192.168.70.1/24"
	vlan                    = 70
	ipv6_interface_type     = "static"
	ipv6_static_subnet      = "fd02::1/64"
	ipv6_ra                 = true
	ipv6_ra_priority        = "medium"
	ipv6_ra_preferred_lifetime = "4h0m0s"
	ipv6_ra_valid_lifetime  = "24h0m0s"

	dhcp_v6_server = {
		enabled     = true
		dns_auto    = false
		start       = "::2"
		stop        = "::7d1"
		lease       = 86400
	}
}

data "unifi_network" "test" {
	name       = unifi_network.test_ipv6_ds.name
	depends_on = [unifi_network.test_ipv6_ds]
}
`
}
