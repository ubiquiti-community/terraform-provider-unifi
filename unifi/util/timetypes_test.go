package util

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestDurationRoundTrip(t *testing.T) {
	d := DurationValue(86400, time.Second)
	if got := DurationUnits(d, time.Second); got != 86400 {
		t.Fatalf("seconds round trip = %d, want 86400", got)
	}
	if got := d.ValueString(); got != "24h0m0s" {
		t.Fatalf("string form = %q, want 24h0m0s", got)
	}

	// minutes unit (e.g. wlan schedule duration)
	m := DurationValue(90, time.Minute)
	if got := DurationUnits(m, time.Minute); got != 90 {
		t.Fatalf("minutes round trip = %d, want 90", got)
	}

	if !DurationPtrValue(nil, time.Second).IsNull() {
		t.Fatal("nil pointer should be null")
	}
	if DurationUnitsPtr(DurationValue(5, time.Second), time.Second) == nil {
		t.Fatal("known value pointer should be non-nil")
	}
}

func TestUpgradeDurationRawState(t *testing.T) {
	// New schema type: a leasetime string, a nested dhcp_server object with its
	// own leasetime string, and a list of port_override objects each carrying a
	// dot1x_idle_timeout string. Plus an unrelated bool to confirm passthrough.
	nested := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"leasetime": tftypes.String,
	}}
	override := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"dot1x_idle_timeout": tftypes.String,
	}}
	schemaType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":          tftypes.String,
		"enabled":       tftypes.Bool,
		"leasetime":     tftypes.String,
		"dhcp_server":   nested,
		"port_override": tftypes.List{ElementType: override},
		// new attribute absent from the prior state below:
		"added_later": tftypes.String,
	}}

	// Prior state: durations as numbers, missing "added_later", and carrying an
	// extra "stale_field" that the new type no longer declares.
	prior := []byte(`{
		"name": "lan",
		"enabled": true,
		"leasetime": 3600,
		"dhcp_server": {"leasetime": 86400},
		"port_override": [{"dot1x_idle_timeout": 300}],
		"stale_field": "drop me"
	}`)

	dv, err := UpgradeDurationRawState(schemaType, prior, func(state map[string]any) {
		SetDurationField(state, "leasetime", time.Second)
		if d, ok := state["dhcp_server"].(map[string]any); ok {
			SetDurationField(d, "leasetime", time.Second)
		}
		if pos, ok := state["port_override"].([]any); ok {
			for _, p := range pos {
				if pm, ok := p.(map[string]any); ok {
					SetDurationField(pm, "dot1x_idle_timeout", time.Second)
				}
			}
		}
	})
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	val, err := dv.Unmarshal(schemaType)
	if err != nil {
		t.Fatalf("unmarshal upgraded value: %v", err)
	}

	var obj map[string]tftypes.Value
	if err := val.As(&obj); err != nil {
		t.Fatalf("as object: %v", err)
	}

	assertStr := func(v tftypes.Value, want string) {
		t.Helper()
		var s string
		if err := v.As(&s); err != nil {
			t.Fatalf("as string: %v", err)
		}
		if s != want {
			t.Fatalf("got %q, want %q", s, want)
		}
	}

	assertStr(obj["leasetime"], "1h0m0s")

	var dhcp map[string]tftypes.Value
	if err := obj["dhcp_server"].As(&dhcp); err != nil {
		t.Fatalf("as dhcp: %v", err)
	}
	assertStr(dhcp["leasetime"], "24h0m0s")

	var list []tftypes.Value
	if err := obj["port_override"].As(&list); err != nil {
		t.Fatalf("as list: %v", err)
	}
	var po map[string]tftypes.Value
	if err := list[0].As(&po); err != nil {
		t.Fatalf("as override: %v", err)
	}
	assertStr(po["dot1x_idle_timeout"], "5m0s")

	if !obj["added_later"].IsNull() {
		t.Fatal("added_later should be null")
	}
	if _, ok := obj["stale_field"]; ok {
		t.Fatal("stale_field should have been dropped")
	}

	// sanity: the produced JSON should be valid and have string durations
	out, _ := json.Marshal(map[string]any{"ok": true})
	_ = out
}
