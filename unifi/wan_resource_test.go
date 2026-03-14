package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWANFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.test", "id"),
					resource.TestCheckResourceAttr("unifi_wan.test", "name", "test-wan"),
					resource.TestCheckResourceAttr("unifi_wan.test", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wan.test", "vlan.id", "10"),
					resource.TestCheckResourceAttr("unifi_wan.test", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_wan.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWANFramework_minimal verifies that a WAN with no optional nested objects
// can be created and imported without "was null, but now..." errors from API defaults.
func TestAccWANFramework_minimal(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_minimal(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.minimal", "id"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "name", "test-wan-minimal"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.minimal", "enabled", "true"),
				),
			},
			{
				ResourceName:      "unifi_wan.minimal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccWANFramework_withNestedObjects verifies that explicitly configured nested
// objects are preserved through create, read, and import.
func TestAccWANFramework_withNestedObjects(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWANFrameworkConfig_withNestedObjects(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wan.nested", "id"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "name", "test-wan-nested"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "type", "dhcp"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "enabled", "true"),
					// VLAN
					resource.TestCheckResourceAttr("unifi_wan.nested", "vlan.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "vlan.id", "20"),
					// DNS
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.preference", "manual"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.primary", "8.8.8.8"),
					resource.TestCheckResourceAttr("unifi_wan.nested", "dns.secondary", "8.8.4.4"),
					// Load Balance
					resource.TestCheckResourceAttrSet(
						"unifi_wan.nested",
						"load_balance.failover_priority",
					),
				),
			},
			{
				ResourceName:      "unifi_wan.nested",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccWANFrameworkConfig_basic() string {
	return `
resource "unifi_wan" "test" {
	name    = "test-wan"
	type    = "dhcp"
	enabled = true

	vlan = {
		enabled = true
		id      = 10
	}
}
`
}

func testAccWANFrameworkConfig_minimal() string {
	return `
resource "unifi_wan" "minimal" {
	name    = "test-wan-minimal"
	type    = "dhcp"
	enabled = true
}
`
}

func testAccWANFrameworkConfig_withNestedObjects() string {
	return `
resource "unifi_wan" "nested" {
	name    = "test-wan-nested"
	type    = "dhcp"
	enabled = true

	vlan = {
		enabled = true
		id      = 20
	}

	dns = {
		preference = "manual"
		primary    = "8.8.8.8"
		secondary  = "8.8.4.4"
	}

	load_balance = {
		failover_priority = 1
	}
}
`
}
