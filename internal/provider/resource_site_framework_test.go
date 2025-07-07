package provider

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
			},
			{
				ResourceName:      "unifi_site.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSiteFrameworkConfig_basic() string {
	return `
resource "unifi_site" "test" {
	description = "tfacc-test"
}
`
}

func TestAccSiteFramework_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteFrameworkConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_site.test", "description", "tfacc-test-before"),
				),
			},
			{
				Config: testAccSiteFrameworkConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_site.test", "description", "tfacc-test-after"),
				),
			},
		},
	})
}

func testAccSiteFrameworkConfig_update_before() string {
	return `
resource "unifi_site" "test" {
	description = "tfacc-test-before"
}
`
}

func testAccSiteFrameworkConfig_update_after() string {
	return `
resource "unifi_site" "test" {
	description = "tfacc-test-after"
}
`
}