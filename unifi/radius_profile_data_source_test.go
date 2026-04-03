package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusProfileDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-ds",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"use_usg_acct_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"use_usg_auth_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"vlan_enabled",
						"false",
					),
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "site"),
					resource.TestCheckResourceAttrSet(
						"data.unifi_radius_profile.test",
						"interim_update_interval",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfileDataSource_withAuthServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_withAuthServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-ds-auth",
					),
					// Data source does not expose auth_server/acct_server blocks
				),
			},
		},
	})
}

func TestAccRadiusProfileDataSource_accountingEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_accountingEnabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_interval",
						"900",
					),
				),
			},
		},
	})
}

func testAccRadiusProfileDataSourceConfig_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-ds"
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}

func testAccRadiusProfileDataSourceConfig_withAuthServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-ds-auth"

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-secret"
  }
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}

func testAccRadiusProfileDataSourceConfig_accountingEnabled() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-ds-acct"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = 900
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}
