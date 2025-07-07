package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserFrameworkDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserFrameworkDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.unifi_user.test", "mac", "01:23:45:67:89:ae"),
					resource.TestCheckResourceAttr("data.unifi_user.test", "name", "tfacc-data-user"),
					resource.TestCheckResourceAttrSet("data.unifi_user.test", "id"),
				),
			},
		},
	})
}

func testAccUserFrameworkDataSourceConfig_basic() string {
	return `
resource "unifi_user" "test" {
	name = "tfacc-data-user"
	mac  = "01:23:45:67:89:ae"
}

data "unifi_user" "test" {
	mac = unifi_user.test.mac
}
`
}