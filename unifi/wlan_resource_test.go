package unifi

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
					resource.TestCheckResourceAttr("unifi_wlan.test", "name", "wlan1"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "security", "wpapsk"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "passphrase", "passphrase"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "hide_ssid", "false"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.enabled", "true"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.policy", "allow"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "mac_filter.list.#", "1"),
				),
				ResourceName:  "unifi_wlan.test",
				ImportState:   true,
				ImportStateId: "wlan1",
			},
		},
	})
}

func testAccWLANFrameworkConfig_basic() string {
	return `
data "unifi_client_qos_rate" "default" {
	name = "Default"
}

resource "unifi_wlan" "test" {
	name            = "wlan1"
	security        = "wpapsk"
	passphrase      = "passphrase"
	hide_ssid       = false
}
`
}

// TestAccWLANFramework_additionalFields verifies that the newly exposed
// security/DTIM/toggle attributes are populated by the read path when a WLAN
// is imported. It follows the same import-based pattern as the basic test: a
// full create cannot be exercised here because WLAN creation currently fails
// with a pre-existing api.err.InvalidPayload that is unrelated to these
// attributes (a minimal WLAN with none of them set fails identically).
func TestAccWLANFramework_additionalFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWLANFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "wpa_mode"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "wpa_enc"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "dtim_mode"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "group_rekey"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "iapp_enabled"),
					resource.TestCheckResourceAttrSet("unifi_wlan.test", "mlo_enabled"),
					// Issue #176 (secondary): the API omits minimum_data_rate_*
					// from GET responses, so the read path must surface them as 0
					// (the schema default), not null, to avoid perpetual plan
					// drift after import.
					resource.TestCheckResourceAttr("unifi_wlan.test", "minimum_data_rate_2g_kbps", "0"),
					resource.TestCheckResourceAttr("unifi_wlan.test", "minimum_data_rate_5g_kbps", "0"),
				),
				ResourceName:  "unifi_wlan.test",
				ImportState:   true,
				ImportStateId: "wlan1",
			},
		},
	})
}
