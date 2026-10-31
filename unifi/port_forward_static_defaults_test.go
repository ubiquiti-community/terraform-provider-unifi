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
// (TransformDefaults) before it runs plan modifiers (SchemaModifyPlan).
//
// On unifi_port_forward this is the sharpest instance of the whole class,
// because the wrong value opens a hole rather than closing one. Measured
// against a live site with eight rules:
//
//	enabled   default true     vs 3 of 8 disabled on the controller
//	protocol  default tcp_udp  vs tcp on 5 of 8
//
// Verified live on a rule that is disabled and forwards to SSH: importing it
// without declaring `enabled` planned `false -> true`, which would have
// reopened the port to the internet, and `tcp -> tcp_udp` alongside it.
//
// `enabled` is deprecated in favour of removing the rule, but deprecation does
// not protect anyone: the attribute still works and its Default still wins.
func TestPortForwardNoStaticDefaultsOnAdoptedAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&portForwardResource{}).Schema(
		context.Background(),
		resource.SchemaRequest{},
		&resp,
	)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	enabled, ok := resp.Schema.Attributes["enabled"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("enabled is %T, want schema.BoolAttribute", resp.Schema.Attributes["enabled"])
	}
	if enabled.Default != nil {
		t.Error(
			"enabled must not carry a static Default: it is applied before plan " +
				"modifiers, so a rule disabled on the controller is planned back " +
				"to true and the apply reopens the port (#544)",
		)
	}
	if len(enabled.PlanModifiers) == 0 {
		t.Error("enabled needs UseStateForUnknown so the controller's value is held")
	}

	protocol, ok := resp.Schema.Attributes["protocol"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("protocol is %T, want schema.StringAttribute", resp.Schema.Attributes["protocol"])
	}
	if protocol.Default != nil {
		t.Error(
			"protocol must not carry a static Default: a tcp-only rule is " +
				"planned back to tcp_udp, widening the forward to UDP (#544)",
		)
	}
	if len(protocol.PlanModifiers) == 0 {
		t.Error("protocol needs UseStateForUnknown so the controller's value is held")
	}
}

// Companion to the schema guard above, covering the CREATE path, which has no
// controller value to hold: a rule created without declaring `enabled` must
// still be written disabled, so omitting the attribute never opens a port.
//
// This holds because unifi.PortForward.Enabled is a plain bool with no
// omitempty, and ValueBool() on an unknown value yields false - so the create
// body carries "enabled": false. Measured on a live controller by creating a
// probe rule with the attribute undeclared: the controller stored
// enabled=false. The test pins the serialization rather than the measurement.
func TestPortForwardCreateWithoutEnabledIsDisabled(t *testing.T) {
	r := &portForwardResource{}

	model := &portForwardResourceModel{
		Name: types.StringValue("probe"),
		// Enabled left unknown, as an undeclared Optional+Computed attribute
		// is during Create.
		Enabled:        types.BoolUnknown(),
		Protocol:       types.StringUnknown(),
		Logging:        types.BoolUnknown(),
		Wan:            types.ObjectNull(portForwardWanModel{}.AttributeTypes()),
		Forward:        types.ObjectNull(portForwardForwardModel{}.AttributeTypes()),
		SourceLimiting: types.ObjectNull(portForwardSourceLimitingModel{}.AttributeTypes()),
		DestinationIPs: types.ListNull(
			types.ObjectType{AttrTypes: portForwardDestinationIPModel{}.AttributeTypes()},
		),
	}

	pf, diags := r.modelToPortForward(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("modelToPortForward: %v", diags)
	}
	if pf.Enabled {
		t.Error(
			"a rule created without declaring enabled must be written disabled: " +
				"omitting the attribute must never open a WAN port (#544)",
		)
	}
}

