package unifi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func i64(v int64) *int64 { return &v }

func TestSanitizeRadioForUpdate(t *testing.T) {
	cases := []struct {
		name string
		in   unifi.DeviceRadioTable
		want func(unifi.DeviceRadioTable) bool
	}{
		{
			"min_rssi 0 dropped when disabled",
			unifi.DeviceRadioTable{MinRssiEnabled: false, MinRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil },
		},
		{
			"min_rssi kept when enabled+valid",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-82)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi != nil && *r.MinRssi == -82 },
		},
		{
			"min_rssi >=0 dropped even if enabled",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil },
		},
		{
			"min_rssi out-of-range high (-10) dropped",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-10)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil },
		},
		{
			"min_rssi out-of-range low (-95) dropped",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-95)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil },
		},
		{
			"min_rssi boundary -90 kept",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-90)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi != nil },
		},
		{
			"maxsta 0 dropped",
			unifi.DeviceRadioTable{Maxsta: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.Maxsta == nil },
		},
		{
			"maxsta 201 out-of-range dropped",
			unifi.DeviceRadioTable{Maxsta: i64(201)},
			func(r unifi.DeviceRadioTable) bool { return r.Maxsta == nil },
		},
		{
			"maxsta 200 boundary kept",
			unifi.DeviceRadioTable{Maxsta: i64(200)},
			func(r unifi.DeviceRadioTable) bool { return r.Maxsta != nil && *r.Maxsta == 200 },
		},
		{
			"sens_level 0 dropped when disabled",
			unifi.DeviceRadioTable{SensLevelEnabled: false, SensLevel: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.SensLevel == nil },
		},
		{
			"sens_level out-of-range (-10) dropped even if enabled",
			unifi.DeviceRadioTable{SensLevelEnabled: true, SensLevel: i64(-10)},
			func(r unifi.DeviceRadioTable) bool { return r.SensLevel == nil },
		},
		{
			"sens_level in-range (-70) kept when enabled",
			unifi.DeviceRadioTable{SensLevelEnabled: true, SensLevel: i64(-70)},
			func(r unifi.DeviceRadioTable) bool { return r.SensLevel != nil },
		},
		{
			"assisted_roaming_rssi 0 dropped when disabled",
			unifi.DeviceRadioTable{AssistedRoamingEnabled: false, AssistedRoamingRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.AssistedRoamingRssi == nil },
		},
		{
			"assisted_roaming_rssi out-of-range (-10) dropped even if enabled",
			unifi.DeviceRadioTable{AssistedRoamingEnabled: true, AssistedRoamingRssi: i64(-10)},
			func(r unifi.DeviceRadioTable) bool { return r.AssistedRoamingRssi == nil },
		},
		{
			"assisted_roaming_rssi in-range (-70) kept when enabled",
			unifi.DeviceRadioTable{AssistedRoamingEnabled: true, AssistedRoamingRssi: i64(-70)},
			func(r unifi.DeviceRadioTable) bool { return r.AssistedRoamingRssi != nil },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.in
			_ = sanitizeRadioForUpdate(
				"ng",
				&r,
			) // diagnostic-emission behavior is covered by TestSanitizeRadioForUpdate_WarnsWhenEnabledAndOutOfRange
			if !c.want(r) {
				t.Fatalf("sanitize failed: %+v", r)
			}
		})
	}
}

