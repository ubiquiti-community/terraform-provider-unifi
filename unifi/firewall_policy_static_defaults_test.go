package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// #544: an Optional+Computed attribute that also carries a static Default is
// overwritten by that Default, because the framework applies defaults
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan). On an
// adopted policy whose configuration leaves the attribute out, the plan
// proposes the constant instead of the controller's value.
//
// Measured on a live site with 377 policies, the provider's defaults
// disagreed with the controller on:
//
//	create_allow_respond  default false   true on 333 of 377
//	protocol              default "all"   something else on 38
//	logging               default false   true on 27
//	port_matching_type    default "ANY"   SPECIFIC/OBJECT on 12 sources, 30 destinations
//	ip_group_id / port_group_id           a group on 1 source, 1 destination
//
// The controller assigns protocol ("all") and port_matching_type ("ANY")
// itself when a create omits them (probed with a disabled throwaway policy),
// so those defaults are simply dropped. `enabled` is the exception: go-unifi
// always sends it, so a create must pick a value, and it stays `true` on
// create only (see firewallPolicyCreateDefaults).
var firewallPolicyHeldAttributes = []string{
	"enabled",
	"protocol",
	"description",
	"logging",
	"match_opposite_protocol",
	"create_allow_respond",
}

var firewallPolicyHeldEndpointAttributes = []string{
	"port_group_id",
	"ip_group_id",
	"port_matching_type",
	"match_opposite_ips",
	"match_opposite_networks",
	"match_opposite_ports",
}

func TestFirewallPolicyNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&firewallPolicyResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	check := func(name string, a schema.Attribute) {
		t.Helper()
		switch a := a.(type) {
		case schema.StringAttribute:
			if a.Default != nil {
				t.Errorf("%s must not carry a static Default (#544)", name)
			}
			if len(a.PlanModifiers) == 0 {
				t.Errorf("%s needs UseStateForUnknown so the controller's value is held", name)
			}
		case schema.BoolAttribute:
			if a.Default != nil {
				t.Errorf("%s must not carry a static Default (#544)", name)
			}
			if len(a.PlanModifiers) == 0 {
				t.Errorf("%s needs UseStateForUnknown so the controller's value is held", name)
			}
		default:
			t.Errorf("%s is %T, want a string or bool attribute", name, a)
		}
	}

	for _, name := range firewallPolicyHeldAttributes {
		check(name, resp.Schema.Attributes[name])
	}
	for _, endpoint := range []string{"source", "destination"} {
		nested, ok := resp.Schema.Attributes[endpoint].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf(
				"%s is %T, want schema.SingleNestedAttribute",
				endpoint,
				resp.Schema.Attributes[endpoint],
			)
		}
		for _, name := range firewallPolicyHeldEndpointAttributes {
			check(endpoint+"."+name, nested.Attributes[name])
		}
	}
}

// TestFirewallPolicyAdoptedPlanKeepsControllerValues drives the real plan step
// for a policy whose controller values all differ from the former defaults,
// with a configuration that declares only the required attributes (including
// required-only source/destination blocks). The plan must be empty.
func TestFirewallPolicyAdoptedPlanKeepsControllerValues(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r := &firewallPolicyResource{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	overrides := map[string]attr.Value{
		"enabled":                 types.BoolValue(false),
		"protocol":                types.StringValue("udp"),
		"description":             types.StringValue("set in the UI"),
		"logging":                 types.BoolValue(true),
		"match_opposite_protocol": types.BoolValue(true),
		"create_allow_respond":    types.BoolValue(true),
		"ip_version":              types.StringValue("BOTH"),
	}
	for _, endpoint := range []string{"source", "destination"} {
		overrides[endpoint+".port_group_id"] = types.StringValue("6813ebd04ee8cb0f1f486beb")
		overrides[endpoint+".ip_group_id"] = types.StringValue("6813ebb54ee8cb0f1f486be9")
		overrides[endpoint+".port_matching_type"] = types.StringValue("OBJECT")
		overrides[endpoint+".match_opposite_ips"] = types.BoolValue(true)
		overrides[endpoint+".match_opposite_networks"] = types.BoolValue(true)
		overrides[endpoint+".match_opposite_ports"] = types.BoolValue(true)
		// Out of scope here (iid / match_opposite_iid / match_mac / mac keep
		// their defaults), pinned to them so the fixture stays valid.
		overrides[endpoint+".iid"] = types.StringValue("")
		overrides[endpoint+".mac"] = types.StringValue("")
	}
	f := planFixture{ctx: ctx, t: t, overrides: overrides}
	prior, config := f.object(schemaResp.Schema, "")
	config = withRequiredOnlyNested(t, schemaResp.Schema, prior, config, "source", "destination")

	planned := planResourceChange(
		t,
		"unifi_firewall_policy",
		schemaResp.Schema,
		prior,
		prior,
		config,
	)
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf("adopted policy planned a change at %s: %v -> %v", d.Path, d.Value1, d.Value2)
	}
}

// TestFirewallPolicyCreateDefaults checks the create side: an undeclared
// `enabled` is still planned `true`, and the attributes the controller assigns
// are left unknown for it to fill in.
func TestFirewallPolicyCreateDefaults(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&firewallPolicyResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t}
	prior, config := f.object(schemaResp.Schema, "")
	config = withRequiredOnlyNested(t, schemaResp.Schema, prior, config, "source", "destination")
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	created := planResourceChange(t, "unifi_firewall_policy", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)
	var attrs map[string]tftypes.Value
	if err := created.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}

	var enabled bool
	if err := attrs["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf("enabled = %v (%v), want true on create", attrs["enabled"], err)
	}
	var ipVersion string
	if err := attrs["ip_version"].As(&ipVersion); err != nil || ipVersion != "IPV4" {
		t.Errorf("ip_version = %v (%v), want IPV4 on create", attrs["ip_version"], err)
	}
	for _, name := range []string{"protocol", "logging", "create_allow_respond"} {
		if attrs[name].IsKnown() {
			t.Errorf(
				"%s = %v on create, want unknown (assigned by the controller)",
				name,
				attrs[name],
			)
		}
	}
}

// withRequiredOnlyNested rewrites the named Required single-nested attributes
// of config so that only their Required leaves are set, as in a configuration
// that writes `source = { zone_id = ..., matching_target = ... }` and nothing
// else. The leaves keep prior's values.
func withRequiredOnlyNested(
	t *testing.T,
	s schema.Schema,
	prior, config tftypes.Value,
	names ...string,
) tftypes.Value {
	t.Helper()
	var priorAttrs, configAttrs map[string]tftypes.Value
	if err := prior.As(&priorAttrs); err != nil {
		t.Fatalf("prior: %v", err)
	}
	if err := config.As(&configAttrs); err != nil {
		t.Fatalf("config: %v", err)
	}
	ctx := context.Background()
	for _, name := range names {
		nested, ok := s.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is not a single nested attribute", name)
		}
		var leaves map[string]tftypes.Value
		if err := priorAttrs[name].As(&leaves); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out := make(map[string]tftypes.Value, len(leaves))
		for leaf, v := range leaves {
			if nested.Attributes[leaf].IsRequired() {
				out[leaf] = v
			} else {
				out[leaf] = tftypes.NewValue(
					nested.Attributes[leaf].GetType().TerraformType(ctx),
					nil,
				)
			}
		}
		configAttrs[name] = tftypes.NewValue(nested.GetType().TerraformType(ctx), out)
	}
	return tftypes.NewValue(config.Type(), configAttrs)
}
