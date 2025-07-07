package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDeviceFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_device.test", "id"),
					resource.TestCheckResourceAttr("unifi_device.test", "name", "Test Device"),
				),
			},
			{
				ResourceName:      "unifi_device.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"allow_adoption", "forget_on_destroy"},
			},
		},
	})
}

func testAccDeviceFrameworkConfig_basic() string {
	return `
resource "unifi_device" "test" {
	mac  = "aa:bb:cc:dd:ee:ff"
	name = "Test Device"
	allow_adoption = false
	forget_on_destroy = false
}
`
}