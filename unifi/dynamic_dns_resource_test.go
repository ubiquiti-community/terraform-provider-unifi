package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDynamicDNS_dyndns(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDynamicDNSConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_dynamic_dns.test", "service", "dyndns"),
					resource.TestCheckResourceAttr(
						"unifi_dynamic_dns.test",
						"host_name",
						"test.example.com",
					),
					resource.TestCheckResourceAttr("unifi_dynamic_dns.test", "interface", "wan"),
				),
			},
			{
				ResourceName:      "unifi_dynamic_dns.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"password",
				}, // Password is sensitive and not returned
			},
		},
	})
}

const testAccDynamicDNSConfig = `
resource "unifi_dynamic_dns" "test" {
	service = "dyndns"
	
	host_name = "test.example.com"

	server   = "dyndns.example.com"
	login    = "testuser"
	password = "password"
}
`
