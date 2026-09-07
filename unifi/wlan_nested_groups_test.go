package unifi

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// wlanUpgradedRoot runs prior JSON state through the WLAN upgrader registered
// for fromVersion and decodes the result against the current schema.
func wlanUpgradedRoot(t *testing.T, fromVersion int64, prior []byte) map[string]tftypes.Value {
	t.Helper()
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	if schemaResp.Schema.Version != 2 {
		t.Fatalf("wlan schema Version = %d, want 2", schemaResp.Schema.Version)
	}
	schemaType := schemaResp.Schema.Type().TerraformType(ctx)

	up, ok := r.UpgradeState(ctx)[fromVersion]
	if !ok {
		t.Fatalf("no upgrader registered for schema version %d", fromVersion)
	}
	resp := &fwresource.UpgradeStateResponse{}
	up.StateUpgrader(ctx, fwresource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: prior},
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
	return root
}

type tfValueAsserts struct {
	t *testing.T
}

func (a tfValueAsserts) obj(v tftypes.Value, name string) map[string]tftypes.Value {
	a.t.Helper()
	var m map[string]tftypes.Value
	if err := v.As(&m); err != nil {
		a.t.Fatalf("%s: as object: %v (value %v)", name, err, v)
	}
	return m
}

func (a tfValueAsserts) str(v tftypes.Value, name, want string) {
	a.t.Helper()
	var s string
	if err := v.As(&s); err != nil || s != want {
		a.t.Errorf("%s = %v (%v), want %q", name, v, err, want)
	}
}

func (a tfValueAsserts) num(v tftypes.Value, name string, want int64) {
	a.t.Helper()
	var f big.Float
	if err := v.As(&f); err != nil {
		a.t.Errorf("%s = %v (%v), want %d", name, v, err, want)
		return
	}
	if n, _ := f.Int64(); n != want {
		a.t.Errorf("%s = %d, want %d", name, n, want)
	}
}

func (a tfValueAsserts) boolean(v tftypes.Value, name string, want bool) {
	a.t.Helper()
	var b bool
	if err := v.As(&b); err != nil || b != want {
		a.t.Errorf("%s = %v (%v), want %v", name, v, err, want)
	}
}

