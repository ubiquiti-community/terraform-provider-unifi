package unifi

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccBGPConfig_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccBGPConfigConfig,
				ExpectError: regexp.MustCompile(".*"),
			},
		},
	})
}

const testAccBGPConfigConfig = `
resource "unifi_bgp" "test" {
	config      = "router bgp 65001\n neighbor 192.168.1.1 remote-as 65002"
	description = "Test BGP configuration"
	enabled     = true
}
`
