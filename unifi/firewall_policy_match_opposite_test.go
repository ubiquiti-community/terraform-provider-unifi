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

// TestFirewallPolicyMatchOppositeSchema checks the four "Match Opposite"
// toggles are user-settable, defaulted to false and present on both endpoints.
func TestFirewallPolicyMatchOppositeSchema(t *testing.T) {
	r := &firewallPolicyResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
	}

	top, ok := resp.Schema.Attributes["match_opposite_protocol"].(schema.BoolAttribute)
	if !ok || !top.Optional || !top.Computed || top.Default == nil {
		t.Fatalf("match_opposite_protocol must be Optional+Computed with a default, got %#v", top)
	}

	for _, ep := range []string{"source", "destination"} {
		nested, ok := resp.Schema.Attributes[ep].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is not a SingleNestedAttribute", ep)
		}
		for _, name := range []string{"match_opposite_ips", "match_opposite_networks", "match_opposite_ports"} {
			a, ok := nested.Attributes[name].(schema.BoolAttribute)
			if !ok {
				t.Fatalf("%s.%s missing or not a BoolAttribute", ep, name)
			}
			if !a.Optional || !a.Computed || a.Default == nil {
				t.Errorf("%s.%s must be Optional+Computed with a default, got %#v", ep, name, a)
			}
		}
	}
}

// TestFirewallPolicyMatchOppositeRoundTrip covers model -> API -> model for
// every match_opposite_* flag so a true value is neither dropped on the wire
// nor lost on read.
func TestFirewallPolicyMatchOppositeRoundTrip(t *testing.T) {
	ctx := context.Background()

	endpoint := func(nets, ips, ports bool) types.Object {
		m := firewallPolicyEndpointModel{
			ZoneID:                types.StringValue("zone"),
			MatchingTarget:        types.StringValue("NETWORK"),
			MatchingTargetType:    types.StringValue("OBJECT"),
			NetworkIDs:            types.ListValueMust(types.StringType, []attr.Value{types.StringValue("net1")}),
			ClientMACs:            types.ListNull(types.StringType),
			IPs:                   types.ListNull(types.StringType),
			WebDomains:            types.ListNull(types.StringType),
			Port:                  types.StringNull(),
			PortGroupID:           types.StringValue(""),
			IPGroupID:             types.StringValue(""),
			PortMatchingType:      types.StringValue("ANY"),
			MatchOppositeIPs:      types.BoolValue(ips),
			MatchOppositeNetworks: types.BoolValue(nets),
			MatchOppositePorts:    types.BoolValue(ports),
		}
		obj, d := types.ObjectValueFrom(ctx, firewallPolicyEndpointModel{}.AttributeTypes(), m)
		if d.HasError() {
			t.Fatalf("building endpoint: %v", d)
		}
		return obj
	}

	model := firewallPolicyModel{
		Name:                  types.StringValue("invert"),
		Action:                types.StringValue("ALLOW"),
		Enabled:               types.BoolValue(true),
		Protocol:              types.StringValue("tcp"),
		MatchOppositeProtocol: types.BoolValue(true),
		IPVersion:             types.StringValue("BOTH"),
		ConnectionStates:      types.ListNull(types.StringType),
		Schedule:              types.ObjectNull(firewallPolicyScheduleModel{}.AttributeTypes()),
		Source:                endpoint(true, false, true),
		Destination:           endpoint(false, true, false),
	}

	fp, d := modelToFirewallPolicy(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", d)
	}
	if !fp.MatchOppositeProtocol {
		t.Errorf("MatchOppositeProtocol not sent")
	}
	if !fp.Source.MatchOppositeNetworks || fp.Source.MatchOppositeIPs || !fp.Source.MatchOppositePorts {
		t.Errorf("source flags wrong: %+v", fp.Source)
	}
	if fp.Destination.MatchOppositeNetworks || !fp.Destination.MatchOppositeIPs || fp.Destination.MatchOppositePorts {
		t.Errorf("destination flags wrong: %+v", fp.Destination)
	}

	// Simulate the controller echoing the policy back.
	var back firewallPolicyModel
	if d := firewallPolicyToModel(ctx, fp, &back); d.HasError() {
		t.Fatalf("firewallPolicyToModel: %v", d)
	}
	if !back.MatchOppositeProtocol.ValueBool() {
		t.Errorf("match_opposite_protocol lost on read")
	}
	var src, dst firewallPolicyEndpointModel
	var diags diag.Diagnostics
	diags.Append(back.Source.As(ctx, &src, basetypes.ObjectAsOptions{})...)
	diags.Append(back.Destination.As(ctx, &dst, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		t.Fatalf("decoding endpoints: %v", diags)
	}
	if !src.MatchOppositeNetworks.ValueBool() || src.MatchOppositeIPs.ValueBool() || !src.MatchOppositePorts.ValueBool() {
		t.Errorf("source flags lost on read: %+v", src)
	}
	if dst.MatchOppositeNetworks.ValueBool() || !dst.MatchOppositeIPs.ValueBool() || dst.MatchOppositePorts.ValueBool() {
		t.Errorf("destination flags lost on read: %+v", dst)
	}

	// A controller response with the flags set must surface them even when the
	// prior model had none (fresh import).
	imported := firewallPolicyModel{}
	if d := firewallPolicyToModel(ctx, &unifi.FirewallPolicy{
		MatchOppositeProtocol: true,
		Source:                &unifi.FirewallPolicySource{MatchOppositeNetworks: true},
		Destination:           &unifi.FirewallPolicyDestination{MatchOppositeIPs: true, MatchOppositePorts: true},
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
	if !imported.MatchOppositeProtocol.ValueBool() || !isrc.MatchOppositeNetworks.ValueBool() ||
		!idst.MatchOppositeIPs.ValueBool() || !idst.MatchOppositePorts.ValueBool() {
		t.Errorf("imported flags not surfaced: proto=%v src=%+v dst=%+v",
			imported.MatchOppositeProtocol, isrc, idst)
	}
}
