package unifi

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/iptypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/validators"
)

func testAccStaticRouteCheckDestroy(s *terraform.State) error {
	ctx := context.Background()
	apiURL := os.Getenv("UNIFI_API")
	if apiURL == "" {
		return nil
	}
	apiClient, err := unifi.New(ctx, &unifi.Config{
		BaseURL:       apiURL,
		Username:      os.Getenv("UNIFI_USERNAME"),
		Password:      os.Getenv("UNIFI_PASSWORD"),
		AllowInsecure: true,
	})
	if err != nil {
		return nil //nolint:nilerr // best-effort check; skip when no live client
	}
	c := &Client{ApiClient: apiClient, Site: "default"}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "unifi_static_route" {
			continue
		}
		site := rs.Primary.Attributes["site"]
		if site == "" {
			site = c.Site
		}
		_, err := c.GetRouting(ctx, site, rs.Primary.ID)
		if err == nil {
			return fmt.Errorf("unifi_static_route %s still exists", rs.Primary.ID)
		}
		if _, ok := err.(*unifi.NotFoundError); !ok {
			return err
		}
	}
	return nil
}

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
		{
			name:      "valid_ipv6_full",
			nextHop:   "2001:0db8:0000:0000:0000:0000:0000:0001",
			wantError: false,
		},
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
		{
			name:      "ipv4_network_ipv6_hop",
			network:   "192.168.100.0/24",
			nextHop:   "2001:db8::1",
			wantError: true,
		},
		{
			name:      "ipv6_network_ipv4_hop",
			network:   "2001:db8::/32",
			nextHop:   "192.168.1.1",
			wantError: true,
		},
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
		CheckDestroy:             testAccStaticRouteCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccStaticRouteFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_static_route.test", "name", "test-route"),
					resource.TestCheckResourceAttr(
						"unifi_static_route.test",
						"network",
						"192.168.100.0/24",
					),
					resource.TestCheckResourceAttr(
						"unifi_static_route.test",
						"type",
						"nexthop-route",
					),
					resource.TestCheckResourceAttr("unifi_static_route.test", "distance", "1"),
					resource.TestCheckResourceAttr(
						"unifi_static_route.test",
						"next_hop",
						"192.168.1.1",
					),
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
					resource.TestCheckResourceAttr(
						"unifi_static_route.test",
						"name",
						"test-route-ipv6",
					),
					resource.TestCheckResourceAttr(
						"unifi_static_route.test",
						"next_hop",
						"2001:db8::1",
					),
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
					resource.TestCheckResourceAttr(
						"unifi_static_route.disabled",
						"enabled",
						"false",
					),
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

func TestNewStaticRouteFrameworkResource(t *testing.T) {
	r := NewStaticRouteFrameworkResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity")
	}
	if _, ok := r.(fwresource.ResourceWithConfigValidators); !ok {
		t.Error("expected ResourceWithConfigValidators")
	}
}

func TestNewStaticRouteListResource(t *testing.T) {
	r := NewStaticRouteListResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure")
	}
}

func Test_staticRouteFrameworkResource_Metadata(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "unifi"}, resp)
	if resp.TypeName != "unifi_static_route" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_static_route")
	}
}

func Test_staticRouteFrameworkResource_IdentitySchema(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("expected identity schema to have 'id' attribute")
	}
}

func Test_staticRouteFrameworkResource_Schema(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	for _, attr := range []string{"id", "site", "name", "network", "type", "distance", "next_hop", "interface", "enabled", "gateway_device", "gateway_type"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected attribute %q in schema", attr)
		}
	}
}

