package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	api "github.com/ubiquiti-community/go-unifi/unifi"
)

func TestFirewallPolicySchemaExposesSettableSchedule(t *testing.T) {
	var response resource.SchemaResponse
	NewFirewallPolicyResource().Schema(
		context.Background(), resource.SchemaRequest{}, &response,
	)
	if response.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %v", response.Diagnostics)
	}

	attribute, ok := response.Schema.Attributes["schedule"]
	if !ok {
		t.Fatal("firewall policy schema has no schedule attribute")
	}
	schedule, ok := attribute.(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("schedule schema type = %T, want schema.SingleNestedAttribute", attribute)
	}
	if !schedule.Optional || !schedule.Computed {
		t.Fatalf(
			"schedule Optional=%v Computed=%v, want both true",
			schedule.Optional,
			schedule.Computed,
		)
	}
	if len(schedule.Validators) == 0 {
		t.Fatal("schedule has no cross-field validator")
	}
	for name, attribute := range schedule.Attributes {
		switch field := attribute.(type) {
		case schema.StringAttribute:
			if !field.Optional || !field.Computed {
				t.Errorf("%s is not optional and computed", name)
			}
		case schema.BoolAttribute:
			if !field.Optional || !field.Computed {
				t.Errorf("%s is not optional and computed", name)
			}
		case schema.SetAttribute:
			if !field.Optional || !field.Computed {
				t.Errorf("%s is not optional and computed", name)
			}
		default:
			t.Errorf("unexpected schedule attribute type for %s: %T", name, attribute)
		}
	}
}

func TestFirewallPolicyScheduleValidation(t *testing.T) {
	tests := map[string]struct {
		schedule  firewallPolicyScheduleModel
		wantError bool
	}{
		"always":        {schedule: scheduleForTest("ALWAYS")},
		"daily all day": {schedule: withScheduleTime(scheduleForTest("EVERY_DAY"), true, "", "")},
		"weekly timed": {
			schedule: withScheduleDays(
				withScheduleTime(scheduleForTest("EVERY_WEEK"), false, "09:00", "17:00"),
				"mon",
			),
		},
		"one time": {
			schedule: withScheduleDate(
				withScheduleTime(scheduleForTest("ONE_TIME_ONLY"), false, "09:00", "17:00"),
				"2026-08-04",
			),
		},
		"custom": {
			schedule: withScheduleRange(
				withScheduleDays(withScheduleTime(scheduleForTest("CUSTOM"), true, "", ""), "wed"),
				"2026-08-01",
				"2026-08-31",
			),
		},
		"missing mode": {schedule: scheduleForTest(""), wantError: true},
		"weekly without days": {
			schedule: withScheduleTime(
				scheduleForTest("EVERY_WEEK"),
				true,
				"",
				"",
			),
			wantError: true,
		},
		"timed without end": {
			schedule: withScheduleTime(
				scheduleForTest("EVERY_DAY"),
				false,
				"09:00",
				"",
			),
			wantError: true,
		},
		"one time all day": {
			schedule: withScheduleDate(
				withScheduleTime(scheduleForTest("ONE_TIME_ONLY"), true, "", ""),
				"2026-08-04",
			),
			wantError: true,
		},
		"invalid calendar date": {
			schedule: withScheduleDate(
				withScheduleTime(scheduleForTest("ONE_TIME_ONLY"), false, "09:00", "17:00"),
				"2026-02-30",
			),
			wantError: true,
		},
		"reversed custom range": {
			schedule: withScheduleRange(
				withScheduleDays(withScheduleTime(scheduleForTest("CUSTOM"), true, "", ""), "fri"),
				"2026-08-31",
				"2026-08-01",
			),
			wantError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			value, diags := types.ObjectValueFrom(
				context.Background(),
				tc.schedule.AttributeTypes(),
				tc.schedule,
			)
			if diags.HasError() {
				t.Fatalf("building schedule: %v", diags)
			}
			resp := &validator.ObjectResponse{}
			firewallPolicyScheduleValidator{}.ValidateObject(
				context.Background(),
				validator.ObjectRequest{
					Path: path.Root("schedule"), ConfigValue: value,
				},
				resp,
			)
			if resp.Diagnostics.HasError() != tc.wantError {
				t.Fatalf(
					"validation error=%v, want %v: %v",
					resp.Diagnostics.HasError(),
					tc.wantError,
					resp.Diagnostics,
				)
			}
		})
	}
}

