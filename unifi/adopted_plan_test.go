package unifi

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// adoptedInputs builds the inputs of a plan for a resource the provider did
// not create (#544): the prior state holds what the controller reported on
// import, and the configuration declares only the Required attributes - plus,
// for each name in declare, an Optional nested object with only its Required
// leaves, as when a block is written to set one attribute.
//
// live pins prior-state values by dotted path (the controller's values,
// including nulls for keys the controller omits); config pins configuration
// values by dotted path.
func adoptedInputs(
	t *testing.T,
	r resource.Resource,
	declare []string,
	live, config map[string]attr.Value,
) (s schema.Schema, prior, cfg tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}
	s = schemaResp.Schema
	f := planFixture{ctx: ctx, t: t, overrides: live}
	prior, cfg = f.object(s, "")

	var priorAttrs, cfgAttrs map[string]tftypes.Value
	if err := prior.As(&priorAttrs); err != nil {
		t.Fatalf("prior as object: %v", err)
	}
	if err := cfg.As(&cfgAttrs); err != nil {
		t.Fatalf("config as object: %v", err)
	}
	declare = append([]string(nil), declare...)
	// A Required nested object is declared too, and like a declared Optional
	// one it carries only its Required leaves in configuration.
	for name, a := range s.Attributes {
		if single, ok := a.(schema.SingleNestedAttribute); ok && single.IsRequired() {
			declare = append(declare, name)
		}
	}
	for _, name := range declare {
		single, ok := s.Attributes[name].(schema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s is %T, want schema.SingleNestedAttribute", name, s.Attributes[name])
		}
		if live[name] == nil {
			priorAttrs[name] = f.synthesize(single, name)
		}
		_, leaves := f.attributes(single.Attributes, name+".")
		cfgAttrs[name] = tftypes.NewValue(single.GetType().TerraformType(ctx), leaves)
	}
	objType := s.Type().TerraformType(ctx)
	prior = tftypes.NewValue(objType, priorAttrs)
	cfg = tftypes.NewValue(objType, cfgAttrs)
	for p, v := range config {
		cfg = setAtPath(t, cfg, strings.Split(p, "."), f.toTerraform(v))
	}
	return s, prior, cfg
}

// adoptedPlan plans an unchanged configuration against an adopted resource
// (see adoptedInputs) and returns the prior and planned states.
func adoptedPlan(
	t *testing.T,
	typeName string,
	r resource.Resource,
	declare []string,
	live, config map[string]attr.Value,
) (prior, planned tftypes.Value) {
	t.Helper()
	s, prior, cfg := adoptedInputs(t, r, declare, live, config)
	return prior, planResourceChange(t, typeName, s, prior, prior, cfg)
}

// createPlan plans a create (null prior state) of the configuration
// adoptedInputs builds, and returns the planned attributes.
func createPlan(
	t *testing.T,
	typeName string,
	r resource.Resource,
	declare []string,
) map[string]tftypes.Value {
	t.Helper()
	s, _, cfg := adoptedInputs(t, r, declare, nil, nil)
	null := tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)
	planned := planResourceChange(t, typeName, s, null, cfg, cfg)
	var attrs map[string]tftypes.Value
	if err := planned.As(&attrs); err != nil {
		t.Fatalf("planned as object: %v", err)
	}
	return attrs
}

// assertPlanned fails unless v is known and equal to want (a string, bool or
// int64).
func assertPlanned(t *testing.T, v tftypes.Value, name string, want any) {
	t.Helper()
	if !v.IsKnown() || v.IsNull() {
		t.Errorf("%s = %v, want %v", name, v, want)
		return
	}
	switch w := want.(type) {
	case string:
		var got string
		if err := v.As(&got); err != nil || got != w {
			t.Errorf("%s = %v, want %q", name, v, w)
		}
	case bool:
		var got bool
		if err := v.As(&got); err != nil || got != w {
			t.Errorf("%s = %v, want %t", name, v, w)
		}
	case int64:
		var got big.Float
		if err := v.As(&got); err != nil {
			t.Errorf("%s = %v, want %d", name, v, w)
			return
		}
		if i, _ := got.Int64(); i != w {
			t.Errorf("%s = %v, want %d", name, v, w)
		}
	default:
		t.Fatalf("assertPlanned: unsupported want type %T", want)
	}
}

// assertPlannedUnknown fails unless v is unknown: an attribute whose value is
// left to the controller (or to a create-time fallback in the resource).
func assertPlannedUnknown(t *testing.T, v tftypes.Value, name string) {
	t.Helper()
	if v.IsKnown() {
		t.Errorf("%s = %v, want unknown (taken from the controller)", name, v)
	}
}

// setAtPath returns v with the attribute at the dotted path replaced by nv.
func setAtPath(t *testing.T, v tftypes.Value, path []string, nv tftypes.Value) tftypes.Value {
	t.Helper()
	if len(path) == 0 {
		return nv
	}
	var attrs map[string]tftypes.Value
	if err := v.As(&attrs); err != nil || attrs == nil {
		t.Fatalf("set %s: parent is not a known object (%v)", strings.Join(path, "."), err)
	}
	child, ok := attrs[path[0]]
	if !ok {
		t.Fatalf("set: no attribute %q", path[0])
	}
	attrs[path[0]] = setAtPath(t, child, path[1:], nv)
	return tftypes.NewValue(v.Type(), attrs)
}

// assertNoPlannedChanges fails for every attribute the plan changed.
func assertNoPlannedChanges(t *testing.T, prior, planned tftypes.Value) {
	t.Helper()
	diffs, err := prior.Diff(planned)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	lines := make([]string, 0, len(diffs))
	for _, d := range diffs {
		lines = append(lines, fmt.Sprintf("  %s: %v -> %v", d.Path, d.Value1, d.Value2))
	}
	sort.Strings(lines)
	if len(lines) > 0 {
		t.Errorf("adopted resource planned changes the configuration does not ask for "+
			"(#544):\n%s", strings.Join(lines, "\n"))
	}
}
