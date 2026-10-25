package util

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestResolveUnknowns(t *testing.T) {
	inner := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"enabled": tftypes.Bool,
		"rssi":    tftypes.Number,
	}}
	group := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"inner": inner,
		"tags":  tftypes.Set{ElementType: tftypes.String},
	}}
	root := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":  tftypes.String,
		"mode":  tftypes.String,
		"inner": inner,
		"whole": inner,
		"bands": tftypes.Set{ElementType: tftypes.String},
		"group": group,
	}}
	obj := func(enabled, rssi tftypes.Value) tftypes.Value {
		return tftypes.NewValue(inner, map[string]tftypes.Value{"enabled": enabled, "rssi": rssi})
	}
	bands := func(vals ...tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, vals)
	}
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	unknownStr := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

	priorGroup := tftypes.NewValue(group, map[string]tftypes.Value{
		"inner": obj(tftypes.NewValue(tftypes.Bool, false), tftypes.NewValue(tftypes.Number, -60)),
		"tags":  bands(str("a")),
	})
	prior := tftypes.NewValue(root, map[string]tftypes.Value{
		"name":  str("old"),
		"mode":  tftypes.NewValue(tftypes.String, nil),
		"inner": obj(tftypes.NewValue(tftypes.Bool, true), tftypes.NewValue(tftypes.Number, -70)),
		"whole": obj(tftypes.NewValue(tftypes.Bool, true), tftypes.NewValue(tftypes.Number, -80)),
		"bands": bands(str("2g"), str("5g")),
		"group": priorGroup,
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
			"group": tftypes.NewValue(group, tftypes.UnknownValue),
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
			"group": priorGroup,
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

	t.Run("unknown under a null prior object is null", func(t *testing.T) {
		nullInner := setAttr(t, prior.Copy(), "inner", tftypes.NewValue(inner, nil))
		plan := setAttr(t, prior.Copy(), "inner", obj(
			tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
			tftypes.NewValue(tftypes.Number, -65),
		))
		want := setAttr(t, prior.Copy(), "inner", obj(
			tftypes.NewValue(tftypes.Bool, nil),
			tftypes.NewValue(tftypes.Number, -65),
		))
		got, err := ResolveUnknowns(plan, nullInner)
		if err != nil {
			t.Fatalf("ResolveUnknowns: %v", err)
		}
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("unknown object under a null prior object is null", func(t *testing.T) {
		nullGroup := setAttr(t, prior.Copy(), "group", tftypes.NewValue(group, nil))
		plan := setAttr(t, prior.Copy(), "group", tftypes.NewValue(group, map[string]tftypes.Value{
			"inner": tftypes.NewValue(inner, tftypes.UnknownValue),
			"tags":  tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		}))
		want := setAttr(t, prior.Copy(), "group", tftypes.NewValue(group, map[string]tftypes.Value{
			"inner": tftypes.NewValue(inner, nil),
			"tags":  tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		}))
		got, err := ResolveUnknowns(plan, nullGroup)
		if err != nil {
			t.Fatalf("ResolveUnknowns: %v", err)
		}
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("unknown set element under a null prior object fails", func(t *testing.T) {
		nullGroup := setAttr(t, prior.Copy(), "group", tftypes.NewValue(group, nil))
		plan := setAttr(t, prior.Copy(), "group", tftypes.NewValue(group, map[string]tftypes.Value{
			"inner": tftypes.NewValue(inner, nil),
			"tags":  bands(str("a"), unknownStr),
		}))
		_, err := ResolveUnknowns(plan, nullGroup)
		if err == nil || !strings.Contains(err.Error(), "unknown value has no prior value") {
			t.Fatalf("err = %v, want an error naming the unresolved value", err)
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
