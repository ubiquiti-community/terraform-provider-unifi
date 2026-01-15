package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_client.test", "name", "tfacc-client"),
					resource.TestCheckResourceAttr("unifi_client.test", "mac", "01:23:45:67:89:ab"),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "false"),
				),
			},
			{
				ResourceName:            "unifi_client.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_existing", "skip_forget_on_destroy"},
			},
		},
	})
}

func testAccClientFrameworkConfig_basic() string {
	return `
resource "unifi_client" "test" {
	name = "tfacc-client"
	mac  = "01:23:45:67:89:ab"
}
`
}

func TestAccClientFramework_blocked(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-blocked-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "blocked", "true"),
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"note",
						"Blocked for testing",
					),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_blocked() string {
	return `
resource "unifi_client" "test" {
	name    = "tfacc-blocked-client"
	mac     = "01:23:45:67:89:ac"
	blocked = true
	note    = "Blocked for testing"
}
`
}

func TestAccClientFramework_fixedIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientFrameworkConfig_fixedIP(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client.test",
						"name",
						"tfacc-fixed-ip-client",
					),
					resource.TestCheckResourceAttr("unifi_client.test", "fixed_ip", "10.0.0.100"),
				),
			},
		},
	})
}

func testAccClientFrameworkConfig_fixedIP() string {
	return `
data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_client" "test" {
	name       = "tfacc-fixed-ip-client"
	mac        = "01:23:45:67:89:ad"
	fixed_ip   = "10.0.0.100"
	network_id = data.unifi_network.default.id
}
`
}
