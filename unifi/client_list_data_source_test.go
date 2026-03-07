package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.test",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.test",
						"site",
						"default",
					),
				),
			},
		},
	})
}

func TestAccClientListDataSource_filtered(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_wired(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.wired",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.wired",
						"site",
						"default",
					),
				),
			},
			{
				Config: testAccClientListDataSourceConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.blocked",
						"clients.#",
					),
				),
			},
		},
	})
}

func testAccClientListDataSourceConfig_basic() string {
	return `
data "unifi_client_list" "test" {
}
`
}

func testAccClientListDataSourceConfig_wired() string {
	return `
data "unifi_client_list" "wired" {
  wired = true
}
`
}

func testAccClientListDataSourceConfig_blocked() string {
	return `
data "unifi_client_list" "blocked" {
  blocked = false
}
`
}
