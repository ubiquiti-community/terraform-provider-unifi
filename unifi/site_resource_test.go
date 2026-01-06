package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_site.test", "description", "tfacc-test"),
					resource.TestCheckResourceAttrSet("unifi_site.test", "name"),
				),
				ResourceName:  "unifi_site.test",
				ImportState:   true,
				ImportStateId: "default",
			},
		},
	})
}

func testAccSiteFrameworkConfig_basic() string {
	return `
resource "unifi_site" "test" {
	name        = "default"
	description = "tfacc-test"
}
`
}
