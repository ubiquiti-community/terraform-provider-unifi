package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWLANFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test", "name", "tfacc-test"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "security", "wpapsk"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "passphrase", "test1234"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "hide_ssid", "false"),
				),
			},
			{
				ResourceName:      "unifi_wlan.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"passphrase"},
			},
		},
	})
}

func testAccWLANFrameworkConfig_basic() string {
	return `
data "unifi_user_group" "default" {
	name = "Default"
}

resource "unifi_wlan" "test" {
	name            = "tfacc-test"
	security        = "wpapsk"
	passphrase      = "test1234"
	user_group_id   = data.unifi_user_group.default.id
	hide_ssid       = false
}
`
}

func TestAccWLANFramework_wpa3(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_wpa3(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test", "name", "tfacc-wpa3"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "security", "wpapsk"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "wpa3_support", "true"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "pmf_mode", "required"),
				),
			},
		},
	})
}

func testAccWLANFrameworkConfig_wpa3() string {
	return `
data "unifi_user_group" "default" {
	name = "Default"
}

resource "unifi_wlan" "test" {
	name            = "tfacc-wpa3"
	security        = "wpapsk"
	passphrase      = "test1234"
	user_group_id   = data.unifi_user_group.default.id
	wpa3_support    = true
	pmf_mode        = "required"
}
`
}

func TestAccWLANFramework_macFilter(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_macFilter(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_wlan.test", "name", "tfacc-mac-filter"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter_enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter_policy", "allow"),
					resource.TestCheckTypeSetElemAttr("unifi_wlan.test", "mac_filter_list.*", "ab:cd:ef:12:34:56"),
				),
			},
		},
	})
}

func testAccWLANFrameworkConfig_macFilter() string {
	return `
data "unifi_user_group" "default" {
	name = "Default"
}

resource "unifi_wlan" "test" {
	name                = "tfacc-mac-filter"
	security            = "wpapsk"
	passphrase          = "test1234"
	user_group_id       = data.unifi_user_group.default.id
	mac_filter_enabled  = true
	mac_filter_policy   = "allow"
	mac_filter_list     = ["ab:cd:ef:12:34:56"]
}
`
}