// TestSanitizeRadioForUpdate_WarnsWhenEnabledAndOutOfRange covers review feedback on
// PR #378: an out-of-range value is dropped either way (the controller would reject
// it), but if the field was ENABLED — the user actually declared and turned on that
// setting — the drop must be visible as a warning, not a silent no-op. Disabled (or
// simply unset) fields drop silently, same as before: that's the normal/expected case.
func TestSanitizeRadioForUpdate_WarnsWhenEnabledAndOutOfRange(t *testing.T) {
	cases := []struct {
		name      string
		in        unifi.DeviceRadioTable
		wantWarn  bool
		wantField string
	}{
		{
			"min_rssi enabled+out-of-range warns",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-10)},
			true,
			"min_rssi",
		},
		{
			"min_rssi disabled+out-of-range silent",
			unifi.DeviceRadioTable{MinRssiEnabled: false, MinRssi: i64(-10)},
			false,
			"",
		},
		{
			"min_rssi enabled+in-range silent",
			unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-80)},
			false,
			"",
		},
		{
			"maxsta out-of-range (non-zero) warns",
			unifi.DeviceRadioTable{Maxsta: i64(201)},
			true,
			"maxsta",
		},
		{"maxsta in-range silent", unifi.DeviceRadioTable{Maxsta: i64(50)}, false, ""},
		// maxsta=0 is the controller's "unset" sentinel (Optional+Computed,
		// UseStateForUnknown) — flows back on every update of a device that never
		// configured maxsta. Must stay silent, not warn on every unrelated update.
		{
			"maxsta=0 (controller unset sentinel) silent, not warned",
			unifi.DeviceRadioTable{Maxsta: i64(0)},
			false,
			"",
		},
		{
			"sens_level enabled+out-of-range warns",
			unifi.DeviceRadioTable{SensLevelEnabled: true, SensLevel: i64(-10)},
			true,
			"sens_level",
		},
		{
			"sens_level disabled+out-of-range silent",
			unifi.DeviceRadioTable{SensLevelEnabled: false, SensLevel: i64(-10)},
			false,
			"",
		},
		{
			"assisted_roaming_rssi enabled+out-of-range warns",
			unifi.DeviceRadioTable{AssistedRoamingEnabled: true, AssistedRoamingRssi: i64(-10)},
			true,
			"assisted_roaming_rssi",
		},
		{
			"assisted_roaming_rssi disabled+out-of-range silent",
			unifi.DeviceRadioTable{AssistedRoamingEnabled: false, AssistedRoamingRssi: i64(-10)},
			false,
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.in
			diags := sanitizeRadioForUpdate("ng", &r)
			if c.wantWarn && len(diags) == 0 {
				t.Fatalf("expected a warning diagnostic, got none")
			}
			if !c.wantWarn && len(diags) != 0 {
				t.Fatalf("expected no diagnostics, got: %+v", diags)
			}
			if c.wantWarn {
				found := false
				for _, d := range diags {
					if strings.Contains(d.Detail(), c.wantField) &&
						strings.Contains(d.Detail(), "ng") {
						found = true
					}
				}
				if !found {
					t.Fatalf(
						"expected a warning mentioning field %q and radio %q, got: %+v",
						c.wantField,
						"ng",
						diags,
					)
				}
			}
		})
	}
}

func TestBuildMinimalUpdateDevice_UsesProvidedPortOverrides(t *testing.T) {
	// current device has real port overrides; deviceReq declares none.
	current := &unifi.Device{PortOverrides: []unifi.DevicePortOverrides{{PortIDX: i64(1)}}}
	req := &unifi.Device{ID: "x", MAC: "aa"}
	// mimic updateDevice's fallback: no declared overrides -> echo current
	po := req.PortOverrides
	if len(po) == 0 && current != nil {
		po = current.PortOverrides
	}
	out := buildMinimalUpdateDevice(req, current, po)
	if out.PortOverrides == nil {
		t.Fatalf("port_overrides should be preserved (non-nil), got nil -> would send null")
	}
	if len(out.PortOverrides) != 1 {
		t.Fatalf("expected 1 preserved override, got %d", len(out.PortOverrides))
	}
}

// TestBuildMinimalUpdateDevice_EmptyOverridesMarshalAsArrayNotNull: a device
// with no port overrides at all (an access point, a gateway) must send
// `port_overrides: []`, never `null`, which UDM/Dream Machine gateways reject
// with api.err.InvalidPayload (400).
func TestBuildMinimalUpdateDevice_EmptyOverridesMarshalAsArrayNotNull(t *testing.T) {
	req := &unifi.Device{ID: "x", MAC: "aa", MgmtNetworkID: "net99"}
	current := &unifi.Device{} // no overrides, e.g. an access point

	// No declared overrides and none on the device: updateDevice's fallback
	// passes currentDevice.PortOverrides through, which is nil here.
	out := buildMinimalUpdateDevice(req, current, current.PortOverrides)
	if out.PortOverrides == nil {
		t.Fatalf("port_overrides must be non-nil so it marshals to [] not null")
	}

	body, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), `"port_overrides":null`) {
		t.Fatalf("body must not contain port_overrides:null, got: %s", body)
	}
	if !strings.Contains(string(body), `"port_overrides":[]`) {
		t.Fatalf("body must contain port_overrides:[], got: %s", body)
	}
}

