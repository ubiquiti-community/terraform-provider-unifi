package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccFirewallPolicy_specificPort exercises the round-trip of a SPECIFIC
// port match on both the destination and source endpoints. It guards the fix
// for #207, where the `port` value was unrepresentable and silently dropped.
//
// Zone-based firewall policies require an operational gateway (UniFi Network
// 8.x+), so this test does not run against the gateway-less demo controller
// used by the default acceptance harness.
func TestAccFirewallPolicy_specificPort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallPolicyConfig_specificPort(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_policy.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_policy.test",
						"destination.port_matching_type",
						"SPECIFIC",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_policy.test",
						"destination.port",
						"8080",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_policy.test",
						"source.port_matching_type",
						"SPECIFIC",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_policy.test",
						"source.port",
						"443",
					),
				),
			},
			{
				ResourceName:      "unifi_firewall_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccFirewallPolicyConfig_specificPort() string {
	return `
data "unifi_firewall_zone" "internal" {
  name = "Internal"
}

data "unifi_firewall_zone" "external" {
  name = "External"
}

resource "unifi_firewall_policy" "test" {
  name     = "tfacc-fwpolicy-specific-port"
  action   = "ALLOW"
  protocol = "tcp"

  source = {
    zone_id            = data.unifi_firewall_zone.internal.id
    matching_target    = "ANY"
    port_matching_type = "SPECIFIC"
    port               = 443
  }

  destination = {
    zone_id            = data.unifi_firewall_zone.external.id
    matching_target    = "ANY"
    port_matching_type = "SPECIFIC"
    port               = 8080
  }
}
`
}
