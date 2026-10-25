package util

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestResolveUnknowns(t *testing.T) {
	inner := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"enabled": tftypes.Bool,
		"rssi":    tftypes.Number,
	}}
	root := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":  tftypes.String,
		"mode":  tftypes.String,
		"inner": inner,
		"whole": inner,
		"bands": tftypes.Set{ElementType: tftypes.String},
	}}
	obj := func(enabled, rssi tftypes.Value) tftypes.Value {
		return tftypes.NewValue(inner, map[string]tftypes.Value{"enabled": enabled, "rssi": rssi})
	}
	bands := func(vals ...tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, vals)
	}
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	unknownStr := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

	prior := tftypes.NewValue(root, map[string]tftypes.Value{
		"name":  str("old"),
		"mode":  tftypes.NewValue(tftypes.String, nil),
		"inner": obj(tftypes.NewValue(tftypes.Bool, true), tftypes.NewValue(tftypes.Number, -70)),
		"whole": obj(tftypes.NewValue(tftypes.Bool, true), tftypes.NewValue(tftypes.Number, -80)),
		"bands": bands(str("2g"), str("5g")),
	})

	t.Run("unknowns take the prior value", func(t *testing.T) {
		plan := tftypes.NewValue(root, map[string]tftypes.Value{
			"name": str("new"),
			"mode": unknownStr,
			"inner": obj(
				tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
				tftypes.NewValue(tftypes.Number, -65),
			),
			"whole": tftypes.NewValue(inner, tftypes.UnknownValue),
			"bands": tftypes.NewValue(
				tftypes.Set{ElementType: tftypes.String},
				tftypes.UnknownValue,
			),
		})
		want := tftypes.NewValue(root, map[string]tftypes.Value{
			"name": str("new"),
			"mode": tftypes.NewValue(tftypes.String, nil),
			"inner": obj(
				tftypes.NewValue(tftypes.Bool, true),
				tftypes.NewValue(tftypes.Number, -65),
			),
			"whole": obj(
				tftypes.NewValue(tftypes.Bool, true),
				tftypes.NewValue(tftypes.Number, -80),
			),
			"bands": bands(str("2g"), str("5g")),
		})
		got, err := ResolveUnknowns(plan, prior)
		if err != nil {
			t.Fatalf("ResolveUnknowns: %v", err)
		}
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("unknown set element fails", func(t *testing.T) {
		plan := setAttr(t, prior.Copy(), "bands", bands(str("2g"), unknownStr))
		if _, err := ResolveUnknowns(plan, prior); err == nil {
			t.Fatal("want an error for an unknown set element")
		}
	})

	t.Run("unknown under a null prior object fails", func(t *testing.T) {
		nullInner := setAttr(t, prior.Copy(), "inner", tftypes.NewValue(inner, nil))
		plan := setAttr(t, prior.Copy(), "inner", obj(
			tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			tftypes.NewValue(tftypes.Number, -65),
		))
		if _, err := ResolveUnknowns(plan, nullInner); err == nil {
			t.Fatal("want an error for an unknown with no prior parent")
		}
	})
}

func setAttr(t *testing.T, obj tftypes.Value, name string, v tftypes.Value) tftypes.Value {
	t.Helper()
	var attrs map[string]tftypes.Value
	if err := obj.As(&attrs); err != nil {
		t.Fatalf("as object: %v", err)
	}
	attrs[name] = v
	return tftypes.NewValue(obj.Type(), attrs)
}
