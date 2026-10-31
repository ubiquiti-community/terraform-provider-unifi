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
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan). An
// adopted NAT rule that leaves these attributes out of its configuration was
// planned back to the constants: a rule disabled in the UI was re-enabled,
// and exclude/logging/pppoe_use_base_interface/invert_* were reset to false.
//
// The controller stores every one of these keys on create, defaulting the
// bools to false when they are omitted (probed with a disabled throwaway
// rule), so the bools are simply held. `enabled` keeps `true` on create only.
func TestNatNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&natResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	check := func(name string, a schema.Attribute) {
		t.Helper()
		b, ok := a.(schema.BoolAttribute)
		if !ok {
			t.Fatalf("%s is %T, want schema.BoolAttribute", name, a)
		}
		if b.Default != nil {
			t.Errorf("%s must not carry a static Default (#544)", name)
		}
		if len(b.PlanModifiers) == 0 {
			t.Errorf("%s needs UseStateForUnknown so the controller's value is held", name)
		}
	}
	for _, name := range []string{"enabled", "exclude", "logging", "pppoe_use_base_interface"} {
		check(name, resp.Schema.Attributes[name])
	}
	for _, filter := range []string{"source_filter", "destination_filter"} {
		nested, ok := resp.Schema.Attributes[filter].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf(
				"%s is %T, want schema.SingleNestedAttribute",
				filter,
				resp.Schema.Attributes[filter],
			)
		}
		for _, name := range []string{"invert_address", "invert_port"} {
			check(filter+"."+name, nested.Attributes[name])
		}
	}
}

// TestNatAdoptedPlanKeepsControllerValues drives the real plan step for a rule
// whose controller values all differ from the former defaults, configured with
// only the required attributes (and filters that set only filter_type). The
// plan must be empty.
func TestNatAdoptedPlanKeepsControllerValues(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&natResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	filterTypes := natFilterAttrTypes()
	filter := func(invert bool, filterType types.String) types.Object {
		vals := map[string]attr.Value{
			"filter_type":        filterType,
			"address":            types.StringNull(),
			"port":               types.StringNull(),
			"invert_address":     types.BoolNull(),
			"invert_port":        types.BoolNull(),
			"network_conf_id":    types.StringNull(),
			"firewall_group_ids": types.SetNull(types.StringType),
		}
		if invert {
			vals["address"] = types.StringValue("192.0.2.0/24")
			vals["port"] = types.StringValue("443")
			vals["invert_address"] = types.BoolValue(true)
			vals["invert_port"] = types.BoolValue(true)
		}
		return types.ObjectValueMust(filterTypes, vals)
	}

	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"enabled":                  types.BoolValue(false),
		"exclude":                  types.BoolValue(true),
		"logging":                  types.BoolValue(true),
		"pppoe_use_base_interface": types.BoolValue(true),
		"source_filter":            filter(true, types.StringValue("ADDRESS_AND_PORT")),
		"destination_filter":       filter(true, types.StringValue("ADDRESS_AND_PORT")),
	}}
	prior, config := f.object(schemaResp.Schema, "")

	var configAttrs map[string]tftypes.Value
	if err := config.As(&configAttrs); err != nil {
		t.Fatalf("config: %v", err)
	}
	// The filters declare only filter_type; address and port are Optional-only
	// and must be configured to be kept, so they stay in the configuration.
	for _, name := range []string{"source_filter", "destination_filter"} {
		cfg := filter(false, types.StringValue("ADDRESS_AND_PORT"))
		attrs := cfg.Attributes()
		attrs["address"] = types.StringValue("192.0.2.0/24")
		attrs["port"] = types.StringValue("443")
		v, err := types.ObjectValueMust(filterTypes, attrs).ToTerraformValue(ctx)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		configAttrs[name] = v
	}
	config = tftypes.NewValue(config.Type(), configAttrs)

	planned := planResourceChange(t, "unifi_nat", schemaResp.Schema, prior, prior, config)
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf("adopted NAT rule planned a change at %s: %v -> %v", d.Path, d.Value1, d.Value2)
	}
}

// TestNatCreatePlansEnabled checks that a rule created without `enabled` is
// still planned enabled, as it was with the schema Default.
func TestNatCreatePlansEnabled(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&natResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	f := planFixture{ctx: ctx, t: t}
	_, config := f.object(schemaResp.Schema, "")
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	created := planResourceChange(t, "unifi_nat", schemaResp.Schema,
		tftypes.NewValue(schemaType, nil), config, config)
	var attrs map[string]tftypes.Value
	if err := created.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	var enabled bool
	if err := attrs["enabled"].As(&enabled); err != nil || !enabled {
		t.Errorf("enabled = %v (%v), want true on create", attrs["enabled"], err)
	}
	for _, name := range []string{"exclude", "logging", "pppoe_use_base_interface"} {
		if attrs[name].IsKnown() {
			t.Errorf(
				"%s = %v on create, want unknown (assigned by the controller)",
				name,
				attrs[name],
			)
		}
	}
}
