package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

// TestUnitStaticRoute_nextHopValidation verifies the next_hop validator accepts IPv4 and IPv6
// and rejects non-IP values, without requiring a real UniFi controller.
func TestUnitStaticRoute_nextHopValidation(t *testing.T) {
	// Replicate the validator used in the schema.
	v := stringvalidator.Any(validators.IPv4Validator(), validators.IPv6Validator())

	tests := []struct {
		name      string
		nextHop   string
		wantError bool
	}{
		{name: "valid_ipv4", nextHop: "192.168.1.1", wantError: false},
		{name: "valid_ipv6_full", nextHop: "2001:0db8:0000:0000:0000:0000:0000:0001", wantError: false},
		{name: "valid_ipv6_compressed", nextHop: "2001:db8::1", wantError: false},
		{name: "valid_ipv6_loopback", nextHop: "::1", wantError: false},
		{name: "invalid_hostname", nextHop: "not-an-ip", wantError: true},
		{name: "invalid_cidr", nextHop: "192.168.1.0/24", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:           path.Root("next_hop"),
				PathExpression: path.MatchRoot("next_hop"),
				ConfigValue:    types.StringValue(tc.nextHop),
				Config:         tfsdk.Config{}, // unused by these validators
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			hasError := resp.Diagnostics.HasError()
			if hasError != tc.wantError {
				t.Errorf("next_hop=%q: got error=%v, want error=%v (diags: %v)",
					tc.nextHop, hasError, tc.wantError, resp.Diagnostics)
			}
		})
	}
}

// TestUnitStaticRoute_ipVersionValidator verifies that mixed IPv4/IPv6 network+next_hop is rejected.
func TestUnitStaticRoute_ipVersionValidator(t *testing.T) {
	tests := []struct {
		name      string
		network   string
		nextHop   string
		wantError bool
	}{
		{name: "ipv4_both", network: "192.168.100.0/24", nextHop: "192.168.1.1", wantError: false},
		{name: "ipv6_both", network: "2001:db8::/32", nextHop: "2001:db8::1", wantError: false},
		{name: "ipv4_network_ipv6_hop", network: "192.168.100.0/24", nextHop: "2001:db8::1", wantError: true},
		{name: "ipv6_network_ipv4_hop", network: "2001:db8::/32", nextHop: "192.168.1.1", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIPVersionMatch(tc.network, tc.nextHop)
			if (err != nil) != tc.wantError {
				t.Errorf("network=%q next_hop=%q: got err=%v, wantError=%v",
					tc.network, tc.nextHop, err, tc.wantError)
			}
		})
	}
}

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

func TestAccStaticRouteFramework_ipv6NextHop(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStaticRouteFrameworkConfig_ipv6(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_static_route.test", "name", "test-route-ipv6"),
					resource.TestCheckResourceAttr("unifi_static_route.test", "next_hop", "2001:db8::1"),
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

func TestAccStaticRouteFramework_enabledAndGateway(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStaticRouteFrameworkConfig_disabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_static_route.disabled", "enabled", "false"),
					// gateway_type defaults to the controller value.
					resource.TestCheckResourceAttr(
						"unifi_static_route.disabled",
						"gateway_type",
						"default",
					),
				),
			},
			{
				ResourceName:      "unifi_static_route.disabled",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccStaticRouteFrameworkConfig_disabled() string {
	return `
resource "unifi_static_route" "disabled" {
	name     = "test-route-disabled"
	network  = "192.168.101.0/24"
	type     = "nexthop-route"
	distance = 1
	next_hop = "192.168.1.1"
	enabled  = false
}
`
}

func testAccStaticRouteFrameworkConfig_ipv6() string {
	return `
resource "unifi_static_route" "test" {
	name     = "test-route-ipv6"
	network  = "2001:db8::/32"
	type     = "nexthop-route"
	distance = 1
	next_hop = "2001:db8::1"
}
`
}