func TestFirewallPolicyScheduleNormalization(t *testing.T) {
	tests := map[string]struct{ date, rangeDates, days, time bool }{
		"ALWAYS":        {},
		"EVERY_DAY":     {time: true},
		"EVERY_WEEK":    {days: true, time: true},
		"ONE_TIME_ONLY": {date: true, time: true},
		"CUSTOM":        {rangeDates: true, days: true, time: true},
	}
	for mode, keep := range tests {
		t.Run(mode+" model", func(t *testing.T) {
			model := withScheduleRange(
				withScheduleDate(
					withScheduleDays(
						withScheduleTime(scheduleForTest(mode), false, "09:00", "17:00"),
						"mon",
					),
					"2026-08-04",
				),
				"2026-08-01",
				"2026-08-31",
			)
			model.Normalize = types.BoolValue(true)
			if !normalizeFirewallPolicyScheduleModel(&model) ||
				(!model.Date.IsNull()) != keep.date ||
				(!model.DateStart.IsNull() || !model.DateEnd.IsNull()) != keep.rangeDates ||
				(len(model.RepeatOnDays.Elements()) > 0) != keep.days ||
				(!model.TimeAllDay.IsNull() || !model.TimeRangeStart.IsNull() || !model.TimeRangeEnd.IsNull()) != keep.time {
				t.Fatalf("unexpected normalized Terraform schedule: %#v", model)
			}
		})
	}
	model := withScheduleTime(scheduleForTest("EVERY_DAY"), true, "09:00", "17:00")
	model.Normalize = types.BoolValue(true)
	normalizeFirewallPolicyScheduleModel(&model)
	if !model.TimeRangeStart.IsNull() || !model.TimeRangeEnd.IsNull() {
		t.Fatalf("all-day model retained a time range: %#v", model)
	}
}

func TestFirewallPolicyScheduleNormalizeRoundTrip(t *testing.T) {
	allDay := true
	policy := testScheduledFirewallPolicy(&api.FirewallPolicySchedule{
		Date: "stale", DateStart: "stale", DateEnd: "stale", Mode: "EVERY_WEEK",
		RepeatOnDays: []string{"mon"}, TimeAllDay: &allDay,
		TimeRangeStart: "09:00", TimeRangeEnd: "17:00",
	})
	var model firewallPolicyModel
	if diags := firewallPolicyToModel(context.Background(), policy, &model); diags.HasError() {
		t.Fatalf("API to model: %v", diags)
	}
	var schedule firewallPolicyScheduleModel
	if diags := model.Schedule.As(
		context.Background(),
		&schedule,
		basetypes.ObjectAsOptions{},
	); diags.HasError() {
		t.Fatalf("reading schedule: %v", diags)
	}
	schedule.Normalize = types.BoolValue(true)
	value, objectDiags := types.ObjectValueFrom(
		context.Background(),
		schedule.AttributeTypes(),
		schedule,
	)
	if objectDiags.HasError() {
		t.Fatalf("building normalized schedule: %v", objectDiags)
	}
	model.Schedule = value

	got, diags := modelToFirewallPolicy(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("model to API: %v", diags)
	}
	if got.Schedule.Date != "" || got.Schedule.DateStart != "" ||
		got.Schedule.TimeRangeStart != "" {
		t.Fatalf("normalized schedule retained unused fields: %#v", got.Schedule)
	}
	if diags := firewallPolicyToModel(context.Background(), got, &model); diags.HasError() {
		t.Fatalf("refreshing model: %v", diags)
	}
	if diags := model.Schedule.As(
		context.Background(),
		&schedule,
		basetypes.ObjectAsOptions{},
	); diags.HasError() ||
		!schedule.Normalize.ValueBool() {
		t.Fatalf("normalize was not preserved across refresh: %v", diags)
	}
}

func scheduleForTest(mode string) firewallPolicyScheduleModel {
	return firewallPolicyScheduleModel{
		Date: types.StringNull(), DateStart: types.StringNull(), DateEnd: types.StringNull(),
		Mode: types.StringValue(mode), Normalize: types.BoolValue(false),
		RepeatOnDays: types.SetNull(types.StringType), TimeAllDay: types.BoolNull(),
		TimeRangeStart: types.StringNull(), TimeRangeEnd: types.StringNull(),
	}
}

