package unifi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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

// TestAccRadiusUser_tunnelType13 verifies that tunnel_type accepts 13 (VLAN),
// which the controller allows (1-13) but the provider previously capped at 12.
func TestAccRadiusUser_tunnelType13(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserConfig_tunnelType13(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.tt13", "tunnel_type", "13"),
				),
			},
			{
				ResourceName:            "unifi_radius_user.tt13",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func testAccRadiusUserConfig_tunnelType13() string {
	return `
resource "unifi_radius_user" "tt13" {
	name        = "test-account-tt13"
	password    = "test-password"
	tunnel_type = 13
}
`
}

// TestAccRadiusUser_moveFromAccount exercises the ResourceWithMoveState support
// (#222): a deprecated unifi_account can be migrated to unifi_radius_user with a
// `moved` block, in place. The move is proven by the target keeping the source's
// ID — a destroy/recreate would assign a new one.
func TestAccRadiusUser_moveFromAccount(t *testing.T) {
	var accountID string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Create the deprecated resource and capture its ID.
			{
				Config: testAccRadiusUserConfig_accountForMove(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_account.move", "name", "move-account"),
					testAccCaptureResourceID("unifi_account.move", &accountID),
				),
			},
			// Move it to unifi_radius_user; the underlying object (ID) must survive.
			{
				Config: testAccRadiusUserConfig_radiusUserAfterMove(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_radius_user.move", "name", "move-account"),
					resource.TestCheckResourceAttrPtr("unifi_radius_user.move", "id", &accountID),
				),
			},
		},
	})
}

func testAccRadiusUserConfig_accountForMove() string {
	return `
resource "unifi_account" "move" {
	name     = "move-account"
	password = "move-password"
}
`
}

func testAccRadiusUserConfig_radiusUserAfterMove() string {
	return `
resource "unifi_radius_user" "move" {
	name     = "move-account"
	password = "move-password"
}

moved {
	from = unifi_account.move
	to   = unifi_radius_user.move
}
`
}

// testAccCaptureResourceID stores the primary ID of a resource into dst so a
// later step can assert it is unchanged.
func testAccCaptureResourceID(name string, dst *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource not found: %s", name)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %s has no ID set", name)
		}
		*dst = rs.Primary.ID
		return nil
	}
}
