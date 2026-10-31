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
// adopted legacy firewall rule that leaves them out of its configuration:
//
//   - `enabled` re-enabled a rule disabled in the UI;
//   - `source` / `destination` / `state` object Defaults replaced the whole
//     group, planning NETv4 over ADDRv4, nulls over the controller's
//     addresses and ports, and false over every state flag;
//   - `logging` and `protocol_match_excepted` were reset to false.
//
// The create-time values are kept, in ModifyPlan.
func TestFirewallRuleNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&firewallRuleResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	check := func(name string, a schema.Attribute) {
		t.Helper()
		var hasDefault bool
		var modifiers int
		switch a := a.(type) {
		case schema.BoolAttribute:
			hasDefault, modifiers = a.Default != nil, len(a.PlanModifiers)
		case schema.StringAttribute:
			hasDefault, modifiers = a.Default != nil, len(a.PlanModifiers)
		case schema.SingleNestedAttribute:
			hasDefault, modifiers = a.Default != nil, len(a.PlanModifiers)
		default:
			t.Fatalf("%s is %T", name, a)
		}
		if hasDefault {
			t.Errorf("%s must not carry a static Default (#544)", name)
		}
		if modifiers == 0 {
			t.Errorf("%s needs UseStateForUnknown so the controller's value is held", name)
		}
	}

	for _, name := range []string{
		"enabled", "logging", "protocol_match_excepted", "source", "destination", "state",
	} {
		check(name, resp.Schema.Attributes[name])
	}
	nestedAttr := func(name string) schema.SingleNestedAttribute {
		t.Helper()
		nested, ok := resp.Schema.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf(
				"%s is %T, want schema.SingleNestedAttribute",
				name,
				resp.Schema.Attributes[name],
			)
		}
		return nested
	}
	for _, group := range []string{"source", "destination"} {
		check(group+".network_type", nestedAttr(group).Attributes["network_type"])
	}
	state := nestedAttr("state")
	for _, name := range []string{"established", "invalid", "new", "related"} {
		check("state."+name, state.Attributes[name])
	}
}

func firewallRuleAdoptedOverrides() map[string]attr.Value {
	return map[string]attr.Value{
		"enabled":                  types.BoolValue(false),
		"logging":                  types.BoolValue(true),
		"protocol_match_excepted":  types.BoolValue(true),
		"source.network_type":      types.StringValue("ADDRv4"),
		"source.address":           types.StringValue("192.0.2.10"),
		"destination.network_type": types.StringValue("ADDRv4"),
		"destination.address":      types.StringValue("198.51.100.0/24"),
		"destination.port":         types.StringValue("443"),
		"state.established":        types.BoolValue(true),
		"state.related":            types.BoolValue(true),
		"state.new":                types.BoolValue(true),
		"state.invalid":            types.BoolValue(true),
	}
}

// TestFirewallRuleAdoptedPlanKeepsControllerValues drives the real plan step
// for a rule whose controller values all differ from the former defaults,
// configured with only the required attributes: the plan must be empty.
func TestFirewallRuleAdoptedPlanKeepsControllerValues(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&firewallRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t, overrides: firewallRuleAdoptedOverrides()}
	prior, config := f.object(schemaResp.Schema, "")

	t.Run("groups omitted", func(t *testing.T) {
		planned := planResourceChange(
			t,
			"unifi_firewall_rule",
			schemaResp.Schema,
			prior,
			prior,
			config,
		)
		assertNoPlanDiff(t, prior, planned)
	})

	// A group declared with its address but not its network type keeps the
	// controller's ADDRv4 rather than being planned NETv4.
	t.Run("groups declared without network_type", func(t *testing.T) {
		var priorAttrs, configAttrs map[string]tftypes.Value
		if err := prior.As(&priorAttrs); err != nil {
			t.Fatal(err)
		}
		if err := config.As(&configAttrs); err != nil {
			t.Fatal(err)
		}
		for _, group := range []string{"source", "destination"} {
			var leaves map[string]tftypes.Value
			if err := priorAttrs[group].As(&leaves); err != nil {
				t.Fatal(err)
			}
			out := make(map[string]tftypes.Value, len(leaves))
			for k, v := range leaves {
				if k == "network_type" {
					v = tftypes.NewValue(tftypes.String, nil)
				}
				out[k] = v
			}
			configAttrs[group] = tftypes.NewValue(priorAttrs[group].Type(), out)
		}
		cfg := tftypes.NewValue(config.Type(), configAttrs)
		planned := planResourceChange(
			t,
			"unifi_firewall_rule",
			schemaResp.Schema,
			prior,
			prior,
			cfg,
		)
		assertNoPlanDiff(t, prior, planned)
	})
}

func assertNoPlanDiff(t *testing.T, prior, planned tftypes.Value) {
	t.Helper()
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf("adopted rule planned a change at %s: %v -> %v", d.Path, d.Value1, d.Value2)
	}
}

// TestFirewallRuleCreatePlansFormerDefaults checks that a create still plans
// what the schema Defaults used to supply, so the create body is unchanged.
func TestFirewallRuleCreatePlansFormerDefaults(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&firewallRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t}
	_, config := f.object(schemaResp.Schema, "")
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	created := planResourceChange(t, "unifi_firewall_rule", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)
	var attrs map[string]tftypes.Value
	if err := created.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}

	var enabled bool
	if err := attrs["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf("enabled = %v (%v), want true on create", attrs["enabled"], err)
	}
	for _, group := range []string{"source", "destination"} {
		g := plannedObject(t, attrs[group], group)
		var nt string
		if err := g["network_type"].As(&nt); err != nil || nt != "NETv4" {
			t.Errorf("%s.network_type = %v (%v), want NETv4", group, g["network_type"], err)
		}
	}
	state := plannedObject(t, attrs["state"], "state")
	for name, v := range state {
		var b bool
		if err := v.As(&b); err != nil || b {
			t.Errorf("state.%s = %v (%v), want false", name, v, err)
		}
	}
}
