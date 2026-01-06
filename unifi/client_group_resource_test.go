package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientGroupFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientGroupFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_client_group.test", "name", "tfacc-group"),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_down",
						"-1",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_up",
						"-1",
					),
				),
			},
			{
				ResourceName:      "unifi_client_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccClientGroupFrameworkConfig_basic() string {
	return `
resource "unifi_client_group" "test" {
	name = "tfacc-group"
}
`
}

func TestAccClientGroupFramework_qos(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientGroupFrameworkConfig_qos(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"name",
						"tfacc-qos-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_down",
						"1000",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_up",
						"500",
					),
				),
			},
		},
	})
}

func testAccClientGroupFrameworkConfig_qos() string {
	return `
resource "unifi_client_group" "test" {
	name               = "tfacc-qos-group"
	qos_rate_max_down  = 1000
	qos_rate_max_up    = 500
}
`
}

func TestAccClientGroupFramework_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientGroupFrameworkConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"name",
						"tfacc-update-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_down",
						"100",
					),
				),
			},
			{
				Config: testAccClientGroupFrameworkConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"name",
						"tfacc-update-group-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_group.test",
						"qos_rate_max_down",
						"200",
					),
				),
			},
		},
	})
}

func testAccClientGroupFrameworkConfig_update_before() string {
	return `
resource "unifi_client_group" "test" {
	name               = "tfacc-update-group"
	qos_rate_max_down  = 100
}
`
}

func testAccClientGroupFrameworkConfig_update_after() string {
	return `
resource "unifi_client_group" "test" {
	name               = "tfacc-update-group-renamed"
	qos_rate_max_down  = 200
}
`
}
