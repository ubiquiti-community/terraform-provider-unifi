package unifi

import (
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
		{"min_rssi 0 dropped when disabled", unifi.DeviceRadioTable{MinRssiEnabled: false, MinRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil }},
		{"min_rssi kept when enabled+valid", unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(-82)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi != nil && *r.MinRssi == -82 }},
		{"min_rssi >=0 dropped even if enabled", unifi.DeviceRadioTable{MinRssiEnabled: true, MinRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.MinRssi == nil }},
		{"maxsta 0 dropped", unifi.DeviceRadioTable{Maxsta: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.Maxsta == nil }},
		{"sens_level 0 dropped when disabled", unifi.DeviceRadioTable{SensLevelEnabled: false, SensLevel: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.SensLevel == nil }},
		{"assisted_roaming_rssi 0 dropped when disabled", unifi.DeviceRadioTable{AssistedRoamingEnabled: false, AssistedRoamingRssi: i64(0)},
			func(r unifi.DeviceRadioTable) bool { return r.AssistedRoamingRssi == nil }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := c.in
			sanitizeRadioForUpdate(&r)
			if !c.want(r) {
				t.Fatalf("sanitize failed: %+v", r)
			}
		})
	}
}

func TestBuildMinimalUpdateDevice_PreservesPortOverrides(t *testing.T) {
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
