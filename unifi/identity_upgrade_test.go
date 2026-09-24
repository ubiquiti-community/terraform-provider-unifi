package unifi

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// testAccIdentityUpgradeSteps creates the resource with a released provider,
// then plans and applies the same config with the provider under test. The
// second step fails if the stored identity can no longer be decoded or if Read
// returns an identity that differs from the upgraded one (#502).
func testAccIdentityUpgradeSteps(
	fromVersion string,
	config string,
	stateChecks ...statecheck.StateCheck,
) []resource.TestStep {
	return []resource.TestStep{
		{
			ExternalProviders: map[string]resource.ExternalProvider{
				"unifi": {
					Source:            "ubiquiti-community/unifi",
					VersionConstraint: fromVersion,
				},
			},
			Config: config,
		},
		{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Config:                   config,
			ConfigStateChecks:        stateChecks,
		},
	}
}

func TestAccDevice_identityUpgradeFromV055(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheck(t) },
		Steps: testAccIdentityUpgradeSteps(
			"0.55.0",
			testAccDeviceFrameworkConfig_basic(),
			statecheck.ExpectIdentityValue(
				"unifi_device.test",
				tfjsonpath.New("mac"),
				knownvalue.StringExact("00:27:22:00:00:02"),
			),
		),
	})
}

// v0.56.0 stored the MAC-keyed identity at version 0.
func TestAccDevice_identityUpgradeFromV056(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheck(t) },
		Steps: testAccIdentityUpgradeSteps(
			"0.56.0",
			testAccDeviceFrameworkConfig_basic(),
			statecheck.ExpectIdentityValue(
				"unifi_device.test",
				tfjsonpath.New("mac"),
				knownvalue.StringExact("00:27:22:00:00:02"),
			),
		),
	})
}

// Resources that only gained an optional site attribute keep decoding
// v0.55.0 identities (site is null) and pass them through on refresh.
func TestAccFirewallGroup_identityUpgradeFromV055(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheck(t) },
		Steps: testAccIdentityUpgradeSteps(
			"0.55.0",
			fmt.Sprintf(`
resource "unifi_firewall_group" "test" {
	name    = "tfacc-upgrade-%s"
	type    = "address-group"
	members = ["192.168.1.10"]
}
`, acctest.RandString(8)),
			statecheck.ExpectIdentityValue(
				"unifi_firewall_group.test",
				tfjsonpath.New("id"),
				knownvalue.NotNull(),
			),
		),
	})
}

func TestAccRadiusProfile_identityUpgradeFromV055(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheck(t) },
		Steps: testAccIdentityUpgradeSteps(
			"0.55.0",
			fmt.Sprintf(`
resource "unifi_radius_profile" "test" {
	name = "tfacc-upgrade-%s"
}
`, acctest.RandString(8)),
			statecheck.ExpectIdentityValue(
				"unifi_radius_profile.test",
				tfjsonpath.New("id"),
				knownvalue.NotNull(),
			),
		),
	})
}

func TestAccDNSRecord_identityUpgradeFromV055(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() { preCheck(t) },
		Steps: testAccIdentityUpgradeSteps(
			"0.55.0",
			fmt.Sprintf(`
resource "unifi_dns_record" "test" {
	name        = "tfacc-upgrade-%s.example.com"
	enabled     = true
	record_type = "A"
	ttl         = "5m0s"
	value       = "192.168.1.100"
}
`, acctest.RandString(8)),
			statecheck.ExpectIdentityValue(
				"unifi_dns_record.test",
				tfjsonpath.New("id"),
				knownvalue.NotNull(),
			),
		),
	})
}