func Test_staticRouteFrameworkResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		req       fwresource.ConfigureRequest
		wantError bool
	}{
		{"nil_provider_data", fwresource.ConfigureRequest{}, false},
		{"wrong_type", fwresource.ConfigureRequest{ProviderData: "wrong"}, true},
		{"correct_client", fwresource.ConfigureRequest{ProviderData: &Client{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &staticRouteFrameworkResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), tt.req, resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

func Test_staticRouteFrameworkResource_ConfigValidators(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	validators := r.ConfigValidators(context.Background())
	if len(validators) == 0 {
		t.Error("expected at least one config validator")
	}
}

func Test_staticRouteIPVersionValidator_Description(t *testing.T) {
	v := &staticRouteIPVersionValidator{}
	want := "network and next_hop must use the same IP version (both IPv4 or both IPv6)"
	if got := v.Description(context.Background()); got != want {
		t.Errorf("Description() = %q, want %q", got, want)
	}
}

func Test_staticRouteIPVersionValidator_MarkdownDescription(t *testing.T) {
	v := &staticRouteIPVersionValidator{}
	want := "network and next_hop must use the same IP version (both IPv4 or both IPv6)"
	if got := v.MarkdownDescription(context.Background()); got != want {
		t.Errorf("MarkdownDescription() = %q, want %q", got, want)
	}
}

func Test_ipVersionsMatch(t *testing.T) {
	tests := []struct {
		name string
		args struct {
			prefix netip.Prefix
			hop    netip.Addr
		}
		want bool
	}{
		{
			name: "both_ipv4",
			args: struct {
				prefix netip.Prefix
				hop    netip.Addr
			}{netip.MustParsePrefix("192.168.0.0/24"), netip.MustParseAddr("10.0.0.1")},
			want: true,
		},
		{
			name: "both_ipv6",
			args: struct {
				prefix netip.Prefix
				hop    netip.Addr
			}{netip.MustParsePrefix("2001:db8::/32"), netip.MustParseAddr("2001:db8::1")},
			want: true,
		},
		{
			name: "ipv4_prefix_ipv6_hop",
			args: struct {
				prefix netip.Prefix
				hop    netip.Addr
			}{netip.MustParsePrefix("192.168.0.0/24"), netip.MustParseAddr("2001:db8::1")},
			want: false,
		},
		{
			name: "ipv6_prefix_ipv4_hop",
			args: struct {
				prefix netip.Prefix
				hop    netip.Addr
			}{netip.MustParsePrefix("2001:db8::/32"), netip.MustParseAddr("10.0.0.1")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ipVersionsMatch(tt.args.prefix, tt.args.hop); got != tt.want {
				t.Errorf("ipVersionsMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateIPVersionMatch(t *testing.T) {
	tests := []struct {
		name    string
		network string
		nextHop string
		wantErr bool
	}{
		{"both_ipv4", "192.168.0.0/24", "10.0.0.1", false},
		{"both_ipv6", "2001:db8::/32", "2001:db8::1", false},
		{"mixed_v4_v6", "192.168.0.0/24", "2001:db8::1", true},
		{"mixed_v6_v4", "2001:db8::/32", "10.0.0.1", true},
		{"invalid_network", "not-cidr", "10.0.0.1", false},
		{"invalid_hop", "192.168.0.0/24", "not-ip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateIPVersionMatch(tt.network, tt.nextHop); (err != nil) != tt.wantErr {
				t.Errorf("validateIPVersionMatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_staticRouteFrameworkResource_applyPlanToState(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	plan := &staticRouteFrameworkResourceModel{
		Name:    types.StringValue("route1"),
		Network: types.StringValue("10.0.0.0/8"),
		Type:    types.StringValue("nexthop-route"),
	}
	state := &staticRouteFrameworkResourceModel{}
	r.applyPlanToState(context.Background(), plan, state)
	if state.Name.ValueString() != "route1" {
		t.Error("expected Name to be copied from plan")
	}
	if state.Network.ValueString() != "10.0.0.0/8" {
		t.Error("expected Network to be copied from plan")
	}
}

func Test_staticRouteFrameworkResource_modelToRouting(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	dist := int64(1)
	model := &staticRouteFrameworkResourceModel{
		Name:          types.StringValue("route1"),
		Network:       types.StringValue("192.168.0.0/24"),
		Type:          types.StringValue("nexthop-route"),
		Distance:      types.Int64Value(1),
		NextHop:       iptypes.NewIPAddressValue("192.168.1.1"),
		Interface:     types.StringNull(),
		Enabled:       types.BoolValue(true),
		GatewayDevice: types.StringNull(),
		GatewayType:   types.StringValue("default"),
	}
	got := r.modelToRouting(context.Background(), model)
	want := &unifi.Routing{
		Type:                "static-route",
		Name:                "route1",
		StaticRouteNetwork:  "192.168.0.0/24",
		StaticRouteType:     "nexthop-route",
		StaticRouteDistance: &dist,
		StaticRouteNexthop:  "192.168.1.1",
		Enabled:             true,
		GatewayType:         "default",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("modelToRouting() = %+v, want %+v", got, want)
	}
}

func Test_staticRouteFrameworkResource_routingToModel(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	dist := int64(1)
	routing := &unifi.Routing{
		ID:                  "abc123",
		Name:                "route1",
		StaticRouteNetwork:  "192.168.0.0/24",
		StaticRouteType:     "nexthop-route",
		StaticRouteDistance: &dist,
		StaticRouteNexthop:  "192.168.1.1",
		Enabled:             true,
		GatewayType:         "default",
	}
	model := &staticRouteFrameworkResourceModel{}
	r.routingToModel(context.Background(), routing, model, "default")
	if model.ID.ValueString() != "abc123" {
		t.Errorf("ID = %q, want %q", model.ID.ValueString(), "abc123")
	}
	if model.Site.ValueString() != "default" {
		t.Errorf("Site = %q, want %q", model.Site.ValueString(), "default")
	}
	if model.NextHop.ValueString() != "192.168.1.1" {
		t.Errorf("NextHop = %q, want %q", model.NextHop.ValueString(), "192.168.1.1")
	}
}

func Test_staticRouteFrameworkResource_ListResourceConfigSchema(t *testing.T) {
	r := &staticRouteFrameworkResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if len(resp.Schema.Attributes) == 0 {
		t.Error("expected non-empty list resource schema")
	}
}
