package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientInfoListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientInfoListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// Verify the data source has clients attribute
					resource.TestCheckResourceAttrSet("data.unifi_client_info_list.test", "clients.#"),
					// Verify site is set
					resource.TestCheckResourceAttr("data.unifi_client_info_list.test", "site", "default"),
				),
			},
		},
	})
}

func testAccClientInfoListDataSourceConfig_basic() string {
	return `
data "unifi_client_info_list" "test" {
}
`
}
