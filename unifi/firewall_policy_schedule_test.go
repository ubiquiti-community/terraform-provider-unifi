package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	api "github.com/ubiquiti-community/go-unifi/unifi"
)

func TestFirewallPolicySchemaExposesOptionalComputedSchedule(t *testing.T) {
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
		t.Fatalf("schedule Optional=%v Computed=%v, want both true", schedule.Optional, schedule.Computed)
	}

	wantFields := []string{
		"date", "date_start", "date_end", "mode", "normalize", "repeat_on_days",
		"time_all_day", "time_range_start", "time_range_end",
	}
	for _, field := range wantFields {
		if _, ok := schedule.Attributes[field]; !ok {
			t.Errorf("schedule schema has no %q attribute", field)
		}
	}
}

func TestFirewallPolicyScheduleValidatorRejectsIncompleteEveryWeek(t *testing.T) {
	schedule, diags := types.ObjectValueFrom(
		context.Background(),
		firewallPolicyScheduleModel{}.AttributeTypes(),
		firewallPolicyScheduleModel{
			Mode:           types.StringValue("EVERY_WEEK"),
			RepeatOnDays:   types.SetValueMust(types.StringType, []attr.Value{}),
			TimeAllDay:     types.BoolValue(false),
			Date:           types.StringNull(),
			DateStart:      types.StringNull(),
			DateEnd:        types.StringNull(),
			TimeRangeStart: types.StringNull(),
			TimeRangeEnd:   types.StringNull(),
		},
	)
	if diags.HasError() {
		t.Fatalf("building schedule object: %v", diags)
	}

	var response validator.ObjectResponse
	firewallPolicyScheduleValidator{}.ValidateObject(
		context.Background(),
		validator.ObjectRequest{ConfigValue: schedule},
		&response,
	)
	if !response.Diagnostics.HasError() {
		t.Fatal("incomplete EVERY_WEEK schedule passed validation")
	}
}

func TestFirewallPolicyScheduleValidatorModeRequirements(t *testing.T) {
	allDays := types.SetValueMust(
		types.StringType,
		[]attr.Value{types.StringValue("mon"), types.StringValue("fri")},
	)
	emptyDays := types.SetValueMust(types.StringType, []attr.Value{})

	cases := map[string]struct {
		schedule  firewallPolicyScheduleModel
		wantError bool
	}{
		"always permits retained metadata": {
			schedule: testScheduleModel("ALWAYS", emptyDays, false),
		},
		"every day timed": {
			schedule: withScheduleTimes(
				testScheduleModel("EVERY_DAY", emptyDays, false), "09:00", "17:00",
			),
		},
		"every day all day": {
			schedule: testScheduleModel("EVERY_DAY", emptyDays, true),
		},
		"every day missing time": {
			schedule:  testScheduleModel("EVERY_DAY", emptyDays, false),
			wantError: true,
		},
		"every day missing all-day flag": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("EVERY_DAY", emptyDays, false), "09:00", "17:00",
				)
				s.TimeAllDay = types.BoolNull()
				return s
			}(),
			wantError: true,
		},
		"every week complete": {
			schedule: withScheduleTimes(
				testScheduleModel("EVERY_WEEK", allDays, false), "09:00", "17:00",
			),
		},
		"every week missing days": {
			schedule: withScheduleTimes(
				testScheduleModel("EVERY_WEEK", emptyDays, false), "09:00", "17:00",
			),
			wantError: true,
		},
		"one time only complete": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("ONE_TIME_ONLY", emptyDays, false), "09:00", "17:00",
				)
				s.Date = types.StringValue("2026-08-01")
				return s
			}(),
		},
		"one time only rejects all day": {
			schedule: func() firewallPolicyScheduleModel {
				s := testScheduleModel("ONE_TIME_ONLY", emptyDays, true)
				s.Date = types.StringValue("2026-08-01")
				return s
			}(),
			wantError: true,
		},
		"one time only missing date": {
			schedule: withScheduleTimes(
				testScheduleModel("ONE_TIME_ONLY", emptyDays, false), "09:00", "17:00",
			),
			wantError: true,
		},
		"one time only missing all-day flag": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("ONE_TIME_ONLY", emptyDays, false), "09:00", "17:00",
				)
				s.Date = types.StringValue("2026-08-01")
				s.TimeAllDay = types.BoolNull()
				return s
			}(),
			wantError: true,
		},
		"custom complete": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("CUSTOM", allDays, false), "19:00", "23:00",
				)
				s.DateStart = types.StringValue("2026-08-01")
				s.DateEnd = types.StringValue("2026-08-31")
				return s
			}(),
		},
		"custom missing date range": {
			schedule: withScheduleTimes(
				testScheduleModel("CUSTOM", allDays, false), "19:00", "23:00",
			),
			wantError: true,
		},
		"custom rejects impossible calendar date": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("CUSTOM", allDays, false), "19:00", "23:00",
				)
				s.DateStart = types.StringValue("2026-02-31")
				s.DateEnd = types.StringValue("2026-03-31")
				return s
			}(),
			wantError: true,
		},
		"custom rejects reversed date range": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("CUSTOM", allDays, false), "19:00", "23:00",
				)
				s.DateStart = types.StringValue("2026-08-31")
				s.DateEnd = types.StringValue("2026-08-01")
				return s
			}(),
			wantError: true,
		},
		"normalized always rejects configured stale time": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("ALWAYS", emptyDays, false), "09:00", "17:00",
				)
				s.Normalize = types.BoolValue(true)
				return s
			}(),
			wantError: true,
		},
		"normalized always rejects explicit false all-day": {
			schedule: func() firewallPolicyScheduleModel {
				s := testScheduleModel("ALWAYS", emptyDays, false)
				s.Normalize = types.BoolValue(true)
				return s
			}(),
			wantError: true,
		},
		"normalized weekly all day rejects configured time": {
			schedule: func() firewallPolicyScheduleModel {
				s := withScheduleTimes(
					testScheduleModel("EVERY_WEEK", allDays, true), "09:00", "17:00",
				)
				s.Normalize = types.BoolValue(true)
				return s
			}(),
			wantError: true,
		},
		"configured schedule missing mode": {
			schedule:  testScheduleModel("", emptyDays, false),
			wantError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			object, diags := types.ObjectValueFrom(
				context.Background(),
				firewallPolicyScheduleModel{}.AttributeTypes(),
				tc.schedule,
			)
			if diags.HasError() {
				t.Fatalf("building schedule object: %v", diags)
			}
			var response validator.ObjectResponse
			firewallPolicyScheduleValidator{}.ValidateObject(
				context.Background(),
				validator.ObjectRequest{ConfigValue: object},
				&response,
			)
			if response.Diagnostics.HasError() != tc.wantError {
				t.Fatalf("HasError = %v, want %v: %v", response.Diagnostics.HasError(), tc.wantError, response.Diagnostics)
			}
		})
	}
}

