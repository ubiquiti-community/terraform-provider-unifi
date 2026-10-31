package unifi

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The controller stores an unset egress priority as "" and validates
// wan_egress_qos as [1-7] or "", so a 0 fails the whole update with
// api.err.InvalidPayload (go-unifi#105). egress_qos.priority used to default
// to 0 and was sent back as 0.

// wanEgressQoSDocument is wanTestDocument with egress QoS disabled and no
// priority, as a WAN configured in the UI stores it.
var wanEgressQoSDocument = strings.Replace(wanTestDocument,
	`"wan_vlan_enabled": false,`,
	`"wan_vlan_enabled": false,
	"wan_egress_qos_enabled": false,
	"wan_egress_qos": "",`, 1)

func egressQoSConfig(enabled bool, priority types.Int64) attr.Value {
	return types.ObjectValueMust(egressQosModel{}.AttributeTypes(), map[string]attr.Value{
		"enabled":  types.BoolValue(enabled),
		"priority": priority,
	})
}

func Test_wanResource_EgressQoSPriority(t *testing.T) {
	statePriority := func(h *updateHarness) types.Int64 {
		t.Helper()
		var v types.Int64
		if diags := h.state.GetAttribute(h.ctx,
			path.Root("egress_qos").AtName("priority"), &v); diags.HasError() {
			t.Fatalf("egress_qos.priority: %v", diags)
		}
		return v
	}

	t.Run("an unset priority imports as null and plans nothing", func(t *testing.T) {
		h := newWANHarnessFrom(t, wanEgressQoSDocument)
		if p := statePriority(h); !p.IsNull() {
			t.Fatalf("egress_qos.priority after import = %v, want null", p)
		}
		h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(wanBaseConfig())))

		cfg := wanBaseConfig()
		cfg["egress_qos"] = egressQoSConfig(false, types.Int64Null())
		h.assertEqual("plan with egress_qos declared", h.state.Raw, h.plan(h.config(cfg)))
	})

	t.Run("enabling without a priority never sends 0", func(t *testing.T) {
		h := newWANHarnessFrom(t, wanEgressQoSDocument)
		cfg := wanBaseConfig()
		cfg["egress_qos"] = egressQoSConfig(true, types.Int64Null())
		assertBodies(t, h.apply(h.config(cfg)), `{"wan_egress_qos_enabled":true}`)
		if p := statePriority(h); !p.IsNull() {
			t.Errorf("egress_qos.priority after apply = %v, want null", p)
		}
	})

	t.Run("a configured priority is sent with enabled, and kept when removed", func(t *testing.T) {
		h := newWANHarnessFrom(t, wanEgressQoSDocument)
		cfg := wanBaseConfig()
		cfg["egress_qos"] = egressQoSConfig(true, types.Int64Value(5))
		assertBodies(t, h.apply(h.config(cfg)),
			`{"wan_egress_qos":5,"wan_egress_qos_enabled":true}`)

		cfg["egress_qos"] = egressQoSConfig(true, types.Int64Null())
		assertBodies(t, h.apply(h.config(cfg)))
		if p := statePriority(h); p.ValueInt64() != 5 {
			t.Errorf("egress_qos.priority = %v, want 5", p)
		}
	})

	// A state written while priority defaulted to 0, planned without a
	// refresh: the 0 must stay off the request.
	t.Run("a legacy 0 in state is not sent", func(t *testing.T) {
		h := newWANHarnessFrom(t, wanEgressQoSDocument)
		if diags := h.state.SetAttribute(h.ctx,
			path.Root("egress_qos").AtName("priority"), types.Int64Value(0)); diags.HasError() {
			t.Fatalf("set legacy priority: %v", diags)
		}
		cfg := wanBaseConfig()
		cfg["egress_qos"] = egressQoSConfig(true, types.Int64Null())
		config := h.config(cfg)
		sent := h.fake.putCount()
		if diags := h.update(h.plan(config), config); diags.HasError() {
			t.Fatalf("Update: %v", diags)
		}
		assertBodies(t, h.fake.putsSince(sent), `{"wan_egress_qos_enabled":true}`)
		if p := statePriority(h); !p.IsNull() {
			t.Errorf("egress_qos.priority after apply = %v, want null", p)
		}
	})

	t.Run("a legacy 0 in state refreshes to null", func(t *testing.T) {
		h := newWANHarnessFrom(t, wanEgressQoSDocument)
		if diags := h.state.SetAttribute(h.ctx,
			path.Root("egress_qos").AtName("priority"), types.Int64Value(0)); diags.HasError() {
			t.Fatalf("set legacy priority: %v", diags)
		}
		h.state = h.read()
		if p := statePriority(h); !p.IsNull() {
			t.Errorf("egress_qos.priority after refresh = %v, want null", p)
		}
	})
}

