package util

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNestFields(t *testing.T) {
	obj := map[string]any{
		"name":               "sw",
		"lcm_brightness":     json.Number("50"),
		"lcm_idle_timeout":   "10m0s",
		"lcm_night_mode_end": "22:00",
	}
	NestFields(obj, "lcm", map[string]string{
		"lcm_brightness":   "brightness",
		"lcm_idle_timeout": "idle_timeout",
		"lcm_missing":      "missing",
	})
	lcm, ok := obj["lcm"].(map[string]any)
	if !ok {
		t.Fatalf("lcm not nested: %#v", obj)
	}
	if lcm["brightness"] != json.Number("50") || lcm["idle_timeout"] != "10m0s" {
		t.Errorf("moved fields = %#v", lcm)
	}
	if _, ok := lcm["missing"]; ok {
		t.Error("absent source key must not create a nested key")
	}
	if _, ok := obj["lcm_brightness"]; ok {
		t.Error("flat key should have been removed")
	}
	if obj["name"] != "sw" {
		t.Error("unrelated keys must pass through")
	}

	// A second call merges into the existing nested map (deeper nesting).
	NestFields(obj, "lcm", map[string]string{"lcm_night_mode_end": "night_mode_end"})
	NestFields(lcm, "night_mode", map[string]string{"night_mode_end": "ends"})
	nm, ok := lcm["night_mode"].(map[string]any)
	if !ok || nm["ends"] != "22:00" {
		t.Errorf("deep nesting failed: %#v", obj["lcm"])
	}

	// No matching keys: target untouched.
	other := map[string]any{"a": 1}
	NestFields(other, "grp", map[string]string{"x": "y"})
	if _, ok := other["grp"]; ok {
		t.Error("target must not be created when no source key exists")
	}
}

func TestEachObjectAndWithObject(t *testing.T) {
	obj := map[string]any{
		"port_override": []any{
			map[string]any{"dot1x_ctrl": "auto"},
			"not-an-object",
			map[string]any{"dot1x_ctrl": "force_authorized"},
		},
		"dhcp_server": map[string]any{"dns_enabled": true},
	}
	n := 0
	EachObject(obj, "port_override", func(m map[string]any) {
		n++
		NestFields(m, "dot1x", map[string]string{"dot1x_ctrl": "ctrl"})
	})
	if n != 2 {
		t.Errorf("EachObject visited %d objects, want 2", n)
	}
	list, ok := obj["port_override"].([]any)
	if !ok || len(list) != 3 {
		t.Fatalf("port_override list = %#v", obj["port_override"])
	}
	first, ok := list[0].(map[string]any)
	if !ok {
		t.Fatalf("first element = %#v", list[0])
	}
	if d, ok := first["dot1x"].(map[string]any); !ok || d["ctrl"] != "auto" {
		t.Errorf("nested port override = %#v", first)
	}
	WithObject(obj, "dhcp_server", func(m map[string]any) {
		NestFields(m, "dns", map[string]string{"dns_enabled": "enabled"})
	})
	ds, ok := obj["dhcp_server"].(map[string]any)
	if !ok {
		t.Fatalf("dhcp_server = %#v", obj["dhcp_server"])
	}
	if d, ok := ds["dns"].(map[string]any); !ok || d["enabled"] != true {
		t.Errorf("WithObject nesting = %#v", ds)
	}
	EachObject(obj, "absent", func(map[string]any) { t.Error("must not be called") })
	WithObject(obj, "absent", func(map[string]any) { t.Error("must not be called") })
}

func TestUpgradeRawStateNestsIntoSchema(t *testing.T) {
	nightMode := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"begins": tftypes.String,
		"ends":   tftypes.String,
	}}
	lcm := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"brightness": tftypes.Number,
		"night_mode": nightMode,
	}}
	schemaType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String,
		"lcm":  lcm,
	}}
	prior := []byte(`{"name":"sw","lcm_brightness":40,"lcm_night_mode_begins":"20:00"}`)

	dv, err := UpgradeRawState(schemaType, prior, func(state map[string]any) {
		NestFields(state, "lcm", map[string]string{
			"lcm_brightness":         "brightness",
			"lcm_night_mode_begins":  "night_mode_begins",
			"lcm_night_mode_ends":    "night_mode_ends",
			"lcm_idle_timeout_never": "never",
		})
		WithObject(state, "lcm", func(m map[string]any) {
			NestFields(m, "night_mode", map[string]string{
				"night_mode_begins": "begins",
				"night_mode_ends":   "ends",
			})
		})
	})
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	val, err := dv.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var obj map[string]tftypes.Value
	if err := val.As(&obj); err != nil {
		t.Fatalf("as object: %v", err)
	}
	var lcmVal map[string]tftypes.Value
	if err := obj["lcm"].As(&lcmVal); err != nil {
		t.Fatalf("as lcm: %v", err)
	}
	var nm map[string]tftypes.Value
	if err := lcmVal["night_mode"].As(&nm); err != nil {
		t.Fatalf("as night_mode: %v", err)
	}
	var begins string
	if err := nm["begins"].As(&begins); err != nil || begins != "20:00" {
		t.Errorf("begins = %q (%v), want 20:00", begins, err)
	}
	if !nm["ends"].IsNull() {
		t.Errorf("ends should be null, got %v", nm["ends"])
	}
}

func testObjTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"a": types.StringType,
		"b": types.Int64Type,
		"n": types.ObjectType{AttrTypes: testNestedTypes()},
	}
}

func testNestedTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"x": types.BoolType,
		"y": types.StringType,
	}
}

func testObj(a types.String, b types.Int64, x types.Bool, y types.String) types.Object {
	nested := types.ObjectValueMust(
		testNestedTypes(),
		map[string]attr.Value{"x": x, "y": y},
	)
	return types.ObjectValueMust(testObjTypes(), map[string]attr.Value{
		"a": a, "b": b, "n": nested,
	})
}

func TestObjectAs(t *testing.T) {
	ctx := context.Background()
	type m struct {
		A types.String `tfsdk:"a"`
		B types.Int64  `tfsdk:"b"`
		N types.Object `tfsdk:"n"`
	}
	if _, ok, _ := ObjectAs[m](ctx, types.ObjectNull(testObjTypes())); ok {
		t.Error("null object must report !ok")
	}
	if _, ok, _ := ObjectAs[m](ctx, types.ObjectUnknown(testObjTypes())); ok {
		t.Error("unknown object must report !ok")
	}
	got, ok, diags := ObjectAs[m](ctx, testObj(
		types.StringValue("v"), types.Int64Value(2), types.BoolValue(true), types.StringNull(),
	))
	if !ok || diags.HasError() {
		t.Fatalf("known object: ok=%v diags=%v", ok, diags)
	}
	if got.A.ValueString() != "v" || got.B.ValueInt64() != 2 {
		t.Errorf("decoded = %+v", got)
	}
}

func TestResolveUnknownObject(t *testing.T) {
	ctx := context.Background()
	src := testObj(
		types.StringValue(
			"api",
		),
		types.Int64Value(7),
		types.BoolValue(true),
		types.StringValue("y"),
	)

	// Unknown dst -> src wholesale.
	if got := ResolveUnknownObject(ctx, types.ObjectUnknown(testObjTypes()), src); !got.Equal(src) {
		t.Errorf("unknown dst = %v, want src", got)
	}
	// Null dst untouched.
	null := types.ObjectNull(testObjTypes())
	if got := ResolveUnknownObject(ctx, null, src); !got.Equal(null) {
		t.Errorf("null dst = %v, want null", got)
	}
	// Partially unknown, including a nested unknown sub-attribute.
	dst := testObj(
		types.StringValue("cfg"), types.Int64Unknown(), types.BoolUnknown(), types.StringNull(),
	)
	got := ResolveUnknownObject(ctx, dst, src)
	attrs := got.Attributes()
	if attrAs[types.String](t, attrs["a"]).ValueString() != "cfg" {
		t.Errorf("configured a overwritten: %v", attrs["a"])
	}
	if attrAs[types.Int64](t, attrs["b"]).ValueInt64() != 7 {
		t.Errorf("unknown b not resolved: %v", attrs["b"])
	}
	n := attrAs[types.Object](t, attrs["n"]).Attributes()
	if !attrAs[types.Bool](t, n["x"]).ValueBool() {
		t.Errorf("nested unknown x not resolved: %v", n["x"])
	}
	if !n["y"].IsNull() {
		t.Errorf("nested null y must stay null: %v", n["y"])
	}
	// Fully known dst is returned as-is.
	if got := ResolveUnknownObject(ctx, src, dst); !got.Equal(src) {
		t.Errorf("known dst changed: %v", got)
	}
}

func TestOverlayKnownObject(t *testing.T) {
	ctx := context.Background()
	applied := testObj(
		types.StringValue(
			"api",
		),
		types.Int64Value(1),
		types.BoolValue(false),
		types.StringValue("y"),
	)
	planned := testObj(
		types.StringValue("cfg"), types.Int64Null(), types.BoolValue(true), types.StringUnknown(),
	)
	got := OverlayKnownObject(ctx, planned, applied).Attributes()
	if attrAs[types.String](t, got["a"]).ValueString() != "cfg" {
		t.Errorf("planned a not re-asserted: %v", got["a"])
	}
	if attrAs[types.Int64](t, got["b"]).ValueInt64() != 1 {
		t.Errorf("null planned b must keep applied: %v", got["b"])
	}
	n := attrAs[types.Object](t, got["n"]).Attributes()
	if !attrAs[types.Bool](t, n["x"]).ValueBool() {
		t.Errorf("nested planned x not re-asserted: %v", n["x"])
	}
	if attrAs[types.String](t, n["y"]).ValueString() != "y" {
		t.Errorf("unknown planned y must keep applied: %v", n["y"])
	}
	if got := OverlayKnownObject(
		ctx,
		types.ObjectNull(testObjTypes()),
		applied,
	); !got.Equal(
		applied,
	) {
		t.Errorf("null planned = %v, want applied", got)
	}
	if got := OverlayKnownObject(
		ctx,
		planned,
		types.ObjectUnknown(testObjTypes()),
	); !got.Equal(
		planned,
	) {
		t.Errorf("unknown applied = %v, want planned", got)
	}
}

// attrAs is a checked type assertion for attr.Value in tests.
func attrAs[T attr.Value](t *testing.T, v attr.Value) T {
	t.Helper()
	out, ok := v.(T)
	if !ok {
		t.Fatalf("value %v is %T, want %T", v, v, out)
	}
	return out
}