func testScheduleModel(mode string, days types.Set, allDay bool) firewallPolicyScheduleModel {
	return firewallPolicyScheduleModel{
		Date:           types.StringNull(),
		DateStart:      types.StringNull(),
		DateEnd:        types.StringNull(),
		Mode:           types.StringValue(mode),
		RepeatOnDays:   days,
		TimeAllDay:     types.BoolValue(allDay),
		TimeRangeStart: types.StringNull(),
		TimeRangeEnd:   types.StringNull(),
	}
}

func withScheduleTimes(
	schedule firewallPolicyScheduleModel,
	start string,
	end string,
) firewallPolicyScheduleModel {
	schedule.TimeRangeStart = types.StringValue(start)
	schedule.TimeRangeEnd = types.StringValue(end)
	return schedule
}

func testBoolPointer(value bool) *bool {
	return &value
}

func TestFirewallPolicyScheduleRoundTripsThroughResourceModel(t *testing.T) {
	want := &api.FirewallPolicySchedule{
		Date:           "2026-07-10",
		DateStart:      "2026-07-01",
		DateEnd:        "2026-07-31",
		Mode:           "EVERY_WEEK",
		RepeatOnDays:   []string{"mon", "wed", "fri"},
		TimeAllDay:     testBoolPointer(false),
		TimeRangeStart: "09:00",
		TimeRangeEnd:   "17:30",
	}
	assertFirewallPolicyScheduleRoundTrip(t, want)
}

func TestFirewallPolicySchedulePreservesLegacyFieldsUnderAlways(t *testing.T) {
	want := &api.FirewallPolicySchedule{
		DateStart:      "2025-06-20",
		DateEnd:        "2025-06-27",
		Mode:           "ALWAYS",
		RepeatOnDays:   []string{},
		TimeAllDay:     testBoolPointer(false),
		TimeRangeStart: "09:00",
		TimeRangeEnd:   "12:00",
	}
	assertFirewallPolicyScheduleRoundTrip(t, want)
}

