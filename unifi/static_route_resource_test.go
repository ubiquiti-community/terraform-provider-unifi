package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStaticRouteFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		CheckDestroy:             nil, // TODO: implement check destroy
		Steps: []resource.TestStep{
			{
				Config: testAccStaticRouteFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_static_route.test", "name", "test-route"),
					resource.TestCheckResourceAttr("unifi_static_route.test", "network", "192.168.100.0/24"),
					resource.TestCheckResourceAttr("unifi_static_route.test", "type", "nexthop-route"),
					resource.TestCheckResourceAttr("unifi_static_route.test", "distance", "1"),
					resource.TestCheckResourceAttr("unifi_static_route.test", "next_hop", "192.168.1.1"),
				),
			},
			{
				ResourceName:      "unifi_static_route.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccStaticRouteFrameworkConfig_basic() string {
	return `
resource "unifi_static_route" "test" {
	name     = "test-route"
	network  = "192.168.100.0/24"
	type     = "nexthop-route"
	distance = 1
	next_hop = "192.168.1.1"
}
`
}