// TestFrameworkToRadioTable_PartialConfigOmitsUnknownFields guards #427: the
// reported repro declares radio_table with only radio/channel/ht/tx_power_mode set.
// Every other sub-field is Optional+Computed with no per-field plan modifier, so it
// arrives Unknown (not Null) in an update plan — and frameworkToRadioTable used to
// convert Unknown the same as a real configured 0/"" via ValueInt64Pointer /
// ValueString, instead of omitting it. The controller rejects the resulting
// min_rssi/sens_level/assisted_roaming_rssi/maxsta/antenna_gain/antenna_id: 0 with
// api.err.InvalidPayload (400). Unknown sub-fields must stay off the wire, exactly
// like Null ones, leaving only the declared channel/ht/tx_power(_mode) behind.
func TestFrameworkToRadioTable_PartialConfigOmitsUnknownFields(t *testing.T) {
	r := &deviceResource{}
	ctx := context.Background()
	attrTypes := radioTableAttrTypes()

	newRadio := func(radio, channel string, ht int64, txPowerMode, txPower string) types.Object {
		values := map[string]attr.Value{
			"radio":                    types.StringValue(radio),
			"channel":                  types.StringValue(channel),
			"ht":                       types.Int64Value(ht),
			"tx_power_mode":            types.StringValue(txPowerMode),
			"tx_power":                 types.StringUnknown(),
			"min_rssi_enabled":         types.BoolUnknown(),
			"min_rssi":                 types.Int64Unknown(),
			"antenna_gain":             types.Int64Unknown(),
			"antenna_id":               types.Int64Unknown(),
			"assisted_roaming_enabled": types.BoolUnknown(),
			"assisted_roaming_rssi":    types.Int64Unknown(),
			"dfs":                      types.BoolUnknown(),
			"hard_noise_floor_enabled": types.BoolUnknown(),
			"loadbalance_enabled":      types.BoolUnknown(),
			"maxsta":                   types.Int64Unknown(),
			"name":                     types.StringUnknown(),
			"sens_level":               types.Int64Unknown(),
			"sens_level_enabled":       types.BoolUnknown(),
			"vwire_enabled":            types.BoolUnknown(),
		}
		if txPower != "" {
			values["tx_power"] = types.StringValue(txPower)
		}
		obj, diags := types.ObjectValue(attrTypes, values)
		if diags.HasError() {
			t.Fatalf("building radio object: %v", diags)
		}
		return obj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{
		newRadio("6e", "117", 160, "auto", ""),
		newRadio("ng", "11", 20, "auto", ""),
		newRadio("na", "149", 80, "custom", "23"),
	})
	if diags.HasError() {
		t.Fatalf("building radio_table list: %v", diags)
	}

	got, convDiags := r.frameworkToRadioTable(ctx, list)
	if convDiags.HasError() {
		t.Fatalf("frameworkToRadioTable: %v", convDiags)
	}
	if len(got) != 3 {
		t.Fatalf("radio count = %d, want 3", len(got))
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	body := string(raw)

	for _, leaked := range []string{
		`"min_rssi"`, `"sens_level"`, `"assisted_roaming_rssi"`, `"maxsta"`,
		`"antenna_gain"`, `"antenna_id"`, `"name"`,
	} {
		if strings.Contains(body, leaked) {
			t.Errorf("PUT body leaked unconfigured radio_table field %s: %s", leaked, body)
		}
	}

	for _, want := range []string{
		`"channel":"117"`, `"ht":160`,
		`"channel":"11"`, `"ht":20`,
		`"channel":"149"`, `"ht":80`, `"tx_power":"23"`, `"tx_power_mode":"custom"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("PUT body missing declared radio_table field %s: %s", want, body)
		}
	}
}
