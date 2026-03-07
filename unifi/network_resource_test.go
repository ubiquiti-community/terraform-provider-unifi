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
					resource.TestCheckResourceAttr("unifi_network.test", "name", "Test"),
					resource.TestCheckResourceAttr("unifi_network.test", "purpose", "corporate"),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"subnet",
						"192.168.2.1/24",
					),
					resource.TestCheckResourceAttr("unifi_network.test", "vlan_id", "10"),
					resource.TestCheckResourceAttr("unifi_network.test", "dhcp_enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"dhcp_start",
						"192.168.2.6",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"dhcp_stop",
						"192.168.2.254",
					),
				),
				ImportState:   false,
				ImportStateId: "name=Test",
				ResourceName:  "unifi_network.test",
			},
		},
	})
}

func testAccNetworkFrameworkConfig_basic() string {
	return `
resource "unifi_network" "test" {
	name    = "Test"
	purpose = "corporate"
	subnet  = "192.168.2.1/24"
	vlan_id = 10

	dhcp_enabled = true
	dhcp_start   = "192.168.2.6"
	dhcp_stop    = "192.168.2.254"
}
`
}

func TestAccNetworkFramework_vlanOnly(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_vlanOnly(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_network.vlan_only", "name", "VLAN_92"),
					resource.TestCheckResourceAttr(
						"unifi_network.vlan_only",
						"purpose",
						"vlan-only",
					),
					resource.TestCheckResourceAttr("unifi_network.vlan_only", "vlan_id", "92"),
				),
			},
		},
	})
}

func testAccNetworkFrameworkConfig_vlanOnly() string {
	return `
resource "unifi_network" "vlan_only" {
	name    = "VLAN_92"
	purpose = "vlan-only"
	vlan_id = 92
}
`
}

func TestAccNetworkFramework_guest(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_guest(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_network.guest", "name", "Guest_Network"),
					resource.TestCheckResourceAttr("unifi_network.guest", "purpose", "guest"),
					resource.TestCheckResourceAttr("unifi_network.guest", "vlan_id", "50"),
				),
			},
		},
	})
}

func testAccNetworkFrameworkConfig_guest() string {
	return `
resource "unifi_network" "guest" {
	name    = "Guest_Network"
	purpose = "guest"
	subnet  = "192.168.50.1/24"
	vlan_id = 50

	dhcp_enabled = true
	dhcp_start   = "192.168.50.6"
	dhcp_stop    = "192.168.50.254"
}
`
}
