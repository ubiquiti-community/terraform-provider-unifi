package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestSettingGlobalSwitchPlansUnsetFieldsUnknown checks that global_switch
// fields left out of configuration plan unknown, or known from state, never
// null: a null plan fails the apply once the read-back fills the controller's
// value.
func TestSettingGlobalSwitchPlansUnsetFieldsUnknown(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schemaType, ok := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatal("schema type is not an object")
	}
	gsType := schemaType.AttributeTypes["global_switch"].(tftypes.Object)

	f := planFixture{ctx: ctx, t: t}
	prior, config := f.object(schemaResp.Schema, "")

	withGlobalSwitch := func(base tftypes.Value, gs map[string]tftypes.Value) tftypes.Value {
		var attrs map[string]tftypes.Value
		if err := base.As(&attrs); err != nil {
			t.Fatalf("as object: %v", err)
		}
		attrs["global_switch"] = tftypes.NewValue(gsType, gs)
		return tftypes.NewValue(schemaType, attrs)
	}
	configured := func() map[string]tftypes.Value {
		gs := map[string]tftypes.Value{}
		for name, typ := range gsType.AttributeTypes {
			gs[name] = tftypes.NewValue(typ, nil)
		}
		gs["jumboframe_enabled"] = tftypes.NewValue(tftypes.Bool, true)
		gs["stp_version"] = tftypes.NewValue(tftypes.String, "rstp")
		return gs
	}
	cfg := withGlobalSwitch(config, configured())

	planned := func(t *testing.T, prior, proposed tftypes.Value) map[string]tftypes.Value {
		t.Helper()
		p := planResourceChange(t, "unifi_setting", schemaResp.Schema, prior, proposed, cfg)
		var attrs map[string]tftypes.Value
		if err := p.As(&attrs); err != nil {
			t.Fatalf("planned as object: %v", err)
		}
		return plannedObject(t, attrs["global_switch"], "global_switch")
	}
	unset := []string{"acl_device_isolation", "dhcp_snoop", "dot1x_portctrl_enabled"}

	t.Run("create", func(t *testing.T) {
		got := planned(t, tftypes.NewValue(schemaType, nil), cfg)
		for _, name := range unset {
			if got[name].IsKnown() {
				t.Errorf("global_switch.%s plans %v, want unknown", name, got[name])
			}
		}
	})

	t.Run("update of a setting whose state has no global_switch", func(t *testing.T) {
		got := planned(t, prior, cfg)
		for _, name := range unset {
			if got[name].IsKnown() {
				t.Errorf("global_switch.%s plans %v, want unknown", name, got[name])
			}
		}
	})

	t.Run("update keeps the state's values for fields left out", func(t *testing.T) {
		priorGS := map[string]tftypes.Value{}
		for name, typ := range gsType.AttributeTypes {
			switch name {
			case "acl_device_isolation":
				priorGS[name] = tftypes.NewValue(typ, []tftypes.Value{})
			case "stp_version":
				priorGS[name] = tftypes.NewValue(typ, "stp")
			default:
				priorGS[name] = tftypes.NewValue(typ, false)
			}
		}
		proposedGS := map[string]tftypes.Value{}
		for name, v := range priorGS {
			proposedGS[name] = v
		}
		proposedGS["jumboframe_enabled"] = tftypes.NewValue(tftypes.Bool, true)
		proposedGS["stp_version"] = tftypes.NewValue(tftypes.String, "rstp")

		got := planned(t, withGlobalSwitch(prior, priorGS), withGlobalSwitch(prior, proposedGS))
		for _, name := range unset {
			if !got[name].IsKnown() || got[name].IsNull() {
				t.Errorf("global_switch.%s plans %v, want the value kept from state", name, got[name])
			}
		}
	})
}
