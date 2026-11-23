package unifi

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func Test_wanResource_slaID(t *testing.T) {
	t.Run("attaching a profile sends exactly wan_sla", func(t *testing.T) {
		h := newWANHarness(t)
		cfg := wanBaseConfig()
		cfg["sla_id"] = types.StringValue("00000000-0000-4000-8000-000000000001")
		assertBodies(
			t,
			h.apply(h.config(cfg)),
			`{"wan_sla":"00000000-0000-4000-8000-000000000001"}`,
		)

		// Leaving it out afterwards keeps the controller's value (Optional + Computed).
		delete(cfg, "sla_id")
		assertBodies(t, h.apply(h.config(cfg)))
		if got := h.fake.stored("wan_sla"); got != "00000000-0000-4000-8000-000000000001" {
			t.Errorf("stored wan_sla = %v, want kept", got)
		}

		// An empty string detaches it.
		cfg["sla_id"] = types.StringValue("")
		assertBodies(t, h.apply(h.config(cfg)), `{"wan_sla":""}`)
	})

	t.Run("a WAN that already has a profile imports without a plan", func(t *testing.T) {
		doc := strings.Replace(wanTestDocument, `"report_wan_event": false,`,
			`"report_wan_event": false, "wan_sla": "00000000-0000-4000-8000-000000000001",`, 1)
		h := newWANHarnessFrom(t, doc)
		var got types.String
		h.state.GetAttribute(h.ctx, path.Root("sla_id"), &got)
		if got.ValueString() != "00000000-0000-4000-8000-000000000001" {
			t.Fatalf("imported sla_id = %v", got)
		}
		h.assertEqual("plan after import", h.state.Raw, h.plan(h.config(wanBaseConfig())))
	})
}

func Test_wanSlaModelToAPI(t *testing.T) {
	m := &wanSlaResourceModel{
		Name:                 types.StringValue("wan-standard"),
		MonitorHealthType:    types.StringValue("MAJORITY"),
		MonitorDownThreshold: types.Int64Value(2), // ignored: only CUSTOM sends it
		Monitors: []wanSlaMonitorModel{
			{
				Type: types.StringValue(
					"ICMP",
				),
				Target: types.StringValue("9.9.9.9"),
				Domain: types.StringNull(),
				Interval: types.Int64Value(
					2,
				),
				TimePeriod:      types.Int64Value(30),
				ThresholdPolicy: types.StringValue("any"),
				LossThreshold: types.Int64Value(
					5,
				),
				LatencyThreshold: types.Int64Null(),
				JitterThreshold:  types.Int64Null(),
			},
			{
				Type: types.StringValue(
					"DNS",
				),
				Target: types.StringValue("8.8.8.8"),
				Domain: types.StringValue("example.com"),
				Interval: types.Int64Value(
					5,
				),
				TimePeriod:      types.Int64Value(30),
				ThresholdPolicy: types.StringValue("any"),
				LossThreshold: types.Int64Value(
					5,
				),
				LatencyThreshold: types.Int64Null(),
				JitterThreshold:  types.Int64Null(),
			},
			{
				Type: types.StringValue(
					"ICMP",
				),
				Target: types.StringValue("1.1.1.1"),
				Domain: types.StringNull(),
				Interval: types.Int64Value(
					2,
				),
				TimePeriod:      types.Int64Value(30),
				ThresholdPolicy: types.StringValue("loss-or-all-others"),
				LossThreshold: types.Int64Value(
					5,
				),
				LatencyThreshold: types.Int64Value(150),
				JitterThreshold:  types.Int64Value(30),
			},
		},
	}
	// The controller has 1.1.1.1 as monitor 1 and 9.9.9.9 as monitor 2; the config lists them in another order.
	current := []unifi.WANSlaMonitor{
		{MonitorID: i64(1), Type: "ICMP", Target: "1.1.1.1"},
		{MonitorID: i64(2), Type: "ICMP", Target: "9.9.9.9"},
	}
	out := wanSlaModelToAPI(m, current)
	if out.MonitorDownThreshold != nil {
		t.Errorf("monitor_down_threshold sent without CUSTOM: %v", *out.MonitorDownThreshold)
	}
	if *out.Monitors[0].MonitorID != 2 || out.Monitors[1].MonitorID != nil ||
		*out.Monitors[2].MonitorID != 1 {
		t.Errorf("ids should follow type+target, not position: %v %v %v",
			out.Monitors[0].MonitorID, out.Monitors[1].MonitorID, out.Monitors[2].MonitorID)
	}
	if out.Monitors[1].Domain != "example.com" || out.Monitors[0].Domain != "" {
		t.Errorf("domain only on DNS: %+v", out.Monitors)
	}
	if out.Monitors[0].Alert.LatencyThreshold != nil ||
		*out.Monitors[2].Alert.JitterThreshold != 30 {
		t.Errorf("unset thresholds stay unset, set ones are sent: %+v", out.Monitors)
	}

	m.MonitorHealthType = types.StringValue("CUSTOM")
	if got := wanSlaModelToAPI(m, nil).MonitorDownThreshold; got == nil || *got != 2 {
		t.Errorf("CUSTOM sends monitor_down_threshold, got %v", got)
	}
}

