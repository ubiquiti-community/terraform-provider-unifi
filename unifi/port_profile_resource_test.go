package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPortProfileFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortProfileFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.test",
						"name",
						"Test Port Profile",
					),
					resource.TestCheckResourceAttr("unifi_port_profile.test", "autoneg", "true"),
				),
			},
			{
				ResourceName:      "unifi_port_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPortProfileFrameworkConfig_basic() string {
	return `
resource "unifi_port_profile" "test" {
	name     = "Test Port Profile"
	autoneg  = true
	op_mode  = "switch"
}
`
}

func TestAccPortProfileFramework_vlanFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortProfileFrameworkConfig_vlanFields(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.vlan", "id"),
					// native_networkconf_id is now persisted/computed (was previously a no-op).
					resource.TestCheckResourceAttrSet(
						"unifi_port_profile.vlan",
						"native_networkconf_id",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.vlan",
						"setting_preference",
						"manual",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.vlan",
						"port_keepalive_enabled",
						"true",
					),
				),
			},
			{
				ResourceName:      "unifi_port_profile.vlan",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPortProfileFrameworkConfig_vlanFields() string {
	return `
resource "unifi_port_profile" "vlan" {
	name                   = "Test Port Profile VLAN"
	op_mode                = "switch"
	setting_preference     = "manual"
	port_keepalive_enabled = true
}
`
}
