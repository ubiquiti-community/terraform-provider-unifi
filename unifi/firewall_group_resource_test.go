package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFirewallGroupFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallGroupFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"name",
						"Test Address Group",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"type",
						"address-group",
					),
					resource.TestCheckResourceAttr("unifi_firewall_group.test", "members.#", "2"),
				),
			},
			{
				ResourceName:      "unifi_firewall_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallGroupFramework_portGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallGroupFrameworkConfig_portGroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"name",
						"Test Port Group",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"type",
						"port-group",
					),
					resource.TestCheckResourceAttr("unifi_firewall_group.test", "members.#", "3"),
				),
			},
		},
	})
}

func testAccFirewallGroupFrameworkConfig_basic() string {
	return `
resource "unifi_firewall_group" "test" {
	name = "Test Address Group"
	type = "address-group"
	members = [
		"192.168.1.10",
		"192.168.1.20"
	]
}
`
}

func testAccFirewallGroupFrameworkConfig_portGroup() string {
	return `
resource "unifi_firewall_group" "test" {
	name = "Test Port Group"
	type = "port-group"
	members = [
		"80",
		"443",
		"8080"
	]
}
`
}
