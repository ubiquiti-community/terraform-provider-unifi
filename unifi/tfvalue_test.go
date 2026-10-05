package unifi

import (
	"context"
	"math/big"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// tfValue builds a value of typ from a sparse Go value, the way a configuration
// would: map[string]any for objects and maps, []any for lists and sets, string,
// bool and int/int64/float64. Object attributes left out of the map are null.
func tfValue(t *testing.T, typ tftypes.Type, v any) tftypes.Value {
	t.Helper()
	if v == nil {
		return tftypes.NewValue(typ, nil)
	}
	switch {
	case typ.Is(tftypes.Object{}):
		obj, ok := typ.(tftypes.Object)
		if !ok {
			t.Fatalf("%s is not an object type", typ)
		}
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("value for %s must be map[string]any, got %T", typ, v)
		}
		for k := range m {
			if _, known := obj.AttributeTypes[k]; !known {
				t.Fatalf("unknown attribute %q", k)
			}
		}
		attrs := make(map[string]tftypes.Value, len(obj.AttributeTypes))
		for name, at := range obj.AttributeTypes {
			attrs[name] = tfValue(t, at, m[name])
		}
		return tftypes.NewValue(typ, attrs)
	case typ.Is(tftypes.List{}), typ.Is(tftypes.Set{}):
		var elem tftypes.Type
		if l, ok := typ.(tftypes.List); ok {
			elem = l.ElementType
		} else if s, ok := typ.(tftypes.Set); ok {
			elem = s.ElementType
		}
		items, ok := v.([]any)
		if !ok {
			t.Fatalf("value for %s must be []any, got %T", typ, v)
		}
		vals := make([]tftypes.Value, len(items))
		for i, item := range items {
			vals[i] = tfValue(t, elem, item)
		}
		return tftypes.NewValue(typ, vals)
	case typ.Is(tftypes.Map{}):
		mt, ok := typ.(tftypes.Map)
		if !ok {
			t.Fatalf("%s is not a map type", typ)
		}
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("value for %s must be map[string]any, got %T", typ, v)
		}
		vals := make(map[string]tftypes.Value, len(m))
		for k, item := range m {
			vals[k] = tfValue(t, mt.ElementType, item)
		}
		return tftypes.NewValue(typ, vals)
	case typ.Is(tftypes.Number):
		switch n := v.(type) {
		case int:
			return tftypes.NewValue(typ, big.NewFloat(float64(n)))
		case int64:
			return tftypes.NewValue(typ, big.NewFloat(float64(n)))
		case float64:
			return tftypes.NewValue(typ, big.NewFloat(n))
		}
		t.Fatalf("value for a number must be int, int64 or float64, got %T", v)
	}
	return tftypes.NewValue(typ, v)
}

// resourceSchema returns the schema of r.
func resourceSchema(t *testing.T, r fwresource.Resource) schema.Schema {
	t.Helper()
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

// tfPlan, tfConfig and tfState build the request objects for r from a sparse
// attribute map; a nil map gives a null (absent) object.
func tfPlan(t *testing.T, s schema.Schema, attrs map[string]any) tfsdk.Plan {
	t.Helper()
	return tfsdk.Plan{Schema: s, Raw: tfObject(t, s, attrs)}
}

func tfConfig(t *testing.T, s schema.Schema, attrs map[string]any) tfsdk.Config {
	t.Helper()
	return tfsdk.Config{Schema: s, Raw: tfObject(t, s, attrs)}
}

func tfState(t *testing.T, s schema.Schema, attrs map[string]any) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: s, Raw: tfObject(t, s, attrs)}
}

func tfObject(t *testing.T, s schema.Schema, attrs map[string]any) tftypes.Value {
	t.Helper()
	typ := s.Type().TerraformType(context.Background())
	if attrs == nil {
		return tftypes.NewValue(typ, nil)
	}
	return tfValue(t, typ, attrs)
}

// tfIdentity returns an empty identity object for r, for responses that set one.
func tfIdentity(t *testing.T, r fwresource.ResourceWithIdentity) *tfsdk.ResourceIdentity {
	t.Helper()
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("identity schema: %v", resp.Diagnostics)
	}
	typ := resp.IdentitySchema.Type().TerraformType(context.Background())
	return &tfsdk.ResourceIdentity{
		Schema: resp.IdentitySchema,
		Raw:    tftypes.NewValue(typ, nil),
	}
}
