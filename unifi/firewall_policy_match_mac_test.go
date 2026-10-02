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

// TestFirewallPolicyMatchMACSchema checks match_mac is user-settable, defaulted
// to false and present on both endpoints.
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
		if !a.Optional || !a.Computed || a.Default == nil {
			t.Errorf("%s.match_mac must be Optional+Computed with a default, got %#v", ep, a)
		}
	}
}

// TestFirewallPolicyMatchMACRoundTrip covers model -> API -> model for
// match_mac so a true value is neither dropped on the wire nor lost on read.
// go-unifi serializes the field without omitempty, so an unmapped value would
// go out as false and disable MAC matching on every update.
func TestFirewallPolicyMatchMACRoundTrip(t *testing.T) {
	ctx := context.Background()

	endpoint := func(matchMAC bool) types.Object {
		m := firewallPolicyEndpointModel{
			ZoneID:             types.StringValue("zone"),
			MatchingTarget:     types.StringValue("CLIENT"),
			MatchingTargetType: types.StringValue("SPECIFIC"),
			NetworkIDs:         types.ListNull(types.StringType),
			ClientMACs: types.ListValueMust(
				types.StringType,
				[]attr.Value{types.StringValue("00:11:22:33:44:55")},
			),
			IPs:                   types.ListNull(types.StringType),
			WebDomains:            types.ListNull(types.StringType),
			Port:                  types.StringNull(),
			PortGroupID:           types.StringValue(""),
			IPGroupID:             types.StringValue(""),
			PortMatchingType:      types.StringValue("ANY"),
			MatchOppositeIPs:      types.BoolValue(false),
			MatchOppositeNetworks: types.BoolValue(false),
			MatchOppositePorts:    types.BoolValue(false),
			MatchMAC:              types.BoolValue(matchMAC),
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
		Source:                endpoint(true),
		Destination:           endpoint(false),
	}

	fp, d := modelToFirewallPolicy(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", d)
	}
	if !fp.Source.MatchMAC {
		t.Errorf("source match_mac not sent")
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
	if dst.MatchMAC.ValueBool() {
		t.Errorf("destination match_mac read as true, want false")
	}

	// A controller response with the flag set must surface it even when the
	// prior model had none (fresh import).
	imported := firewallPolicyModel{}
	if d := firewallPolicyToModel(ctx, &unifi.FirewallPolicy{
		Source:      &unifi.FirewallPolicySource{MatchMAC: true},
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
}
