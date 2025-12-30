package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user.test", "name", "tfacc-user"),
					resource.TestCheckResourceAttr("unifi_user.test", "mac", "01:23:45:67:89:ab"),
					resource.TestCheckResourceAttr("unifi_user.test", "blocked", "false"),
				),
			},
			{
				ResourceName:            "unifi_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"allow_existing", "skip_forget_on_destroy"},
			},
		},
	})
}

func testAccUserFrameworkConfig_basic() string {
	return `
resource "unifi_user" "test" {
	name = "tfacc-user"
	mac  = "01:23:45:67:89:ab"
}
`
}

func TestAccUserFramework_blocked(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserFrameworkConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_user.test", "name", "tfacc-blocked-user"),
					resource.TestCheckResourceAttr("unifi_user.test", "blocked", "true"),
					resource.TestCheckResourceAttr(
						"unifi_user.test",
						"note",
						"Blocked for testing",
					),
				),
			},
		},
	})
}

func testAccUserFrameworkConfig_blocked() string {
	return `
resource "unifi_user" "test" {
	name    = "tfacc-blocked-user"
	mac     = "01:23:45:67:89:ac"
	blocked = true
	note    = "Blocked for testing"
}
`
}

func TestAccUserFramework_fixedIP(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserFrameworkConfig_fixedIP(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_user.test",
						"name",
						"tfacc-fixed-ip-user",
					),
					resource.TestCheckResourceAttr("unifi_user.test", "fixed_ip", "10.0.0.100"),
				),
			},
		},
	})
}

func testAccUserFrameworkConfig_fixedIP() string {
	return `
data "unifi_network" "default" {
	name = "Default"
}

resource "unifi_user" "test" {
	name       = "tfacc-fixed-ip-user"
	mac        = "01:23:45:67:89:ad"
	fixed_ip   = "10.0.0.100"
	network_id = data.unifi_network.default.id
}
`
}
