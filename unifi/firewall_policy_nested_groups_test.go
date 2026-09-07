package unifi

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// TestFirewallPolicyUpgradeState_nestsPrefixedGroups guards the v1 -> v2
// schema upgrade: icmp_typename/icmp_v6_typename move under icmp, and
// schedule.time_all_day/time_range_start/time_range_end move under
// schedule.time (with the boundaries one level deeper, under range). The v0
// upgrader applies the same rewrite after converting the integer port, so
// both prior versions are fed through.
func TestFirewallPolicyUpgradeState_nestsPrefixedGroups(t *testing.T) {
	ctx := context.Background()
	r := &firewallPolicyResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	const prior = `{
		"id": "pol-1", "site": "default", "name": "scheduled", "action": "ALLOW",
		"enabled": true, "protocol": "tcp", "description": "", "logging": false,
		"index": 10001, "create_allow_respond": false, "ip_version": "IPV4",
		"connection_state_type": "ALL", "connection_states": [],
		"icmp_typename": "ANY", "icmp_v6_typename": "ECHO_REQUEST",
		"schedule": %s,
		"source": {
			"zone_id": "z1", "matching_target": "ANY", "network_ids": [], "client_macs": [],
			"ips": [], "web_domains": [], "port": %s, "port_group_id": "", "ip_group_id": "",
			"port_matching_type": "ANY", "matching_target_type": "ANY"
		},
		"destination": {
			"zone_id": "z2", "matching_target": "ANY", "network_ids": [], "client_macs": [],
			"ips": [], "web_domains": [], "port": %s, "port_group_id": "", "ip_group_id": "",
			"port_matching_type": "SPECIFIC", "matching_target_type": "ANY"
		},
		"timeouts": null
	}`
	const timedSchedule = `{
		"date": null, "date_start": null, "date_end": null, "mode": "EVERY_WEEK",
		"normalize": false, "repeat_on_days": ["mon", "wed"],
		"time_all_day": false, "time_range_start": "09:00", "time_range_end": "17:30"
	}`

	tests := map[string]struct {
		version          int64
		schedule         string
		srcPort, dstPort string
		nullSchedule     bool
	}{
		"v0 integer port": {version: 0, schedule: timedSchedule, srcPort: "0", dstPort: "443"},
		"v1 string port":  {version: 1, schedule: timedSchedule, srcPort: "null", dstPort: `"443"`},
		"v1 null schedule": {
			version: 1, schedule: "null", srcPort: "null", dstPort: `"443"`, nullSchedule: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			up, ok := r.UpgradeState(ctx)[tc.version]
			if !ok {
				t.Fatalf("no upgrader registered for schema version %d", tc.version)
			}
			resp := &fwresource.UpgradeStateResponse{}
			up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
				RawState: &tfprotov6.RawState{
					JSON: fmt.Appendf(nil, prior, tc.schedule, tc.srcPort, tc.dstPort),
				},
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
			obj := func(v tftypes.Value, name string) map[string]tftypes.Value {
				t.Helper()
				var m map[string]tftypes.Value
				if err := v.As(&m); err != nil {
					t.Fatalf("%s: as object: %v (value %v)", name, err, v)
				}
				return m
			}
			str := func(v tftypes.Value, name, want string) {
				t.Helper()
				var s string
				if err := v.As(&s); err != nil || s != want {
					t.Errorf("%s = %v (%v), want %q", name, v, err, want)
				}
			}
			boolean := func(v tftypes.Value, name string, want bool) {
				t.Helper()
				var b bool
				if err := v.As(&b); err != nil || b != want {
					t.Errorf("%s = %v (%v), want %v", name, v, err, want)
				}
			}

			str(root["name"], "name", "scheduled")
			for _, flat := range []string{"icmp_typename", "icmp_v6_typename"} {
				if _, exists := root[flat]; exists {
					t.Errorf("flat attribute %q survived the upgrade", flat)
				}
			}
			icmp := obj(root["icmp"], "icmp")
			str(icmp["typename"], "icmp.typename", "ANY")
			str(icmp["v6_typename"], "icmp.v6_typename", "ECHO_REQUEST")

			// v0's integer port becomes a string; 0 means "no port".
			src := obj(root["source"], "source")
			if !src["port"].IsNull() {
				t.Errorf("source.port = %v, want null", src["port"])
			}
			dst := obj(root["destination"], "destination")
			str(dst["port"], "destination.port", "443")

			if tc.nullSchedule {
				if !root["schedule"].IsNull() {
					t.Errorf("schedule = %v, want null", root["schedule"])
				}
				return
			}
			schedule := obj(root["schedule"], "schedule")
			for _, flat := range []string{"time_all_day", "time_range_start", "time_range_end"} {
				if _, exists := schedule[flat]; exists {
					t.Errorf("flat attribute schedule.%q survived the upgrade", flat)
				}
			}
			str(schedule["mode"], "schedule.mode", "EVERY_WEEK")
			if !schedule["date"].IsNull() {
				t.Errorf("schedule.date = %v, want null", schedule["date"])
			}
			tm := obj(schedule["time"], "schedule.time")
			boolean(tm["all_day"], "schedule.time.all_day", false)
			rng := obj(tm["range"], "schedule.time.range")
			str(rng["start"], "schedule.time.range.start", "09:00")
			str(rng["end"], "schedule.time.range.end", "17:30")
		})
	}
}

