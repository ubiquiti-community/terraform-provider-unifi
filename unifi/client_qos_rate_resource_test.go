package unifi

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientQosRate_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"-1",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_up",
						"-1",
					),
				),
			},
			{
				ResourceName:      "unifi_client_qos_rate.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccClientQosRateConfig_basic() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name = "tfacc-group"
}
`
}

func TestAccClientQosRate_qos(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_qos(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-qos-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"1000",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_up",
						"500",
					),
				),
			},
		},
	})
}

func testAccClientQosRateConfig_qos() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-qos-group"
	qos_rate_max_down  = 1000
	qos_rate_max_up    = 500
}
`
}

func TestAccClientQosRate_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientQosRateConfig_update_before(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-update-group",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"100",
					),
				),
			},
			{
				Config: testAccClientQosRateConfig_update_after(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"name",
						"tfacc-update-group-renamed",
					),
					resource.TestCheckResourceAttr(
						"unifi_client_qos_rate.test",
						"qos_rate_max_down",
						"200",
					),
				),
			},
		},
	})
}

func testAccClientQosRateConfig_update_before() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-update-group"
	qos_rate_max_down  = 100
}
`
}

func testAccClientQosRateConfig_update_after() string {
	return `
resource "unifi_client_qos_rate" "test" {
	name               = "tfacc-update-group-renamed"
	qos_rate_max_down  = 200
}
`
}
