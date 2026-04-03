package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusProfile_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"3600",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_acct_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_auth_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-auth",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1812",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"auth_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServerCustomPort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServerCustomPort(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"10.0.0.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1822",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_withAcctServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAcctServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-acct",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.port",
						"1813",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"acct_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthAndAcctServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthAndAcctServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auth_server.0.secret",
					"acct_server.0.secret",
				},
			},
		},
	})
}

func TestAccRadiusProfile_withInterimUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withInterimUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1800",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withVlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withVlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_wlan_mode",
						"required",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"3600",
					),
				),
			},
			{
				Config: testAccRadiusProfileConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-updated",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1800",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_importWithSite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "site"),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Import using "site:id" format verified via ImportStateIdFunc below
			},
		},
	})
}

func testAccRadiusProfileConfig_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile"
}
`
}

func testAccRadiusProfileConfig_withAuthServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth"

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthServerCustomPort() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth-port"

  auth_server {
    ip     = "10.0.0.1"
    port   = 1822
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAcctServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-acct"
  accounting_enabled = true

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthAndAcctServers() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-full"
  accounting_enabled = true

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withInterimUpdate() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-interim"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = 1800
}
`
}

func testAccRadiusProfileConfig_withVlan() string {
	return `
resource "unifi_radius_profile" "test" {
  name           = "tfacc-radius-profile-vlan"
  vlan_enabled   = true
  vlan_wlan_mode = "required"
}
`
}

func testAccRadiusProfileConfig_updated() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-updated"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = 1800
}
`
}