func TestFirewallPolicyScheduleModesRoundTrip(t *testing.T) {
	cases := map[string]*api.FirewallPolicySchedule{
		"ALWAYS": {
			Mode: "ALWAYS",
		},
		"EVERY_DAY": {
			Mode:           "EVERY_DAY",
			TimeAllDay:     testBoolPointer(false),
			TimeRangeStart: "08:00",
			TimeRangeEnd:   "18:00",
		},
		"EVERY_WEEK": {
			Mode:         "EVERY_WEEK",
			RepeatOnDays: []string{"mon", "fri"},
			TimeAllDay:   testBoolPointer(true),
		},
		"ONE_TIME_ONLY": {
			Date:           "2026-08-01",
			Mode:           "ONE_TIME_ONLY",
			TimeAllDay:     testBoolPointer(false),
			TimeRangeStart: "09:00",
			TimeRangeEnd:   "17:00",
		},
		"CUSTOM": {
			DateStart:      "2026-08-01",
			DateEnd:        "2026-08-31",
			Mode:           "CUSTOM",
			RepeatOnDays:   []string{"tue", "thu"},
			TimeAllDay:     testBoolPointer(false),
			TimeRangeStart: "19:00",
			TimeRangeEnd:   "23:00",
		},
	}

	for name, schedule := range cases {
		t.Run(name, func(t *testing.T) {
			assertFirewallPolicyScheduleRoundTrip(t, schedule)
		})
	}
}

func TestFirewallPolicyScheduleCanonicalizesMissingDaysToEmptySet(t *testing.T) {
	policy := &api.FirewallPolicy{
		Schedule: &api.FirewallPolicySchedule{Mode: "ALWAYS"},
	}
	var model firewallPolicyModel
	diags := firewallPolicyToModel(context.Background(), policy, &model)
	if diags.HasError() {
		t.Fatalf("API to resource model conversion failed: %v", diags)
	}

	var schedule firewallPolicyScheduleModel
	diags = model.Schedule.As(
		context.Background(), &schedule, basetypes.ObjectAsOptions{},
	)
	if diags.HasError() {
		t.Fatalf("reading schedule model: %v", diags)
	}
	if schedule.RepeatOnDays.IsNull() || schedule.RepeatOnDays.IsUnknown() ||
		len(schedule.RepeatOnDays.Elements()) != 0 {
		t.Fatalf("repeat_on_days = %v, want known empty set", schedule.RepeatOnDays)
	}
}