// TestFirewallPolicyNestedGroups_roundTrip checks that the icmp and
// schedule.time groups convert API -> model -> API without loss, and that a
// null icmp object contributes nothing on the wire (as the unset flat
// attributes did).
func TestFirewallPolicyNestedGroups_roundTrip(t *testing.T) {
	ctx := context.Background()

	allDay := false
	api := testScheduledFirewallPolicy(&unifi.FirewallPolicySchedule{
		Mode:           "EVERY_DAY",
		RepeatOnDays:   []string{},
		TimeAllDay:     &allDay,
		TimeRangeStart: "08:15",
		TimeRangeEnd:   "18:45",
	})
	api.ICMPTypename = "ANY"
	api.ICMPV6Typename = "ECHO_REQUEST"

	var model firewallPolicyModel
	if diags := firewallPolicyToModel(ctx, api, &model); diags.HasError() {
		t.Fatalf("firewallPolicyToModel: %v", diags)
	}

	icmp := model.ICMP.Attributes()
	if got := attrAs[types.String](t, icmp["typename"]).ValueString(); got != "ANY" {
		t.Errorf("icmp.typename = %q, want ANY", got)
	}
	if got := attrAs[types.String](t, icmp["v6_typename"]).ValueString(); got != "ECHO_REQUEST" {
		t.Errorf("icmp.v6_typename = %q, want ECHO_REQUEST", got)
	}

	var schedule firewallPolicyScheduleModel
	if diags := model.Schedule.As(ctx, &schedule, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("reading schedule: %v", diags)
	}
	gotAllDay, start, end := firewallPolicyScheduleTimeFields(schedule.Time)
	if gotAllDay.IsNull() || gotAllDay.ValueBool() ||
		start.ValueString() != "08:15" || end.ValueString() != "18:45" {
		t.Errorf("schedule.time = %v, want all_day=false range 08:15-18:45", schedule.Time)
	}

	got, diags := modelToFirewallPolicy(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallPolicy: %v", diags)
	}
	if got.ICMPTypename != "ANY" || got.ICMPV6Typename != "ECHO_REQUEST" {
		t.Errorf(
			"icmp typenames = %q/%q, want ANY/ECHO_REQUEST",
			got.ICMPTypename, got.ICMPV6Typename,
		)
	}
	if !reflect.DeepEqual(got.Schedule, api.Schedule) {
		t.Errorf(
			"schedule changed during round-trip:\n got: %#v\nwant: %#v",
			got.Schedule,
			api.Schedule,
		)
	}

	// A null icmp object contributes nothing, matching the unset flat leaves.
	model.ICMP = types.ObjectNull(firewallPolicyICMPModel{}.AttributeTypes())
	got, diags = modelToFirewallPolicy(ctx, model)
	if diags.HasError() {
		t.Fatalf("modelToFirewallPolicy with null icmp: %v", diags)
	}
	if got.ICMPTypename != "" || got.ICMPV6Typename != "" {
		t.Errorf("null icmp sent %q/%q, want empty", got.ICMPTypename, got.ICMPV6Typename)
	}
}
