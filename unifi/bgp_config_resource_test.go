package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBGPConfig_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccBGPConfigConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_bgp_config.test", "enabled", "true"),
					resource.TestCheckResourceAttr(
						"unifi_bgp_config.test",
						"description",
						"Test BGP configuration",
					),
				),
			},
			{
				ResourceName:      "unifi_bgp_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const testAccBGPConfigConfig = `
resource "unifi_bgp_config" "test" {
	config      = "router bgp 65001\n neighbor 192.168.1.1 remote-as 65002"
	description = "Test BGP configuration"
	enabled     = true
}
`
