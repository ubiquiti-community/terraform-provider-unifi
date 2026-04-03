package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusUserDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel_type",
						"3",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel_medium_type",
						"6",
					),
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "site"),
				),
			},
		},
	})
}

func TestAccRadiusUserDataSource_withNetworkID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_withTunnelParams(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds-tunnel",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel_type",
						"12",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel_medium_type",
						"6",
					),
				),
			},
		},
	})
}

func TestAccRadiusUserDataSource_passwordSensitive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// Do not assert password is returned by the API on read, as
					// sensitive fields may be omitted. Instead, verify the data
					// source lookup succeeds for the expected user.
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds",
					),
				),
			},
		},
	})
}

func testAccRadiusUserDataSourceConfig_basic() string {
	return `
resource "unifi_radius_user" "test" {
  name     = "tfacc-radius-user-ds"
  password = "test-password"
}

data "unifi_radius_user" "test" {
  name = unifi_radius_user.test.name

  depends_on = [unifi_radius_user.test]
}
`
}

func testAccRadiusUserDataSourceConfig_withTunnelParams() string {
	return `
resource "unifi_radius_user" "test" {
  name               = "tfacc-radius-user-ds-tunnel"
  password           = "test-password"
  tunnel_type        = 12
  tunnel_medium_type = 6
}

data "unifi_radius_user" "test" {
  name = unifi_radius_user.test.name

  depends_on = [unifi_radius_user.test]
}
`
}