// The three defaults #553 left in place, measured again on the same site (8
// rules): `log` is false on all 8 and `src_limiting_enabled` false on all 8,
// so the defaults agreed with the controller there - but both keys are always
// stored, so a rule with logging or source limiting switched on in the UI was
// planned back to false. `src` is stored on none of the 8: the controller keeps
// no source address unless one is sent (probed with disabled throwaway rules,
// including one limiting by firewall group), so `ip` gets no value at all on
// create instead of a fabricated "any".
func TestPortForwardNoStaticDefaultsOnRemainingAttributes(t *testing.T) {
	var resp resource.SchemaResponse
	(&portForwardResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}

	logging, ok := resp.Schema.Attributes["logging"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf("logging is %T, want schema.BoolAttribute", resp.Schema.Attributes["logging"])
	}
	if logging.Default != nil || len(logging.PlanModifiers) == 0 {
		t.Error("logging must hold the controller's value: no Default, UseStateForUnknown (#544)")
	}

	sl, ok := resp.Schema.Attributes["source_limiting"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("source_limiting is %T, want schema.SingleNestedAttribute",
			resp.Schema.Attributes["source_limiting"])
	}
	ip, ok := sl.Attributes["ip"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("source_limiting.ip is %T, want schema.StringAttribute", sl.Attributes["ip"])
	}
	if ip.Default != nil || len(ip.PlanModifiers) == 0 {
		t.Error(
			"source_limiting.ip must hold the controller's value: no Default, UseStateForUnknown (#544)",
		)
	}
	enabled, ok := sl.Attributes["enabled"].(schema.BoolAttribute)
	if !ok {
		t.Fatalf(
			"source_limiting.enabled is %T, want schema.BoolAttribute",
			sl.Attributes["enabled"],
		)
	}
	if enabled.Default != nil || len(enabled.PlanModifiers) == 0 {
		t.Error(
			"source_limiting.enabled must hold the controller's value: no Default, UseStateForUnknown (#544)",
		)
	}
}

// TestPortForwardAdoptedPlanKeepsControllerValues drives the real plan step for
// a rule with logging and IP source limiting switched on at the controller,
// configured with a `source_limiting` block that only names the type. The plan
// must be empty; with the defaults it proposed logging, the source address and
// source limiting itself back to false / "any".
func TestPortForwardAdoptedPlanKeepsControllerValues(t *testing.T) {
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	(&portForwardResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	slTypes := portForwardSourceLimitingModel{}.AttributeTypes()
	sl := func(ip types.String, enabled types.Bool) types.Object {
		return types.ObjectValueMust(slTypes, map[string]attr.Value{
			"ip":                ip,
			"firewall_group_id": types.StringNull(),
			"enabled":           enabled,
			"type":              types.StringValue("ip"),
		})
	}

	f := planFixture{ctx: ctx, t: t, overrides: map[string]attr.Value{
		"logging":         types.BoolValue(true),
		"source_limiting": sl(types.StringValue("198.51.100.0/24"), types.BoolValue(true)),
	}}
	prior, config := f.object(schemaResp.Schema, "")
	var configAttrs map[string]tftypes.Value
	if err := config.As(&configAttrs); err != nil {
		t.Fatalf("config: %v", err)
	}
	v, err := sl(types.StringNull(), types.BoolNull()).ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("source_limiting: %v", err)
	}
	configAttrs["source_limiting"] = v
	config = tftypes.NewValue(config.Type(), configAttrs)

	planned := planResourceChange(t, "unifi_port_forward", schemaResp.Schema, prior, prior, config)
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	for _, d := range diffs {
		t.Errorf(
			"adopted port forward planned a change at %s: %v -> %v",
			d.Path,
			d.Value1,
			d.Value2,
		)
	}
}

// TestPortForwardCreateSourceLimitingSendsNoSourceIP covers the create side of
// dropping the `ip` default: a rule limiting by firewall group no longer sends
// a fabricated src = "any", and source limiting is still sent as configured.
func TestPortForwardCreateSourceLimitingSendsNoSourceIP(t *testing.T) {
	r := &portForwardResource{}
	sl := types.ObjectValueMust(
		portForwardSourceLimitingModel{}.AttributeTypes(),
		map[string]attr.Value{
			"ip":                types.StringUnknown(),
			"firewall_group_id": types.StringValue("6813ebb54ee8cb0f1f486be9"),
			"enabled":           types.BoolValue(true),
			"type":              types.StringUnknown(),
		},
	)
	model := &portForwardResourceModel{
		Name:           types.StringValue("probe"),
		Enabled:        types.BoolUnknown(),
		Protocol:       types.StringUnknown(),
		Logging:        types.BoolUnknown(),
		Wan:            types.ObjectNull(portForwardWanModel{}.AttributeTypes()),
		Forward:        types.ObjectNull(portForwardForwardModel{}.AttributeTypes()),
		SourceLimiting: sl,
		DestinationIPs: types.ListNull(
			types.ObjectType{AttrTypes: portForwardDestinationIPModel{}.AttributeTypes()},
		),
	}

	pf, diags := r.modelToPortForward(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("modelToPortForward: %v", diags)
	}
	if pf.Src != "" {
		t.Errorf("src = %q, want it omitted when source_limiting.ip is not set", pf.Src)
	}
	if !pf.SrcLimitingEnabled || pf.SrcLimitingType != "firewall_group" ||
		pf.SrcFirewallGroupID != "6813ebb54ee8cb0f1f486be9" {
		t.Errorf("source limiting not sent as configured: %+v", pf)
	}
	if pf.Log {
		t.Error("a rule created without logging must be written with logging off")
	}
}
