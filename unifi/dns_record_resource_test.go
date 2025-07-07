package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSRecordFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             nil, // TODO: implement check destroy
		Steps: []resource.TestStep{
			{
				Config: testAccDNSRecordFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_dns_record.test", "name", "test-record"),
					resource.TestCheckResourceAttr(
						"unifi_dns_record.test",
						"value",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr("unifi_dns_record.test", "port", "80"),
					resource.TestCheckResourceAttr("unifi_dns_record.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_dns_record.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDNSRecordFrameworkConfig_basic() string {
	return `
resource "unifi_dns_record" "test" {
	name    = "test-record"
	value   = "192.168.1.100"
	port    = 80
	enabled = true
}
`
}
