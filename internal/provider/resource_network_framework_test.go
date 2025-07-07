package provider

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
					resource.TestCheckResourceAttr("unifi_network.test", "name", "tfacc-framework"),
					resource.TestCheckResourceAttr("unifi_network.test", "purpose", "corporate"),
					resource.TestCheckResourceAttr("unifi_network.test", "subnet", "10.0.0.0/24"),
					resource.TestCheckResourceAttr("unifi_network.test", "vlan_id", "10"),
					resource.TestCheckResourceAttr("unifi_network.test", "dhcp_enabled", "true"),
					resource.TestCheckResourceAttr("unifi_network.test", "dhcp_start", "10.0.0.6"),
					resource.TestCheckResourceAttr("unifi_network.test", "dhcp_stop", "10.0.0.254"),
				),
			},
			{
				ResourceName:      "unifi_network.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNetworkFramework_importByName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkFrameworkConfig_basic(),
			},
			{
				ResourceName:      "unifi_network.test",
				ImportState:       true,
				ImportStateId:     "name=tfacc-framework",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccNetworkFrameworkConfig_basic() string {
	return `
resource "unifi_network" "test" {
	name    = "tfacc-framework"
	purpose = "corporate"
	subnet  = "10.0.0.0/24"
	vlan_id = 10

	dhcp_enabled = true
	dhcp_start   = "10.0.0.6"
	dhcp_stop    = "10.0.0.254"
}
`
}