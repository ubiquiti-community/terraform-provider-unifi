package unifi

import (
	"encoding/json"
	"strings"
	"testing"

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

// TestBuildMinimalUpdateDevice_EmptyOverridesMirrorCurrentDevice: when nothing
// is declared and the device has no overrides, the body must mirror the
// current device's exact representation. go-unifi's UpdateDevice diffs the
// body against the existing device and only sends changed keys, so an
// identical value (null-for-null, []-for-[]) always drops out of the PUT.
// The previous unconditional `[]` manufactured a spurious null→[] diff on
// access points (whose existing port_overrides is null) and some controllers
// reject `port_overrides: []` on such devices with api.err.Invalid (#427).
func TestBuildMinimalUpdateDevice_EmptyOverridesMirrorCurrentDevice(t *testing.T) {
	t.Run("current null stays null so the diff drops the key", func(t *testing.T) {
		req := &unifi.Device{ID: "x", MAC: "aa", MgmtNetworkID: "net99"}
		current := &unifi.Device{} // no overrides at all, e.g. an access point

		out := buildMinimalUpdateDevice(req, current, current.PortOverrides)
		if out.PortOverrides != nil {
			t.Fatalf(
				"port_overrides = %#v, want nil to mirror the existing device's null "+
					"(anything else manufactures a diff and gets sent)",
				out.PortOverrides,
			)
		}
	})

	t.Run("current [] stays [] so the diff drops the key", func(t *testing.T) {
		req := &unifi.Device{ID: "x", MAC: "aa"}
		current := &unifi.Device{PortOverrides: []unifi.DevicePortOverrides{}}

		out := buildMinimalUpdateDevice(req, current, current.PortOverrides)
		if out.PortOverrides == nil {
			t.Fatalf("port_overrides must stay non-nil to mirror the existing device's []")
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
	})
}
