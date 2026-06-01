package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusUser_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             nil, // TODO: implement check destroy
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"name",
						"test-account",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"password",
						"test-password",
					),
					resource.TestCheckResourceAttr("unifi_radius_user.test", "tunnel_type", "3"),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.test",
						"tunnel_medium_type",
						"6",
					),
				),
			},
			{
				ResourceName:            "unifi_radius_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"}, // Password is not returned by API
			},
		},
	})
}

func testAccRadiusUserConfig_basic() string {
	return `
resource "unifi_radius_user" "test" {
	name     = "test-account"
	password = "test-password"
}
`
}

func TestAccRadiusUser_vlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_vlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.vlan", "vlan", "100"),
					resource.TestCheckResourceAttr(
						"unifi_radius_user.vlan",
						"tunnel_config_type",
						"802.1x",
					),
				),
			},
			{
				ResourceName:            "unifi_radius_user.vlan",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccRadiusUserConfig_vlan() string {
	return `
resource "unifi_radius_user" "vlan" {
	name               = "test-account-vlan"
	password           = "test-password"
	vlan               = 100
	tunnel_config_type = "802.1x"
}
`
}
