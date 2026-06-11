package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// nullPortOverrideAttrValues returns every port-override attribute set to its
// typed null, so a test only has to override the few fields it cares about.
func nullPortOverrideAttrValues() map[string]attr.Value {
	attrs := portOverrideAttrTypes()
	vals := make(map[string]attr.Value, len(attrs))
	for name, t := range attrs {
		switch tt := t.(type) {
		case basetypes.StringType:
			vals[name] = types.StringNull()
		case basetypes.Int64Type:
			vals[name] = types.Int64Null()
		case basetypes.BoolType:
			vals[name] = types.BoolNull()
		case basetypes.ListType:
			vals[name] = types.ListNull(tt.ElemType)
		}
		// Any unhandled attr type is intentionally left out so ObjectValue fails
		// loudly (signalling the helper needs updating) rather than silently.
	}
	return vals
}

func portOverrideSetWith(t *testing.T, overrides map[string]attr.Value) types.Set {
	t.Helper()
	attrs := nullPortOverrideAttrValues()
	for k, v := range overrides {
		attrs[k] = v
	}
	obj, d := types.ObjectValue(portOverrideAttrTypes(), attrs)
	if d.HasError() {
		t.Fatalf("building port override object: %v", d)
	}
	set, d := types.SetValue(
		types.ObjectType{AttrTypes: portOverrideAttrTypes()},
		[]attr.Value{obj},
	)
	if d.HasError() {
		t.Fatalf("building port override set: %v", d)
	}
	return set
}

// TestFrameworkToPortOverrides_AggregateOpMode guards #177: to form an SFP+ link
// aggregation the port's op_mode must be written as "aggregate" alongside the
// aggregate_members. op_mode is otherwise skipped (default "switch") so gateway
// devices that reject op_mode on PUT keep working (#213).
func TestFrameworkToPortOverrides_AggregateOpMode(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	members, d := types.ListValue(types.Int64Type, []attr.Value{
		types.Int64Value(9),
		types.Int64Value(10),
	})
	if d.HasError() {
		t.Fatalf("building members list: %v", d)
	}

	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":             types.Int64Value(9),
		"op_mode":           types.StringValue("aggregate"),
		"aggregate_members": members,
	})

	pos, diags := r.frameworkToPortOverrides(ctx, set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if len(pos) != 1 {
		t.Fatalf("got %d port overrides, want 1", len(pos))
	}
	po := pos[0]
	if po.OpMode != "aggregate" {
		t.Errorf("OpMode = %q, want aggregate (LAG would not engage)", po.OpMode)
	}
	if len(po.AggregateMembers) != 2 || po.AggregateMembers[0] != 9 ||
		po.AggregateMembers[1] != 10 {
		t.Errorf("AggregateMembers = %v, want [9 10]", po.AggregateMembers)
	}
}

// TestFrameworkToPortOverrides_SwitchOpModeOmitted ensures the default "switch"
// op_mode is not sent on the wire (it has omitempty), preserving the gateway
// write fix (#213).
func TestFrameworkToPortOverrides_SwitchOpModeOmitted(t *testing.T) {
	ctx := context.Background()
	r := &deviceResource{}

	set := portOverrideSetWith(t, map[string]attr.Value{
		"index":   types.Int64Value(1),
		"op_mode": types.StringValue("switch"),
	})

	pos, diags := r.frameworkToPortOverrides(ctx, set)
	if diags.HasError() {
		t.Fatalf("frameworkToPortOverrides errored: %v", diags)
	}
	if len(pos) != 1 {
		t.Fatalf("got %d port overrides, want 1", len(pos))
	}
	if pos[0].OpMode != "" {
		t.Errorf("OpMode = %q, want empty (omitted) for the switch default", pos[0].OpMode)
	}
}

// TestPortOverridesToFramework_TaggedNetworkIDsTypedNull is a regression test
// for #235. portOverridesToFramework must initialize the tagged_networkconf_ids
// model field to a typed null list. Previously it was left as an untyped
// zero-value types.List, which made types.ObjectValueFrom fail with a
// "types.ListType[!!! MISSING TYPE !!!]" Value Conversion Error during the
// Read/refresh (and import) of any unifi_device that has port overrides.
func TestPortOverridesToFramework_TaggedNetworkIDsTypedNull(t *testing.T) {
	r := &deviceResource{}

	set, diags := r.portOverridesToFramework(context.Background(), []unifi.DevicePortOverrides{
		{Name: "Port 1"},
	})

	if diags.HasError() {
		t.Fatalf(
			"portOverridesToFramework returned diagnostics (regression #235): %v",
			diags.Errors(),
		)
	}
	if set.IsNull() {
		t.Fatal("expected a non-null port_override set for a single override")
	}

	elems := set.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 port_override element, got %d", len(elems))
	}

	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("expected port_override element to be types.Object, got %T", elems[0])
	}

	taggedAttr, ok := obj.Attributes()["tagged_networkconf_ids"]
	if !ok {
		t.Fatal("port_override is missing the tagged_networkconf_ids attribute")
	}

	list, ok := taggedAttr.(types.List)
	if !ok {
		t.Fatalf("expected tagged_networkconf_ids to be types.List, got %T", taggedAttr)
	}
	if !list.IsNull() {
		t.Errorf("expected tagged_networkconf_ids to be a null list, got %v", list)
	}
	if et := list.ElementType(context.Background()); !et.Equal(types.StringType) {
		t.Errorf("expected tagged_networkconf_ids element type to be string, got %v", et)
	}
}