func TestWANEgressQoSPrioritySchema(t *testing.T) {
	var resp fwresource.SchemaResponse
	(&wanResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, &resp)
	qos, ok := resp.Schema.Attributes["egress_qos"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatal("egress_qos is not a single nested attribute")
	}
	priority, ok := qos.Attributes["priority"].(schema.Int64Attribute)
	if !ok {
		t.Fatal("egress_qos.priority is not an int64 attribute")
	}
	if priority.Default != nil {
		t.Error("egress_qos.priority has a static default; an unset priority must stay null")
	}
	if !priority.Optional || !priority.Computed {
		t.Error("egress_qos.priority must be Optional + Computed")
	}
}

func TestWANEgressQOSPriority(t *testing.T) {
	for _, tc := range []struct {
		in   types.Int64
		want *int64
	}{
		{types.Int64Null(), nil},
		{types.Int64Unknown(), nil},
		{types.Int64Value(0), nil},
		{types.Int64Value(3), new(int64(3))},
	} {
		got := wanEgressQOSPriority(tc.in)
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("wanEgressQOSPriority(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestWANUpgradeState_egressQoSPriorityZeroToNull guards the v1 -> v2
// upgrade (and v0, which runs the same rewrite after nesting).
func TestWANUpgradeState_egressQoSPriorityZeroToNull(t *testing.T) {
	ctx := context.Background()
	r := &wanResource{}
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	upgrade := func(t *testing.T, version int64, prior string) tftypes.Value {
		t.Helper()
		up, ok := r.UpgradeState(ctx)[version]
		if !ok {
			t.Fatalf("no upgrader registered for schema version %d", version)
		}
		resp := &fwresource.UpgradeStateResponse{}
		up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
			RawState: &tfprotov6.RawState{JSON: []byte(prior)},
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("upgrade failed: %v", resp.Diagnostics)
		}
		val, err := resp.DynamicValue.Unmarshal(schemaType)
		if err != nil {
			t.Fatalf("unmarshal upgraded value: %v", err)
		}
		var root map[string]tftypes.Value
		if err := val.As(&root); err != nil {
			t.Fatalf("as object: %v", err)
		}
		var qos map[string]tftypes.Value
		if err := root["egress_qos"].As(&qos); err != nil {
			t.Fatalf("egress_qos: %v", err)
		}
		var enabled bool
		if err := qos["enabled"].As(&enabled); err != nil || !enabled {
			t.Errorf("egress_qos.enabled = %v, want true", qos["enabled"])
		}
		return qos["priority"]
	}

	for _, version := range []int64{0, 1} {
		got := upgrade(t, version, `{"id": "wan-1", "name": "Internet",
			"egress_qos": {"enabled": true, "priority": 0}}`)
		if !got.IsNull() {
			t.Errorf("v%d: egress_qos.priority 0 upgraded to %v, want null", version, got)
		}

		got = upgrade(t, version, `{"id": "wan-1", "name": "Internet",
			"egress_qos": {"enabled": true, "priority": 4}}`)
		var f big.Float
		if err := got.As(&f); err != nil {
			t.Fatalf("v%d: egress_qos.priority = %v: %v", version, got, err)
		}
		if n, _ := f.Int64(); n != 4 {
			t.Errorf("v%d: egress_qos.priority = %d, want 4", version, n)
		}
	}
}
