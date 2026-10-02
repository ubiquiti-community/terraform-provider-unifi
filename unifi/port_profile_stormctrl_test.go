package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// The storm-control, rate-limit and priority-queue attributes were hardcoded to
// null/false on read regardless of what the controller held, so a configured
// value could never survive the post-apply read: Create failed with "Provider
// produced inconsistent result after apply", left the resource tainted, and the
// next apply looped on the same error (#496).
//
// Unit rather than acceptance test: the conversion is what regressed, and it
// needs no controller.
func TestPortProfileStormctrlRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	i64 := func(v int64) *int64 { return &v }

	api := &unifi.PortProfile{
		Name:                         "test",
		EgressRateLimitKbps:          i64(64000),
		EgressRateLimitKbpsEnabled:   true,
		PriorityQueue1Level:          i64(10),
		PriorityQueue2Level:          i64(20),
		PriorityQueue3Level:          i64(30),
		PriorityQueue4Level:          i64(40),
		StormctrlBroadcastastEnabled: true,
		StormctrlBroadcastastRate:    i64(1000),
		StormctrlMcastEnabled:        true,
		StormctrlMcastRate:           i64(2000),
		StormctrlType:                "rate",
		StormctrlUcastEnabled:        true,
		StormctrlUcastRate:           i64(3000),
	}

	model := &portProfileResourceModel{Name: types.StringValue("test")}
	if d := r.portProfileToModel(ctx, api, model, "default"); d.HasError() {
		t.Fatalf("conversion: %v", d)
	}

	for _, tc := range []struct {
		name string
		got  types.Int64
		want int64
	}{
		{"egress_rate_limit_kbps", model.EgressRateLimitKbps, 64000},
		{"priority_queue1_level", model.PriorityQueue1Level, 10},
		{"priority_queue2_level", model.PriorityQueue2Level, 20},
		{"priority_queue3_level", model.PriorityQueue3Level, 30},
		{"priority_queue4_level", model.PriorityQueue4Level, 40},
		{"stormctrl_bcast_rate", model.StormctrlBcastRate, 1000},
		{"stormctrl_mcast_rate", model.StormctrlMcastRate, 2000},
		{"stormctrl_ucast_rate", model.StormctrlUcastRate, 3000},
	} {
		if tc.got.IsNull() || tc.got.IsUnknown() {
			t.Errorf("%s should be known, got %v", tc.name, tc.got)
			continue
		}
		if tc.got.ValueInt64() != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got.ValueInt64(), tc.want)
		}
	}

	for _, tc := range []struct {
		name string
		got  types.Bool
	}{
		{"egress_rate_limit_kbps_enabled", model.EgressRateLimitKbpsEnabled},
		{"stormctrl_bcast_enabled", model.StormctrlBcastEnabled},
		{"stormctrl_mcast_enabled", model.StormctrlMcastEnabled},
		{"stormctrl_ucast_enabled", model.StormctrlUcastEnabled},
	} {
		if !tc.got.ValueBool() {
			t.Errorf("%s = false, want true", tc.name)
		}
	}

	if got := model.StormctrlType.ValueString(); got != "rate" {
		t.Errorf("stormctrl_type = %q, want %q", got, "rate")
	}
}

// A controller that holds none of these must still read back as null rather
// than zero, so an unset attribute stays unset instead of planning 0.
func TestPortProfileStormctrlAbsentReadsNull(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	model := &portProfileResourceModel{Name: types.StringValue("test")}
	api := &unifi.PortProfile{Name: "test"}
	if d := r.portProfileToModel(ctx, api, model, "default"); d.HasError() {
		t.Fatalf("conversion: %v", d)
	}

	for _, tc := range []struct {
		name string
		got  types.Int64
	}{
		{"egress_rate_limit_kbps", model.EgressRateLimitKbps},
		{"stormctrl_bcast_rate", model.StormctrlBcastRate},
		{"stormctrl_mcast_level", model.StormctrlMcastLevel},
		{"stormctrl_ucast_rate", model.StormctrlUcastRate},
	} {
		if !tc.got.IsNull() {
			t.Errorf("%s = %v, want null", tc.name, tc.got)
		}
	}
	if !model.StormctrlType.IsNull() {
		t.Errorf("stormctrl_type = %v, want null", model.StormctrlType)
	}
}
