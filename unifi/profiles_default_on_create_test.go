package unifi

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The attributes below were Optional+Computed with a static schema Default.
// The framework applies a Default before any plan modifier runs, so an object
// adopted from the controller (import, or a resource whose configuration
// leaves the attribute out) planned the constant over the controller's value
// on its first apply (#544). Each case pins every such attribute to a
// non-default controller value and plans a configuration that sets only the
// required attributes: the plan must keep the controller's values. The create
// check proves a new object still gets the old defaults.
func TestProfilesNoStaticDefaultsOnAdoptedObjects(t *testing.T) {
	cases := []struct {
		typeName string
		resource resource.Resource
		adopted  map[string]attr.Value
		created  map[string]any
	}{
		{
			typeName: "unifi_port_profile",
			resource: &portProfileResource{},
			adopted: map[string]attr.Value{
				"autoneg":                types.BoolValue(false),
				"full_duplex":            types.BoolValue(true),
				"isolation":              types.BoolValue(true),
				"port_keepalive_enabled": types.BoolValue(true),
				"dot1x.ctrl":             types.StringValue("mac_based"),
				"dot1x.idle_timeout":     timetypes.NewGoDurationValueFromStringMust("10m0s"),
				"lldpmed.enabled":        types.BoolValue(false),
			},
			created: map[string]any{
				"autoneg":            true,
				"dot1x.ctrl":         "force_authorized",
				"dot1x.idle_timeout": "5m0s",
				"lldpmed.enabled":    true,
			},
		},
		{
			typeName: "unifi_radius_profile",
			resource: &radiusProfileResource{},
			adopted: map[string]attr.Value{
				// The built-in Default profile: use_usg_auth_server = true on
				// the controller, against the old `false` default.
				"use_usg_auth_server":     types.BoolValue(true),
				"use_usg_acct_server":     types.BoolValue(true),
				"accounting_enabled":      types.BoolValue(true),
				"interim_update.enabled":  types.BoolValue(true),
				"interim_update.interval": timetypes.NewGoDurationValueFromStringMust("10m0s"),
				"vlan.enabled":            types.BoolValue(true),
				"vlan.wlan_mode":          types.StringValue("required"),
			},
			created: map[string]any{
				"interim_update.enabled":  false,
				"interim_update.interval": "1h0m0s",
				"vlan.enabled":            false,
				"vlan.wlan_mode":          "",
			},
		},
		{
			typeName: "unifi_radius_user",
			resource: &radiusUserResource{},
			adopted: map[string]attr.Value{
				"tunnel.type":        types.Int64Value(13),
				"tunnel.medium_type": types.Int64Value(1),
			},
			created: map[string]any{
				"tunnel.type":        int64(3),
				"tunnel.medium_type": int64(6),
			},
		},
		{
			// The controller stores no tunnel type ("" read as null since
			// go-unifi v1.42.0): the plan must keep null, not write 3/6.
			typeName: "unifi_radius_user",
			resource: &radiusUserResource{},
			adopted: map[string]attr.Value{
				"tunnel.type":        types.Int64Null(),
				"tunnel.medium_type": types.Int64Null(),
			},
		},
		{
			typeName: "unifi_power_supervisor",
			resource: &powerSupervisorResource{},
			adopted: map[string]attr.Value{
				"enabled":            types.BoolValue(false),
				"heartbeat_interval": timetypes.NewGoDurationValueFromStringMust("30s"),
				"silence_threshold":  timetypes.NewGoDurationValueFromStringMust("5m0s"),
				"power_off_duration": timetypes.NewGoDurationValueFromStringMust("10s"),
			},
			created: map[string]any{
				"enabled":            true,
				"heartbeat_interval": "1m0s",
				"silence_threshold":  "15m0s",
				"power_off_duration": "2m0s",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			ctx := context.Background()
			s := resourceSchema(t, tc.resource)
			schemaType := s.Type().TerraformType(ctx)

			for name := range tc.adopted {
				if def, ok := attributeDefault(ctx, schemaAttributeAt(t, s, name)); ok {
					t.Errorf("%s still carries a static Default (%v)", name, def)
				}
			}

			f := planFixture{ctx: ctx, t: t, overrides: tc.adopted}
			prior, config := f.object(s, "")

			planned := planResourceChange(t, tc.typeName, s, prior, prior, config)
			if diffs, err := prior.Diff(planned); err != nil || len(diffs) != 0 {
				for _, d := range diffs {
					t.Errorf("adopted object: %s planned %v -> %v", d.Path, d.Value1, d.Value2)
				}
			}

			created := planResourceChange(t, tc.typeName, s,
				tftypes.NewValue(schemaType, nil), config, config)
			for name, want := range tc.created {
				got := plannedAt(t, created, name)
				if !got.IsKnown() || got.IsNull() {
					t.Errorf("create: %s = %v, want %v", name, got, want)
					continue
				}
				if !plannedEquals(t, got, want) {
					t.Errorf("create: %s = %v, want %v", name, got, want)
				}
			}
		})
	}
}

