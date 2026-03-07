package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNetworkMembersGroupListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkMembersGroupListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_network_members_group_list.test",
						"groups.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_network_members_group_list.test",
						"site",
						"default",
					),
				),
			},
		},
	})
}

func testAccNetworkMembersGroupListDataSourceConfig_basic() string {
	return `
data "unifi_network_members_group_list" "test" {
}
`
}