// TestWLANUpgradeState_v1NestsPrefixedGroups guards the v1 -> v2 schema
// upgrade: flat wpa_*/wpa3_*/radius_*/ap_group_* attributes and
// schedule[].start_* move into nested objects, and every other attribute
// passes through.
func TestWLANUpgradeState_v1NestsPrefixedGroups(t *testing.T) {
	prior := []byte(`{
		"id": "wlan-1", "site": "default", "name": "corp", "security": "wpaeap",
		"user_group_id": "ug-1", "pmf_mode": "optional", "enabled": true,
		"wpa_mode": "auto", "wpa_enc": "gcmp",
		"wpa3_support": true, "wpa3_transition": true,
		"wpa3_fast_roaming": false, "wpa3_enhanced_192": true,
		"radius_profile_id": "rp-1", "radius_mac_auth_enabled": true,
		"ap_group_ids": ["apg-1", "apg-2"], "ap_group_mode": "groups",
		"dtim_mode": "custom", "dtim_ng": 2, "dtim_6e": 3,
		"minimum_data_rate_2g_kbps": 1000,
		"schedule": [
			{"day_of_week": "mon", "start_hour": 8, "start_minute": 30, "duration": "2h0m0s", "name": "am"},
			{"day_of_week": "tue", "start_hour": 22, "start_minute": null, "duration": "30m0s", "name": null}
		]
	}`)

	root := wlanUpgradedRoot(t, 1, prior)
	a := tfValueAsserts{t}

	a.str(root["name"], "name", "corp")
	a.str(root["dtim_mode"], "dtim_mode", "custom")
	a.num(root["dtim_6e"], "dtim_6e", 3)
	a.num(root["minimum_data_rate_2g_kbps"], "minimum_data_rate_2g_kbps", 1000)
	for _, flat := range []string{
		"wpa_mode", "wpa_enc", "wpa3_support", "wpa3_transition", "wpa3_fast_roaming",
		"wpa3_enhanced_192", "radius_profile_id", "radius_mac_auth_enabled",
		"ap_group_ids", "ap_group_mode",
	} {
		if _, exists := root[flat]; exists {
			t.Errorf("flat attribute %q survived the upgrade", flat)
		}
	}

	wpa := a.obj(root["wpa"], "wpa")
	a.str(wpa["mode"], "wpa.mode", "auto")
	a.str(wpa["enc"], "wpa.enc", "gcmp")

	wpa3 := a.obj(root["wpa3"], "wpa3")
	a.boolean(wpa3["support"], "wpa3.support", true)
	a.boolean(wpa3["transition"], "wpa3.transition", true)
	a.boolean(wpa3["fast_roaming"], "wpa3.fast_roaming", false)
	a.boolean(wpa3["enhanced_192"], "wpa3.enhanced_192", true)

	radius := a.obj(root["radius"], "radius")
	a.str(radius["profile_id"], "radius.profile_id", "rp-1")
	a.boolean(radius["mac_auth_enabled"], "radius.mac_auth_enabled", true)

	apGroup := a.obj(root["ap_group"], "ap_group")
	a.str(apGroup["mode"], "ap_group.mode", "groups")
	var ids []tftypes.Value
	if err := apGroup["ids"].As(&ids); err != nil || len(ids) != 2 {
		t.Fatalf("ap_group.ids = %v (%v), want two elements", apGroup["ids"], err)
	}
	a.str(ids[0], "ap_group.ids[0]", "apg-1")

	var scheds []tftypes.Value
	if err := root["schedule"].As(&scheds); err != nil || len(scheds) != 2 {
		t.Fatalf("schedule = %v (%v), want two elements", root["schedule"], err)
	}
	s0 := a.obj(scheds[0], "schedule[0]")
	for _, flat := range []string{"start_hour", "start_minute"} {
		if _, exists := s0[flat]; exists {
			t.Errorf("flat schedule attribute %q survived the upgrade", flat)
		}
	}
	a.str(s0["day_of_week"], "schedule[0].day_of_week", "mon")
	a.str(s0["duration"], "schedule[0].duration", "2h0m0s")
	start0 := a.obj(s0["start"], "schedule[0].start")
	a.num(start0["hour"], "schedule[0].start.hour", 8)
	a.num(start0["minute"], "schedule[0].start.minute", 30)

	s1 := a.obj(scheds[1], "schedule[1]")
	start1 := a.obj(s1["start"], "schedule[1].start")
	a.num(start1["hour"], "schedule[1].start.hour", 22)
	if !start1["minute"].IsNull() {
		t.Errorf("schedule[1].start.minute = %v, want null", start1["minute"])
	}
}

// TestWLANUpgradeState_v0NestsAndConvertsDuration guards that the v0 upgrader
// still converts schedule[].duration from integer minutes and also applies
// the v2 nesting, since every upgrader targets the current schema.
func TestWLANUpgradeState_v0NestsAndConvertsDuration(t *testing.T) {
	prior := []byte(`{
		"id": "wlan-0", "site": "default", "name": "legacy", "security": "wpapsk",
		"wpa3_support": false, "wpa3_transition": false,
		"ap_group_mode": "all",
		"schedule": [
			{"day_of_week": "sat", "start_hour": 9, "start_minute": 15, "duration": 90, "name": "wknd"}
		]
	}`)

	root := wlanUpgradedRoot(t, 0, prior)
	a := tfValueAsserts{t}

	wpa3 := a.obj(root["wpa3"], "wpa3")
	a.boolean(wpa3["support"], "wpa3.support", false)
	// Leaves the v0 provider never wrote are filled with null by reconciliation.
	if !wpa3["fast_roaming"].IsNull() {
		t.Errorf("wpa3.fast_roaming = %v, want null", wpa3["fast_roaming"])
	}
	apGroup := a.obj(root["ap_group"], "ap_group")
	a.str(apGroup["mode"], "ap_group.mode", "all")
	if !apGroup["ids"].IsNull() {
		t.Errorf("ap_group.ids = %v, want null", apGroup["ids"])
	}
	// A group with none of its flat keys present stays null.
	if !root["wpa"].IsNull() {
		t.Errorf("wpa = %v, want null when no wpa_* key existed", root["wpa"])
	}
	if !root["radius"].IsNull() {
		t.Errorf("radius = %v, want null when no radius_* key existed", root["radius"])
	}

	var scheds []tftypes.Value
	if err := root["schedule"].As(&scheds); err != nil || len(scheds) != 1 {
		t.Fatalf("schedule = %v (%v), want one element", root["schedule"], err)
	}
	s0 := a.obj(scheds[0], "schedule[0]")
	a.str(s0["duration"], "schedule[0].duration", "1h30m0s")
	start := a.obj(s0["start"], "schedule[0].start")
	a.num(start["hour"], "schedule[0].start.hour", 9)
	a.num(start["minute"], "schedule[0].start.minute", 15)
}