// A RADIUS server port left out of configuration defaults to 1812/1813 only
// for a server with no prior element; an existing server keeps the
// controller's port, including none (a stored "" reads as null).
func TestRadiusProfileServerPortDefaultsOnNewServersOnly(t *testing.T) {
	ctx := context.Background()
	s := resourceSchema(t, &radiusProfileResource{})
	schemaType := s.Type().TerraformType(ctx)
	f := planFixture{ctx: ctx, t: t}
	prior, config := f.object(s, "")

	listType, ok := s.Blocks["auth_server"].Type().TerraformType(ctx).(tftypes.List)
	if !ok {
		t.Fatal("auth_server is not a list block")
	}
	objType, ok := listType.ElementType.(tftypes.Object)
	if !ok {
		t.Fatal("auth_server element is not an object")
	}
	server := func(port any) tftypes.Value {
		return tftypes.NewValue(objType, map[string]tftypes.Value{
			"ip":     tftypes.NewValue(tftypes.String, nil),
			"port":   tftypes.NewValue(tftypes.Number, port),
			"secret": tftypes.NewValue(tftypes.String, "s"),
		})
	}
	with := func(base tftypes.Value, servers ...tftypes.Value) tftypes.Value {
		var attrs map[string]tftypes.Value
		if err := base.As(&attrs); err != nil {
			t.Fatalf("as object: %v", err)
		}
		attrs["auth_server"] = tftypes.NewValue(listType, servers)
		return tftypes.NewValue(schemaType, attrs)
	}
	ports := func(planned tftypes.Value) []tftypes.Value {
		var attrs map[string]tftypes.Value
		if err := planned.As(&attrs); err != nil {
			t.Fatalf("as object: %v", err)
		}
		var elems []tftypes.Value
		if err := attrs["auth_server"].As(&elems); err != nil {
			t.Fatalf("auth_server as list: %v", err)
		}
		out := make([]tftypes.Value, 0, len(elems))
		for _, e := range elems {
			var m map[string]tftypes.Value
			if err := e.As(&m); err != nil {
				t.Fatalf("server as object: %v", err)
			}
			out = append(out, m["port"])
		}
		return out
	}

	// Adopted: the controller holds one server with no port and one on 1645.
	priorState := with(prior, server(nil), server(big.NewFloat(1645)))
	cfg := with(config, server(nil), server(nil), server(nil))
	// Core proposes the prior port for the elements it can pair by index.
	proposed := with(config, server(nil), server(big.NewFloat(1645)), server(nil))
	got := ports(planResourceChange(t, "unifi_radius_profile", s, priorState, proposed, cfg))
	if len(got) != 3 {
		t.Fatalf("planned %d servers, want 3", len(got))
	}
	if !got[0].IsNull() {
		t.Errorf("server 0 port = %v, want null as on the controller", got[0])
	}
	if !plannedEquals(t, got[1], int64(1645)) {
		t.Errorf("server 1 port = %v, want 1645", got[1])
	}
	if !plannedEquals(t, got[2], int64(1812)) {
		t.Errorf("new server port = %v, want 1812", got[2])
	}

	// Create: every server without a port gets 1812.
	createCfg := with(config, server(nil))
	got = ports(planResourceChange(t, "unifi_radius_profile", s,
		tftypes.NewValue(schemaType, nil), createCfg, createCfg))
	if len(got) != 1 || !plannedEquals(t, got[0], int64(1812)) {
		t.Errorf("create port = %v, want 1812", got)
	}
}

// schemaAttributeAt resolves a dotted attribute path through single nested
// attributes.
func schemaAttributeAt(t *testing.T, s schema.Schema, dotted string) schema.Attribute {
	t.Helper()
	attrs := s.Attributes
	var a schema.Attribute
	for _, part := range strings.Split(dotted, ".") {
		var ok bool
		if a, ok = attrs[part]; !ok {
			t.Fatalf("no attribute %s", dotted)
		}
		if single, ok := a.(schema.SingleNestedAttribute); ok {
			attrs = single.Attributes
		}
	}
	return a
}

// plannedAt returns the planned value at a dotted path.
func plannedAt(t *testing.T, planned tftypes.Value, dotted string) tftypes.Value {
	t.Helper()
	v := planned
	for _, part := range strings.Split(dotted, ".") {
		if !v.IsKnown() || v.IsNull() {
			return v
		}
		var m map[string]tftypes.Value
		if err := v.As(&m); err != nil {
			t.Fatalf("%s: as object: %v", dotted, err)
		}
		v = m[part]
	}
	return v
}

func plannedEquals(t *testing.T, v tftypes.Value, want any) bool {
	t.Helper()
	if !v.IsKnown() || v.IsNull() {
		return false
	}
	switch w := want.(type) {
	case bool:
		var b bool
		return v.As(&b) == nil && b == w
	case string:
		var s string
		return v.As(&s) == nil && s == w
	case int64:
		var n big.Float
		if v.As(&n) != nil {
			return false
		}
		i, _ := n.Int64()
		return i == w
	}
	t.Fatalf("unsupported want type %T", want)
	return false
}