func Test_wanSlaAPIToModel(t *testing.T) {
	var m wanSlaResourceModel
	wanSlaAPIToModel(&unifi.WANSla{
		ID: "x", Name: "n", MonitorHealthType: "MAJORITY", MonitorDownThreshold: i64(1),
		Monitors: []unifi.WANSlaMonitor{{
			MonitorID: i64(1), Type: "ICMP", Target: "1.1.1.1", Interval: 2, TimePeriod: 30,
			Alert: unifi.WANSlaAlert{ThresholdPolicy: "any", LossThreshold: i64(5)},
		}},
	}, &m)
	if !m.MonitorDownThreshold.IsNull() {
		t.Errorf("a stored threshold without CUSTOM reads as null, got %v", m.MonitorDownThreshold)
	}
	mon := m.Monitors[0]
	if !mon.Domain.IsNull() || !mon.LatencyThreshold.IsNull() ||
		mon.LossThreshold.ValueInt64() != 5 ||
		mon.MonitorID.ValueInt64() != 1 {
		t.Errorf("read %+v", mon)
	}
}

func Test_wanSlaResource_ValidateConfig(t *testing.T) {
	r := &wanSlaResource{}
	ctx := context.Background()
	var sresp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sresp)
	sch := sresp.Schema
	listType, ok := sch.Attributes["monitors"].GetType().(types.ListType)
	if !ok {
		t.Fatalf("monitors is %T, want a list", sch.Attributes["monitors"].GetType())
	}
	monType, ok := listType.ElemType.(types.ObjectType)
	if !ok {
		t.Fatalf("monitors elements are %T, want objects", listType.ElemType)
	}
	mon := func(typ, domain string, loss attr.Value) attr.Value {
		d := types.StringNull()
		if domain != "" {
			d = types.StringValue(domain)
		}
		return types.ObjectValueMust(monType.AttrTypes, map[string]attr.Value{
			"monitor_id":        types.Int64Unknown(),
			"type":              types.StringValue(typ),
			"target":            types.StringValue("1.1.1.1"),
			"domain":            d,
			"interval":          types.Int64Value(2),
			"time_period":       types.Int64Value(30),
			"threshold_policy":  types.StringValue("any"),
			"loss_threshold":    loss,
			"latency_threshold": types.Int64Null(),
			"jitter_threshold":  types.Int64Null(),
		})
	}
	cases := map[string]struct {
		health string
		down   attr.Value
		mons   []attr.Value
		errors int
	}{
		"valid": {
			"MAJORITY",
			types.Int64Null(),
			[]attr.Value{
				mon("ICMP", "", types.Int64Value(5)),
				mon("DNS", "ui.com", types.Int64Value(5)),
			},
			0,
		},
		"DNS without domain": {
			"ANY",
			types.Int64Null(),
			[]attr.Value{mon("DNS", "", types.Int64Value(5))},
			1,
		},
		"ICMP with domain": {
			"ANY",
			types.Int64Null(),
			[]attr.Value{mon("ICMP", "ui.com", types.Int64Value(5))},
			1,
		},
		"no threshold": {
			"ANY",
			types.Int64Null(),
			[]attr.Value{mon("ICMP", "", types.Int64Null())},
			1,
		},
		"CUSTOM without its count": {
			"CUSTOM",
			types.Int64Null(),
			[]attr.Value{mon("ICMP", "", types.Int64Value(5))},
			1,
		},
		"a count without CUSTOM": {
			"ALL",
			types.Int64Value(1),
			[]attr.Value{mon("ICMP", "", types.Int64Value(5))},
			1,
		},
		"CUSTOM with its count": {
			"CUSTOM",
			types.Int64Value(1),
			[]attr.Value{mon("ICMP", "", types.Int64Value(5))},
			0,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			obj := map[string]attr.Value{}
			obj["name"] = types.StringValue("n")
			obj["monitor_health_type"] = types.StringValue(c.health)
			obj["monitor_down_threshold"] = c.down
			obj["monitors"] = types.ListValueMust(monType, c.mons)
			objType, ok := sch.Type().TerraformType(ctx).(tftypes.Object)
			if !ok {
				t.Fatal("schema type is not an object")
			}
			vals := map[string]tftypes.Value{}
			for name, typ := range objType.AttributeTypes {
				vals[name] = tftypes.NewValue(typ, nil)
			}
			for name, v := range obj {
				tv, err := v.ToTerraformValue(ctx)
				if err != nil {
					t.Fatal(err)
				}
				vals[name] = tv
			}
			cfg := tfsdk.Config{Schema: sch, Raw: tftypes.NewValue(objType, vals)}
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, &resp)
			if got := resp.Diagnostics.ErrorsCount(); got != c.errors {
				t.Errorf("errors = %d, want %d: %v", got, c.errors, resp.Diagnostics)
			}
		})
	}
}
