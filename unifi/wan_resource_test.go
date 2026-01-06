package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWANFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.test", "id"),
					resource.TestCheckResourceAttr("unifi_wan.test", "name", "Internet 1"),
					resource.TestCheckResourceAttr("unifi_wan.test", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.test", "type_v6", "dhcpv6"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan_enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan", "10"),
					resource.TestCheckResourceAttr("unifi_wan.test", "enabled", "true"),
				),
				ResourceName:  "unifi_wan.test",
				ImportState:   true,
				ImportStateId: "name=Internet 1",
			},
		},
	})
}

func testAccWANFrameworkConfig_basic() string {
	return `
resource "unifi_wan" "test" {
	name         = "Internet 1"
	type         = "dhcp"
	type_v6      = "dhcpv6"
	vlan_enabled = true
	vlan         = 10
	enabled      = true

	dns_preference = "manual"
	dns1           = "1.1.1.1"
	dns2           = "1.0.0.1"

	smartq_enabled   = true
	smartq_up_rate   = 500000
	smartq_down_rate = 500000

	egress_qos_enabled = true
	egress_qos         = 1
	dhcp_cos           = 0
	dhcpv6_cos         = 0

	provider_capabilities = {
		download_kilobits_per_second = 1000000
		upload_kilobits_per_second   = 100000
	}

	load_balance_type   = "weighted"
	load_balance_weight = 75
	failover_priority   = 2
}
`
}
