package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func ptrInt64(v int64) *int64 { return &v }

// TestFirewallPolicyEndpointSpecificPort is a unit round-trip for the SPECIFIC
// port match (#207): a `port` set on a firewall policy endpoint must reach the
// go-unifi source/destination struct. It guards the fix where the port value
// was previously unrepresentable and silently dropped.
//
// This is a unit test (model -> API conversion) rather than an acceptance test
// because exercising it end-to-end requires zone-based firewall and named
// firewall zones, which the dockerized acceptance controller does not provide.
func TestFirewallPolicyEndpointSpecificPort(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	m := firewallPolicyEndpointModel{
		ZoneID:           types.StringValue("zone-1"),
		MatchingTarget:   types.StringValue("ANY"),
		NetworkIDs:       types.ListNull(types.StringType),
		ClientMACs:       types.ListNull(types.StringType),
		IPs:              types.ListNull(types.StringType),
		Port:             types.Int64Value(443),
		PortGroupID:      types.StringNull(),
		PortMatchingType: types.StringValue("SPECIFIC"),
	}

	src := endpointModelToSource(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("source conversion errored: %v", diags)
	}
	if src.Port == nil || *src.Port != 443 {
		t.Errorf("source Port = %v, want 443", src.Port)
	}
	if src.PortMatchingType != "SPECIFIC" {
		t.Errorf("source PortMatchingType = %q, want SPECIFIC", src.PortMatchingType)
	}

	m.Port = types.Int64Value(8080)
	dst := endpointModelToDestination(ctx, m, &diags)
	if diags.HasError() {
		t.Fatalf("destination conversion errored: %v", diags)
	}
	if dst.Port == nil || *dst.Port != 8080 {
		t.Errorf("destination Port = %v, want 8080", dst.Port)
	}
	if dst.PortMatchingType != "SPECIFIC" {
		t.Errorf("destination PortMatchingType = %q, want SPECIFIC", dst.PortMatchingType)
	}
}

// TestFirewallPolicyPreservesFirmwareFields guards #220: the UCG Max firmware
// rejects a PUT that omits connection_state_type, icmp_typename, icmp_v6_typename
// or the source/destination matching_target_type. These fields are not
// user-settable, so the provider round-trips them through state. This test reads
// an API object into the model and converts it back, asserting nothing is dropped.
func TestFirewallPolicyPreservesFirmwareFields(t *testing.T) {
	ctx := context.Background()

	// A policy as the controller returns it, with all firmware-managed fields set.
	api := &unifi.FirewallPolicy{
		ID:                  "pol-1",
		Name:                "allow-vpn-to-nas-snmp",
		Action:              "ALLOW",
		Enabled:             true,
		Protocol:            "all",
		Version:             "BOTH",
		ConnectionStateType: "ALL",
		ICMPTypename:        "ANY",
		ICMPV6Typename:      "ANY",
		Source: &unifi.FirewallPolicySource{
			ZoneID:             "zone-vpn",
			MatchingTarget:     "IP",
			MatchingTargetType: "OBJECT",
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:             "zone-internal",
			MatchingTarget:     "IP",
			MatchingTargetType: "OBJECT",
			PortMatchingType:   "SPECIFIC",
			Port:               ptrInt64(161),
		},
	}

	// Read API -> model (Read/Create response path).
	var model firewallPolicyModel
	if diags := firewallPolicyToModel(ctx, api, &model); diags.HasError() {
		t.Fatalf("firewallPolicyToModel errored: %v", diags)
	}
	if model.ConnectionStateType.ValueString() != "ALL" {
		t.Errorf("ConnectionStateType = %q, want ALL", model.ConnectionStateType.ValueString())
	}
	if model.ICMPTypename.ValueString() != "ANY" {
		t.Errorf("ICMPTypename = %q, want ANY", model.ICMPTypename.ValueString())
	}
	if model.ICMPV6Typename.ValueString() != "ANY" {
		t.Errorf("ICMPV6Typename = %q, want ANY", model.ICMPV6Typename.ValueString())
	}

	// Convert model -> API (Update PUT path) and assert the fields survive.
	out, diags := modelToFirewallPolicy(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallPolicy errored: %v", diags)
	}
	if out.ConnectionStateType != "ALL" {
		t.Errorf("PUT ConnectionStateType = %q, want ALL", out.ConnectionStateType)
	}
	if out.ICMPTypename != "ANY" {
		t.Errorf("PUT ICMPTypename = %q, want ANY", out.ICMPTypename)
	}
	if out.ICMPV6Typename != "ANY" {
		t.Errorf("PUT ICMPV6Typename = %q, want ANY", out.ICMPV6Typename)
	}
	if out.Source == nil || out.Source.MatchingTargetType != "OBJECT" {
		t.Errorf("PUT source MatchingTargetType not preserved: %+v", out.Source)
	}
	if out.Destination == nil || out.Destination.MatchingTargetType != "OBJECT" {
		t.Errorf("PUT destination MatchingTargetType not preserved: %+v", out.Destination)
	}
	if out.Destination == nil || out.Destination.Port == nil || *out.Destination.Port != 161 {
		t.Errorf("PUT destination Port not preserved: %+v", out.Destination)
	}
}

// TestFirewallPolicyConnectionStatesRoundTrip guards #227: a policy whose
// connection_state_type is CUSTOM must round-trip its connection_states. The
// model->API conversion previously hard-coded an empty slice, so updates sent
// "connection_states": [] and the firmware rejected CUSTOM policies (HTTP 400).
func TestFirewallPolicyConnectionStatesRoundTrip(t *testing.T) {
	ctx := context.Background()
	fp := &unifi.FirewallPolicy{
		ID:                  "p1",
		Name:                "deny-vpn-to-lan",
		Action:              "BLOCK",
		Protocol:            "all",
		ConnectionStateType: "CUSTOM",
		ConnectionStates:    []string{"NEW", "ESTABLISHED"},
		Source: &unifi.FirewallPolicySource{
			ZoneID:           "z1",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
		Destination: &unifi.FirewallPolicyDestination{
			ZoneID:           "z2",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
	}

	var model firewallPolicyModel
	if d := firewallPolicyToModel(ctx, fp, &model); d.HasError() {
		t.Fatalf("firewallPolicyToModel: %v", d)
	}
	var states []string
	if d := model.ConnectionStates.ElementsAs(ctx, &states, false); d.HasError() {
		t.Fatalf("reading connection_states: %v", d)
	}
	if len(states) != 2 || states[0] != "NEW" || states[1] != "ESTABLISHED" {
		t.Errorf("read connection_states = %v, want [NEW ESTABLISHED]", states)
	}

	out, d := modelToFirewallPolicy(ctx, model)
	if d.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", d)
	}
	if len(out.ConnectionStates) != 2 || out.ConnectionStates[0] != "NEW" ||
		out.ConnectionStates[1] != "ESTABLISHED" {
		t.Errorf("PUT dropped connection_states: %v, want [NEW ESTABLISHED]", out.ConnectionStates)
	}
}
