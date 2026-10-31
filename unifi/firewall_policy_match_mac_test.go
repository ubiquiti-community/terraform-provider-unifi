package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestFirewallPolicyMatchMACSchema checks match_mac is user-settable, present
// on both endpoints, and defaulted on create only (no static Default, #544).
func TestFirewallPolicyMatchMACSchema(t *testing.T) {
	r := &firewallPolicyResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	for _, ep := range []string{"source", "destination"} {
		nested, ok := resp.Schema.Attributes[ep].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is not a SingleNestedAttribute", ep)
		}
		a, ok := nested.Attributes["match_mac"].(schema.BoolAttribute)
		if !ok {
			t.Fatalf("%s.match_mac missing or not a BoolAttribute", ep)
		}
		if !a.Optional || !a.Computed || a.Default != nil || len(a.PlanModifiers) == 0 {
			t.Errorf("%s.match_mac must be Optional+Computed with a create-only default, got %#v",
				ep, a)
		}
	}
}

// TestFirewallPolicyMatchMACRoundTrip covers model -> API -> model for
// match_mac on an IP source (the only target the controller stores it on) so a
// true value is neither dropped on the wire nor lost on read.
// go-unifi serializes the field without omitempty, so an unmapped value would
// go out as false and disable MAC matching on every update.
func TestFirewallPolicyMatchMACRoundTrip(t *testing.T) {
	ctx := context.Background()

	endpoint := func(matchMAC bool, mac string) types.Object {
		m := firewallPolicyEndpointModel{
			ZoneID:             types.StringValue("zone"),
			MatchingTarget:     types.StringValue("IP"),
			MatchingTargetType: types.StringValue("SPECIFIC"),
			NetworkIDs:         types.ListNull(types.StringType),
			ClientMACs:         types.ListNull(types.StringType),
			IPs: types.ListValueMust(
				types.StringType,
				[]attr.Value{types.StringValue("192.0.2.10")},
			),
			WebDomains:            types.ListNull(types.StringType),
			Regions:               types.ListNull(types.StringType),
			Port:                  types.StringNull(),
			PortGroupID:           types.StringValue(""),
			IPGroupID:             types.StringValue(""),
			PortMatchingType:      types.StringValue("ANY"),
			MatchOppositeIPs:      types.BoolValue(false),
			MatchOppositeNetworks: types.BoolValue(false),
			MatchOppositePorts:    types.BoolValue(false),
			MatchMAC:              types.BoolValue(matchMAC),
			MAC:                   types.StringValue(mac),
		}
		obj, d := types.ObjectValueFrom(ctx, firewallPolicyEndpointModel{}.AttributeTypes(), m)
		if d.HasError() {
			t.Fatalf("building endpoint: %v", d)
		}
		return obj
	}

	model := firewallPolicyModel{
		Name:                  types.StringValue("mac-match"),
		Action:                types.StringValue("ALLOW"),
		Enabled:               types.BoolValue(true),
		Protocol:              types.StringValue("all"),
		MatchOppositeProtocol: types.BoolValue(false),
		IPVersion:             types.StringValue("BOTH"),
		ConnectionStates:      types.ListNull(types.StringType),
		Schedule:              types.ObjectNull(firewallPolicyScheduleModel{}.AttributeTypes()),
		Source:                endpoint(true, "aa:bb:cc:dd:ee:ff"),
		Destination:           endpoint(false, ""),
	}

	fp, d := modelToFirewallPolicy(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", d)
	}
	if !fp.Source.MatchMAC {
		t.Errorf("source match_mac not sent")
	}
	if fp.Source.MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("source mac = %q, want aa:bb:cc:dd:ee:ff", fp.Source.MAC)
	}
	if fp.Destination.MatchMAC {
		t.Errorf("destination match_mac sent as true, want false")
	}

	// Simulate the controller echoing the policy back.
	var back firewallPolicyModel
	if d := firewallPolicyToModel(ctx, fp, &back); d.HasError() {
		t.Fatalf("firewallPolicyToModel: %v", d)
	}
	var src, dst firewallPolicyEndpointModel
	var diags diag.Diagnostics
	diags.Append(back.Source.As(ctx, &src, basetypes.ObjectAsOptions{})...)
	diags.Append(back.Destination.As(ctx, &dst, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		t.Fatalf("decoding endpoints: %v", diags)
	}
	if !src.MatchMAC.ValueBool() {
		t.Errorf("source match_mac lost on read")
	}
	if src.MAC.ValueString() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("source mac lost on read: %q", src.MAC.ValueString())
	}
	if dst.MAC.IsNull() || dst.MAC.ValueString() != "" {
		t.Errorf("destination mac = %v, want \"\"", dst.MAC)
	}
	if dst.MatchMAC.ValueBool() {
		t.Errorf("destination match_mac read as true, want false")
	}

	// A controller response with the flag set must surface it even when the
	// prior model had none (fresh import).
	imported := firewallPolicyModel{}
	if d := firewallPolicyToModel(ctx, &unifi.FirewallPolicy{
		Source:      &unifi.FirewallPolicySource{MatchMAC: true, MAC: "aa:bb:cc:dd:ee:ff"},
		Destination: &unifi.FirewallPolicyDestination{MatchMAC: true},
	}, &imported); d.HasError() {
		t.Fatalf("firewallPolicyToModel(import): %v", d)
	}
	var isrc, idst firewallPolicyEndpointModel
	diags = nil
	diags.Append(imported.Source.As(ctx, &isrc, basetypes.ObjectAsOptions{})...)
	diags.Append(imported.Destination.As(ctx, &idst, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		t.Fatalf("decoding imported endpoints: %v", diags)
	}
	if !isrc.MatchMAC.ValueBool() || !idst.MatchMAC.ValueBool() {
		t.Errorf("imported match_mac not surfaced: src=%+v dst=%+v", isrc, idst)
	}
	if isrc.MAC.ValueString() != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("imported source mac not surfaced: %q", isrc.MAC.ValueString())
	}
}