// TestWLANNestedGroups_roundTrip covers the model <-> API mapping of the
// nested wpa/wpa3/radius/ap_group groups and schedule[].start: values travel
// to the go-unifi struct, come back from a read, and an unknown planned group
// keeps the controller's values on update.
func TestWLANNestedGroups_roundTrip(t *testing.T) {
	ctx := context.Background()
	r := &wlanFrameworkResource{}

	apGroupIDs, d := types.SetValueFrom(ctx, types.StringType, []string{"apg-1"})
	if d.HasError() {
		t.Fatalf("building ap group set: %v", d)
	}
	start, d := types.ObjectValueFrom(ctx, wlanScheduleStartAttrTypes(), wlanScheduleStartModel{
		Hour:   types.Int64Value(7),
		Minute: types.Int64Value(45),
	})
	if d.HasError() {
		t.Fatalf("building schedule start: %v", d)
	}
	schedule, d := types.ListValueFrom(ctx,
		types.ObjectType{AttrTypes: wlanScheduleAttrTypes()},
		[]wlanScheduleModel{{
			DayOfWeek: types.StringValue("wed"),
			Start:     start,
			Duration:  timetypes.NewGoDurationValue(90 * time.Minute),
			Name:      types.StringValue("mid"),
		}})
	if d.HasError() {
		t.Fatalf("building schedule list: %v", d)
	}

	plan := wlanFrameworkResourceModel{
		Name:     types.StringValue("nested"),
		Security: types.StringValue("wpaeap"),
		WPA: types.ObjectValueMust(wlanWPAAttrTypes(), map[string]attr.Value{
			"mode": types.StringValue("wpa1"),
			"enc":  types.StringValue("gcmp-256"),
		}),
		WPA3: types.ObjectValueMust(wlanWPA3AttrTypes(), map[string]attr.Value{
			"support":      types.BoolValue(true),
			"transition":   types.BoolValue(false),
			"fast_roaming": types.BoolValue(true),
			"enhanced_192": types.BoolValue(true),
		}),
		Radius: types.ObjectValueMust(wlanRadiusAttrTypes(), map[string]attr.Value{
			"profile_id":       types.StringValue("rp-9"),
			"mac_auth_enabled": types.BoolValue(true),
		}),
		ApGroup: types.ObjectValueMust(wlanApGroupAttrTypes(), map[string]attr.Value{
			"ids":  apGroupIDs,
			"mode": types.StringValue("groups"),
		}),
		Schedule: schedule,
	}

	// plan -> API
	api, diags := r.planToWLAN(ctx, plan)
	if diags.HasError() {
		t.Fatalf("planToWLAN: %v", diags)
	}
	if api.WPAMode != "wpa1" || api.WPAEnc != "gcmp-256" {
		t.Errorf("wpa: %q %q", api.WPAMode, api.WPAEnc)
	}
	if !api.WPA3Support || api.WPA3Transition || !api.WPA3FastRoaming || !api.WPA3Enhanced192 {
		t.Errorf("wpa3: %v %v %v %v",
			api.WPA3Support, api.WPA3Transition, api.WPA3FastRoaming, api.WPA3Enhanced192)
	}
	if api.RADIUSProfileID != "rp-9" || !api.RADIUSMACAuthEnabled {
		t.Errorf("radius: %q %v", api.RADIUSProfileID, api.RADIUSMACAuthEnabled)
	}
	if api.ApGroupMode != "groups" || len(api.ApGroupIDs) != 1 || api.ApGroupIDs[0] != "apg-1" {
		t.Errorf("ap_group: %q %v", api.ApGroupMode, api.ApGroupIDs)
	}
	if len(api.ScheduleWithDuration) != 1 {
		t.Fatalf("schedule: %+v, want one entry", api.ScheduleWithDuration)
	}
	sched := api.ScheduleWithDuration[0]
	if sched.StartHour == nil || *sched.StartHour != 7 ||
		sched.StartMinute == nil || *sched.StartMinute != 45 ||
		sched.DurationMinutes == nil || *sched.DurationMinutes != 90 {
		t.Errorf("schedule[0]: %+v", sched)
	}

	// API -> model: every group is rebuilt from the response.
	var back wlanFrameworkResourceModel
	if d := r.wlanToModel(ctx, api, &back, "default"); d.HasError() {
		t.Fatalf("wlanToModel: %v", d)
	}
	if !back.WPA.Equal(plan.WPA) {
		t.Errorf("wpa read back = %v, want %v", back.WPA, plan.WPA)
	}
	if !back.WPA3.Equal(plan.WPA3) {
		t.Errorf("wpa3 read back = %v, want %v", back.WPA3, plan.WPA3)
	}
	if !back.Radius.Equal(plan.Radius) {
		t.Errorf("radius read back = %v, want %v", back.Radius, plan.Radius)
	}
	if !back.ApGroup.Equal(plan.ApGroup) {
		t.Errorf("ap_group read back = %v, want %v", back.ApGroup, plan.ApGroup)
	}
	if !back.Schedule.Equal(plan.Schedule) {
		t.Errorf("schedule read back = %v, want %v", back.Schedule, plan.Schedule)
	}

	// Null/unknown groups contribute nothing on the write side.
	empty, diags := r.planToWLAN(ctx, wlanFrameworkResourceModel{
		Name:    types.StringValue("bare"),
		WPA:     types.ObjectNull(wlanWPAAttrTypes()),
		WPA3:    types.ObjectUnknown(wlanWPA3AttrTypes()),
		Radius:  types.ObjectNull(wlanRadiusAttrTypes()),
		ApGroup: types.ObjectUnknown(wlanApGroupAttrTypes()),
	})
	if diags.HasError() {
		t.Fatalf("planToWLAN (bare): %v", diags)
	}
	if empty.WPAMode != "" || empty.WPA3Support || empty.RADIUSProfileID != "" ||
		empty.ApGroupMode != "" || len(empty.ApGroupIDs) != 0 {
		t.Errorf("unset groups leaked into the payload: %+v", empty)
	}

	// A fresh read with nothing set falls back to the old flat defaults.
	var fresh wlanFrameworkResourceModel
	if d := r.wlanToModel(ctx, &unifi.WLAN{ID: "x"}, &fresh, "default"); d.HasError() {
		t.Fatalf("wlanToModel (fresh): %v", d)
	}
	if !fresh.WPA.Equal(wlanWPADefault()) {
		t.Errorf("fresh wpa = %v, want %v", fresh.WPA, wlanWPADefault())
	}
	if !fresh.WPA3.Equal(wlanWPA3Default()) {
		t.Errorf("fresh wpa3 = %v, want %v", fresh.WPA3, wlanWPA3Default())
	}
	freshRadius := types.ObjectValueMust(wlanRadiusAttrTypes(), map[string]attr.Value{
		"profile_id":       types.StringNull(),
		"mac_auth_enabled": types.BoolValue(false),
	})
	if !fresh.Radius.Equal(freshRadius) {
		t.Errorf("fresh radius = %v, want %v", fresh.Radius, freshRadius)
	}
	if !fresh.ApGroup.Equal(wlanApGroupDefault()) {
		t.Errorf("fresh ap_group = %v, want %v", fresh.ApGroup, wlanApGroupDefault())
	}
	if !fresh.Schedule.IsNull() {
		t.Errorf("fresh schedule = %v, want null", fresh.Schedule)
	}

	// On update an unknown planned group keeps the controller's values, and a
	// partially-known group only re-asserts the leaves the plan knows.
	update := &wlanFrameworkResourceModel{
		WPA3: types.ObjectUnknown(wlanWPA3AttrTypes()),
		Radius: types.ObjectValueMust(wlanRadiusAttrTypes(), map[string]attr.Value{
			"profile_id":       types.StringUnknown(),
			"mac_auth_enabled": types.BoolValue(false),
		}),
	}
	state := back
	r.applyPlanToState(ctx, update, &state)
	if !state.WPA3.Equal(back.WPA3) {
		t.Errorf("unknown planned wpa3 must keep controller value: %v", state.WPA3)
	}
	radius := state.Radius.Attributes()
	if attrAs[types.String](t, radius["profile_id"]).ValueString() != "rp-9" {
		t.Errorf("unknown radius.profile_id must keep controller value: %v", radius["profile_id"])
	}
	if attrAs[types.Bool](t, radius["mac_auth_enabled"]).ValueBool() {
		t.Errorf("planned radius.mac_auth_enabled not applied: %v", radius["mac_auth_enabled"])
	}
}