func TestFirewallPolicyOmittedScheduleDefaultsToAlways(t *testing.T) {
	policy := &api.FirewallPolicy{
		Name:     "unscheduled policy",
		Action:   "BLOCK",
		Enabled:  true,
		Protocol: "all",
		Version:  "BOTH",
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

	var model firewallPolicyModel
	diags := firewallPolicyToModel(context.Background(), policy, &model)
	if diags.HasError() {
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

func TestFirewallPolicyScheduleNormalizeClearsFieldsUnusedByAlways(t *testing.T) {
	want := &api.FirewallPolicySchedule{
		Date:           "2025-06-20",
		DateStart:      "2025-06-20",
		DateEnd:        "2025-06-27",
		Mode:           "ALWAYS",
		RepeatOnDays:   []string{"mon", "fri"},
		TimeAllDay:     testBoolPointer(false),
		TimeRangeStart: "09:00",
		TimeRangeEnd:   "12:00",
	}
	policy := &api.FirewallPolicy{
		Name:        "normalize schedule",
		Action:      "BLOCK",
		Protocol:    "all",
		Version:     "BOTH",
		Schedule:    want,
		Source:      &api.FirewallPolicySource{ZoneID: "internal", MatchingTarget: "ANY"},
		Destination: &api.FirewallPolicyDestination{ZoneID: "external", MatchingTarget: "ANY"},
	}

	var model firewallPolicyModel
	diags := firewallPolicyToModel(context.Background(), policy, &model)
	if diags.HasError() {
		t.Fatalf("API to resource model conversion failed: %v", diags)
	}
	var schedule firewallPolicyScheduleModel
	diags = model.Schedule.As(
		context.Background(), &schedule, basetypes.ObjectAsOptions{},
	)
	if diags.HasError() {
		t.Fatalf("reading schedule model: %v", diags)
	}
	schedule.Normalize = types.BoolValue(true)
	model.Schedule, diags = types.ObjectValueFrom(
		context.Background(),
		firewallPolicyScheduleModel{}.AttributeTypes(),
		schedule,
	)
	if diags.HasError() {
		t.Fatalf("writing schedule model: %v", diags)
	}

	roundTripped, diags := modelToFirewallPolicy(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("resource model to API conversion failed: %v", diags)
	}
	got := roundTripped.Schedule
	if got == nil || got.Mode != "ALWAYS" || got.Date != "" ||
		got.DateStart != "" || got.DateEnd != "" || len(got.RepeatOnDays) != 0 ||
		got.TimeAllDay != nil || got.TimeRangeStart != "" || got.TimeRangeEnd != "" {
		t.Fatalf("normalized ALWAYS schedule retained unused fields: %#v", got)
	}
}

func TestFirewallPolicyScheduleNormalizeByMode(t *testing.T) {
	falseValue := testBoolPointer(false)
	cases := map[string]*api.FirewallPolicySchedule{
		"ALWAYS": {
			Mode: "ALWAYS",
		},
		"EVERY_DAY": {
			Mode:           "EVERY_DAY",
			TimeAllDay:     falseValue,
			TimeRangeStart: "09:00",
			TimeRangeEnd:   "17:00",
		},
		"EVERY_WEEK": {
			Mode:           "EVERY_WEEK",
			RepeatOnDays:   []string{"mon", "fri"},
			TimeAllDay:     falseValue,
			TimeRangeStart: "09:00",
			TimeRangeEnd:   "17:00",
		},
		"ONE_TIME_ONLY": {
			Date:           "2026-08-15",
			Mode:           "ONE_TIME_ONLY",
			TimeAllDay:     falseValue,
			TimeRangeStart: "09:00",
			TimeRangeEnd:   "17:00",
		},
		"CUSTOM": {
			DateStart:      "2026-08-01",
			DateEnd:        "2026-08-31",
			Mode:           "CUSTOM",
			RepeatOnDays:   []string{"mon", "fri"},
			TimeAllDay:     falseValue,
			TimeRangeStart: "19:00",
			TimeRangeEnd:   "23:00",
		},
	}

	for mode, want := range cases {
		t.Run(mode, func(t *testing.T) {
			got := &api.FirewallPolicySchedule{
				Date:           "2026-08-15",
				DateStart:      "2026-08-01",
				DateEnd:        "2026-08-31",
				Mode:           mode,
				RepeatOnDays:   []string{"mon", "fri"},
				TimeAllDay:     testBoolPointer(false),
				TimeRangeStart: "09:00",
				TimeRangeEnd:   "17:00",
			}
			if mode == "CUSTOM" {
				got.TimeRangeStart = "19:00"
				got.TimeRangeEnd = "23:00"
			}
			normalizeFirewallPolicySchedule(got)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("normalized %s schedule:\n got: %#v\nwant: %#v", mode, got, want)
			}
		})
	}
}

func TestFirewallPolicyScheduleNormalizePlanMatchesControllerResult(t *testing.T) {
	schedule := firewallPolicyScheduleModel{
		Date:           types.StringNull(),
		DateStart:      types.StringNull(),
		DateEnd:        types.StringNull(),
		Mode:           types.StringValue("EVERY_WEEK"),
		Normalize:      types.BoolValue(true),
		RepeatOnDays:   types.SetValueMust(types.StringType, []attr.Value{types.StringValue("mon"), types.StringValue("fri")}),
		TimeAllDay:     types.BoolValue(true),
		TimeRangeStart: types.StringValue("09:00"),
		TimeRangeEnd:   types.StringValue("17:00"),
	}
	value, diags := types.ObjectValueFrom(
		context.Background(),
		firewallPolicyScheduleModel{}.AttributeTypes(),
		schedule,
	)
	if diags.HasError() {
		t.Fatalf("building schedule object: %v", diags)
	}

	normalized, changed, diags := normalizeFirewallPolicySchedulePlan(
		context.Background(), value,
	)
	if diags.HasError() {
		t.Fatalf("normalizing schedule plan: %v", diags)
	}
	if !changed {
		t.Fatal("normalization did not report a planned change")
	}

	var got firewallPolicyScheduleModel
	diags = normalized.As(
		context.Background(), &got, basetypes.ObjectAsOptions{},
	)
	if diags.HasError() {
		t.Fatalf("reading normalized schedule plan: %v", diags)
	}
	if !got.TimeRangeStart.IsNull() || !got.TimeRangeEnd.IsNull() {
		t.Fatalf(
			"all-day normalized plan retained time range: start=%v end=%v",
			got.TimeRangeStart,
			got.TimeRangeEnd,
		)
	}
	if got.RepeatOnDays.IsNull() || len(got.RepeatOnDays.Elements()) != 2 {
		t.Fatalf("EVERY_WEEK normalized plan changed weekdays: %v", got.RepeatOnDays)
	}
}

func assertFirewallPolicyScheduleRoundTrip(t *testing.T, want *api.FirewallPolicySchedule) {
	t.Helper()
	policy := &api.FirewallPolicy{
		Name:     "scheduled policy",
		Action:   "BLOCK",
		Enabled:  true,
		Protocol: "all",
		Version:  "BOTH",
		Schedule: want,
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

	var model firewallPolicyModel
	diags := firewallPolicyToModel(context.Background(), policy, &model)
	if diags.HasError() {
		t.Fatalf("API to resource model conversion failed: %v", diags)
	}

	roundTripped, diags := modelToFirewallPolicy(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("resource model to API conversion failed: %v", diags)
	}
	normalizedWant := *want
	if normalizedWant.RepeatOnDays == nil {
		normalizedWant.RepeatOnDays = []string{}
	}
	if !reflect.DeepEqual(roundTripped.Schedule, &normalizedWant) {
		t.Fatalf("schedule changed during round-trip:\n got: %#v\nwant: %#v", roundTripped.Schedule, &normalizedWant)
	}
}
