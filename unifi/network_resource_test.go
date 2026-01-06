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
					resource.TestCheckResourceAttr("unifi_network.test", "name", "Default"),
					resource.TestCheckResourceAttr("unifi_network.test", "purpose", "corporate"),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"subnet",
						"192.168.1.1/24",
					),
					resource.TestCheckResourceAttr("unifi_network.test", "vlan_id", "10"),
					resource.TestCheckResourceAttr("unifi_network.test", "dhcp_enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"dhcp_start",
						"192.168.1.6",
					),
					resource.TestCheckResourceAttr(
						"unifi_network.test",
						"dhcp_stop",
						"192.168.1.254",
					),
				),
				ImportState:   true,
				ImportStateId: "name=Default",
				ResourceName:  "unifi_network.test",
			},
		},
	})
}

func testAccNetworkFrameworkConfig_basic() string {
	return `
resource "unifi_network" "test" {
	name    = "Default"
	purpose = "corporate"
	subnet  = "192.168.1.1/24"
	vlan_id = 10

	dhcp_enabled = true
	dhcp_start   = "192.168.1.6"
	dhcp_stop    = "192.168.1.254"
}
`
}
