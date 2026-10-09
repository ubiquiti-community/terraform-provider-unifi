package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
	"github.com/ubiquiti-community/terraform-provider-unifi/unifi/util"
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

	erl, _, d := util.ObjectAs[portEgressRateLimitModel](ctx, model.EgressRateLimit)
	if d.HasError() {
		t.Fatalf("egress_rate_limit: %v", d)
	}
	if erl.Kbps.ValueInt64() != 64000 || !erl.Enabled.ValueBool() {
		t.Errorf("egress_rate_limit = %+v, want kbps=64000 enabled=true", erl)
	}

	for name, got := range map[string]types.Int64{
		"priority_queue1_level": model.PriorityQueue1Level,
		"priority_queue2_level": model.PriorityQueue2Level,
		"priority_queue3_level": model.PriorityQueue3Level,
		"priority_queue4_level": model.PriorityQueue4Level,
	} {
		if got.IsNull() || got.IsUnknown() {
			t.Errorf("%s should be known, got %v", name, got)
		}
	}

	sc, d := portStormctrlFromObject(ctx, model.Stormctrl)
	if d.HasError() {
		t.Fatalf("stormctrl: %v", d)
	}
	if sc.Type != "rate" {
		t.Errorf("stormctrl.type = %q, want %q", sc.Type, "rate")
	}
	if !sc.BcastEnabled || !sc.McastEnabled || !sc.UcastEnabled {
		t.Errorf("stormctrl classes should all be enabled, got %+v", sc)
	}
	for name, tc := range map[string]struct {
		got  *int64
		want int64
	}{
		"stormctrl.bcast.rate": {sc.BcastRate, 1000},
		"stormctrl.mcast.rate": {sc.McastRate, 2000},
		"stormctrl.ucast.rate": {sc.UcastRate, 3000},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("%s = %v, want %d", name, tc.got, tc.want)
		}
	}
}

// A controller that holds none of these must still read back as unset rather
// than zero, so an unset attribute stays unset instead of planning 0.
func TestPortProfileStormctrlAbsentReadsNull(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	model := &portProfileResourceModel{Name: types.StringValue("test")}
	api := &unifi.PortProfile{Name: "test"}
	if d := r.portProfileToModel(ctx, api, model, "default"); d.HasError() {
		t.Fatalf("conversion: %v", d)
	}

	erl, _, _ := util.ObjectAs[portEgressRateLimitModel](ctx, model.EgressRateLimit)
	if !erl.Kbps.IsNull() {
		t.Errorf("egress_rate_limit.kbps = %v, want null", erl.Kbps)
	}
	sc, d := portStormctrlFromObject(ctx, model.Stormctrl)
	if d.HasError() {
		t.Fatalf("stormctrl: %v", d)
	}
	if sc.BcastRate != nil || sc.McastLevel != nil || sc.UcastRate != nil {
		t.Errorf("stormctrl rates/levels should be unset, got %+v", sc)
	}
	if sc.Type != "" {
		t.Errorf("stormctrl.type = %q, want empty", sc.Type)
	}
}
