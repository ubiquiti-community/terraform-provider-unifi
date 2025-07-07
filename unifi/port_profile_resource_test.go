package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPortProfileFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortProfileFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.test", "id"),
					resource.TestCheckResourceAttr("unifi_port_profile.test", "name", "Test Port Profile"),
					resource.TestCheckResourceAttr("unifi_port_profile.test", "autoneg", "true"),
				),
			},
			{
				ResourceName:      "unifi_port_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPortProfileFrameworkConfig_basic() string {
	return `
resource "unifi_port_profile" "test" {
	name     = "Test Port Profile"
	autoneg  = true
	forward  = "native"
	op_mode  = "switch"
}
`
}