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
					// Verify ip_subnet and subnet fields are populated (regression for struct/object mismatch bug)
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "subnet"),
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "ip_subnet"),
					// Verify multicast_dns is readable (regression for struct/object mismatch bug)
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
					// Verify ip_subnet field is accessible via ID lookup
					resource.TestCheckResourceAttrSet("data.unifi_network.test", "ip_subnet"),
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

output "network_ip_subnet" {
	value = data.unifi_network.test.ip_subnet
}

output "network_mdns" {
	value = data.unifi_network.test.multicast_dns
}
`
}
