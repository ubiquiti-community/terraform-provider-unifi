package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserGroupFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user_group.test", "name", "tfacc-group"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_down", "-1"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_up", "-1"),
				),
			},
			{
				ResourceName:      "unifi_user_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccUserGroupFrameworkConfig_basic() string {
	return `
resource "unifi_user_group" "test" {
	name = "tfacc-group"
}
`
}

func TestAccUserGroupFramework_qos(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupFrameworkConfig_qos(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user_group.test", "name", "tfacc-qos-group"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_down", "1000"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_up", "500"),
				),
			},
		},
	})
}

func testAccUserGroupFrameworkConfig_qos() string {
	return `
resource "unifi_user_group" "test" {
	name               = "tfacc-qos-group"
	qos_rate_max_down  = 1000
	qos_rate_max_up    = 500
}
`
}

func TestAccUserGroupFramework_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserGroupFrameworkConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user_group.test", "name", "tfacc-update-group"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_down", "100"),
				),
			},
			{
				Config: testAccUserGroupFrameworkConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user_group.test", "name", "tfacc-update-group-renamed"),
					resource.TestCheckResourceAttr("unifi_user_group.test", "qos_rate_max_down", "200"),
				),
			},
		},
	})
}

func testAccUserGroupFrameworkConfig_update_before() string {
	return `
resource "unifi_user_group" "test" {
	name               = "tfacc-update-group"
	qos_rate_max_down  = 100
}
`
}

func testAccUserGroupFrameworkConfig_update_after() string {
	return `
resource "unifi_user_group" "test" {
	name               = "tfacc-update-group-renamed"
	qos_rate_max_down  = 200
}
`
}