func withScheduleTime(
	s firewallPolicyScheduleModel,
	allDay bool,
	start, end string,
) firewallPolicyScheduleModel {
	s.TimeAllDay = types.BoolValue(allDay)
	if start != "" {
		s.TimeRangeStart = types.StringValue(start)
	}
	if end != "" {
		s.TimeRangeEnd = types.StringValue(end)
	}
	return s
}

func withScheduleDays(s firewallPolicyScheduleModel, days ...string) firewallPolicyScheduleModel {
	s.RepeatOnDays = types.SetValueMust(types.StringType, stringsToValues(days))
	return s
}

func withScheduleDate(s firewallPolicyScheduleModel, date string) firewallPolicyScheduleModel {
	s.Date = types.StringValue(date)
	return s
}

func withScheduleRange(
	s firewallPolicyScheduleModel,
	start, end string,
) firewallPolicyScheduleModel {
	s.DateStart, s.DateEnd = types.StringValue(start), types.StringValue(end)
	return s
}

func stringsToValues(values []string) []attr.Value {
	result := make([]attr.Value, len(values))
	for i, value := range values {
		result[i] = types.StringValue(value)
	}
	return result
}

func TestFirewallPolicyScheduleRoundTripsControllerValue(t *testing.T) {
	allDay := false
	want := &api.FirewallPolicySchedule{
		Date:           "2026-07-10",
		DateStart:      "2026-07-01",
		DateEnd:        "2026-07-31",
		Mode:           "EVERY_WEEK",
		RepeatOnDays:   []string{"mon", "wed", "fri"},
		TimeAllDay:     &allDay,
		TimeRangeStart: "09:00",
		TimeRangeEnd:   "17:30",
	}
	assertFirewallPolicyScheduleRoundTrip(t, want)
}

func TestFirewallPolicySchedulePreservesLegacyAlwaysMetadata(t *testing.T) {
	allDay := false
	want := &api.FirewallPolicySchedule{
		DateStart:      "2025-06-20",
		DateEnd:        "2025-06-27",
		Mode:           "ALWAYS",
		RepeatOnDays:   []string{},
		TimeAllDay:     &allDay,
		TimeRangeStart: "09:00",
		TimeRangeEnd:   "12:00",
	}
	assertFirewallPolicyScheduleRoundTrip(t, want)
}

func TestFirewallPolicyOmittedScheduleFallsBackToAlways(t *testing.T) {
	policy := testScheduledFirewallPolicy(nil)
	var model firewallPolicyModel
	if diags := firewallPolicyToModel(context.Background(), policy, &model); diags.HasError() {
		t.Fatalf("API to resource model conversion failed: %v", diags)
	}
	roundTripped, diags := modelToFirewallPolicy(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("resource model to API conversion failed: %v", diags)
	}
	if roundTripped.Schedule == nil || roundTripped.Schedule.Mode != "ALWAYS" {
		t.Fatalf("omitted schedule = %#v, want ALWAYS fallback", roundTripped.Schedule)
	}
}

func assertFirewallPolicyScheduleRoundTrip(t *testing.T, want *api.FirewallPolicySchedule) {
	t.Helper()
	var model firewallPolicyModel
	if diags := firewallPolicyToModel(
		context.Background(), testScheduledFirewallPolicy(want), &model,
	); diags.HasError() {
		t.Fatalf("API to resource model conversion failed: %v", diags)
	}
	roundTripped, diags := modelToFirewallPolicy(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("resource model to API conversion failed: %v", diags)
	}
	if !reflect.DeepEqual(roundTripped.Schedule, want) {
		t.Fatalf(
			"schedule changed during round-trip:\n got: %#v\nwant: %#v",
			roundTripped.Schedule,
			want,
		)
	}
}

func testScheduledFirewallPolicy(schedule *api.FirewallPolicySchedule) *api.FirewallPolicy {
	return &api.FirewallPolicy{
		Name:     "scheduled policy",
		Action:   "BLOCK",
		Enabled:  true,
		Protocol: "all",
		Version:  "BOTH",
		Schedule: schedule,
		Source: &api.FirewallPolicySource{
			ZoneID:           "zone-internal",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
		Destination: &api.FirewallPolicyDestination{
			ZoneID:           "zone-external",
			MatchingTarget:   "ANY",
			PortMatchingType: "ANY",
		},
	}
}
