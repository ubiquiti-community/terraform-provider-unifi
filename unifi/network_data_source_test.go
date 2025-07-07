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
					resource.TestCheckResourceAttr("data.unifi_network.test", "name", "test-network"),
					resource.TestCheckResourceAttr("data.unifi_network.test", "purpose", "corporate"),
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
				),
			},
		},
	})
}

func testAccNetworkFrameworkDataSourceConfig_basic() string {
	return `
resource "unifi_network" "test" {
	name     = "test-network"
	purpose  = "corporate"
	subnet   = "10.0.0.0/24"
	vlan_id  = 100
}

data "unifi_network" "test" {
	name = unifi_network.test.name
}
`
}

func testAccNetworkFrameworkDataSourceConfig_byID() string {
	return `
resource "unifi_network" "test" {
	name     = "test-network-by-id"
	purpose  = "corporate"
	subnet   = "10.0.1.0/24"
	vlan_id  = 101
}

data "unifi_network" "test" {
	id = unifi_network.test.id
}
`
}