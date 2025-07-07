package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAccountFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             nil, // TODO: implement check destroy
		Steps: []resource.TestStep{
			{
				Config: testAccAccountFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_account.test", "name", "test-account"),
					resource.TestCheckResourceAttr("unifi_account.test", "password", "test-password"),
					resource.TestCheckResourceAttr("unifi_account.test", "tunnel_type", "13"),
					resource.TestCheckResourceAttr("unifi_account.test", "tunnel_medium_type", "6"),
				),
			},
			{
				ResourceName:      "unifi_account.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"password"}, // Password is not returned by API
			},
		},
	})
}

func testAccAccountFrameworkConfig_basic() string {
	return `
resource "unifi_account" "test" {
	name     = "test-account"
	password = "test-password"
}
`
}