// TestFirewallPolicyMatchMACValidateConfig covers #575: the controller only
// stores match_mac on a source IP target. On a CLIENT source (and on the
// destination) it drops the flag, so match_mac = true must fail at plan time
// instead of producing an inconsistent result after apply.
func TestFirewallPolicyMatchMACValidateConfig(t *testing.T) {
	r := &firewallPolicyResource{}
	s := resourceSchema(t, r)

	endpoint := func(target, matchMAC any) map[string]any {
		return map[string]any{
			"zone_id":         "zone",
			"matching_target": target,
			"match_mac":       matchMAC,
		}
	}
	withMAC := func(ep map[string]any, mac any) map[string]any {
		ep["mac"] = mac
		return ep
	}
	const mac = "aa:bb:cc:dd:ee:ff"

	cases := []struct {
		name      string
		source    map[string]any
		dest      map[string]any
		wantPaths []string
	}{
		{
			"client source match_mac true",
			endpoint("CLIENT", true),
			endpoint("ANY", nil),
			[]string{"source.match_mac"},
		},
		{"client source match_mac false", endpoint("CLIENT", false), endpoint("ANY", nil), nil},
		{"client source match_mac unset", endpoint("CLIENT", nil), endpoint("ANY", nil), nil},
		{
			"client source match_mac unknown",
			endpoint("CLIENT", tfUnknown),
			endpoint("ANY", nil),
			nil,
		},
		{
			"any source match_mac true",
			endpoint("ANY", true),
			endpoint("ANY", nil),
			[]string{"source.match_mac"},
		},
		{
			"ip source match_mac true without mac",
			endpoint("IP", true),
			endpoint("ANY", nil),
			[]string{"source.mac"},
		},
		{
			"ip source match_mac true empty mac",
			withMAC(endpoint("IP", true), ""),
			endpoint("ANY", nil),
			[]string{"source.mac"},
		},
		{
			"ip source match_mac true with mac",
			withMAC(endpoint("IP", true), mac),
			endpoint("ANY", nil),
			nil,
		},
		{
			"ip source unknown mac",
			withMAC(endpoint("IP", true), tfUnknown),
			endpoint("ANY", nil),
			nil,
		},
		{
			"ip source invalid mac",
			withMAC(endpoint("IP", true), "not-a-mac"),
			endpoint("ANY", nil),
			[]string{"source.mac"},
		},
		{
			"ip source eui64 mac",
			withMAC(endpoint("IP", true), "aa:bb:cc:dd:ee:ff:00:11"),
			endpoint("ANY", nil),
			[]string{"source.mac"},
		},
		{
			"mac without match_mac",
			withMAC(endpoint("IP", nil), mac),
			endpoint("ANY", nil),
			[]string{"source.mac"},
		},
		{
			"mac on client source",
			withMAC(endpoint("CLIENT", true), mac),
			endpoint("ANY", nil),
			[]string{"source.match_mac", "source.mac"},
		},
		{
			"mac on destination",
			withMAC(endpoint("IP", true), mac),
			withMAC(endpoint("IP", nil), mac),
			[]string{"destination.mac"},
		},
		{"unknown target match_mac true", endpoint(tfUnknown, true), endpoint("ANY", nil), nil},
		{
			"destination match_mac true",
			endpoint("IP", nil),
			endpoint("IP", true),
			[]string{"destination.match_mac"},
		},
		{
			"both invalid",
			endpoint("CLIENT", true),
			endpoint("ANY", true),
			[]string{"source.match_mac", "destination.match_mac"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := resource.ValidateConfigRequest{
				Config: tfConfig(t, s, map[string]any{
					"name":        "p",
					"action":      "ALLOW",
					"source":      tc.source,
					"destination": tc.dest,
				}),
			}
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(context.Background(), req, &resp)

			var got []string
			for _, d := range resp.Diagnostics.Errors() {
				if wp, ok := d.(diag.DiagnosticWithPath); ok {
					got = append(got, wp.Path().String())
				} else {
					t.Fatalf("unexpected diagnostic without path: %v", d)
				}
			}
			if len(got) != len(tc.wantPaths) {
				t.Fatalf("errors at %v, want %v", got, tc.wantPaths)
			}
			for i := range got {
				if got[i] != tc.wantPaths[i] {
					t.Errorf("error %d at %s, want %s", i, got[i], tc.wantPaths[i])
				}
			}
		})
	}
}
