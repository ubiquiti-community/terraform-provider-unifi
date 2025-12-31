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
