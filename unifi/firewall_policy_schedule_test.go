package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	api "github.com/ubiquiti-community/go-unifi/unifi"
)

func TestFirewallPolicySchemaExposesComputedSchedule(t *testing.T) {
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
	if schedule.Optional || !schedule.Computed {
		t.Fatalf(
			"schedule Optional=%v Computed=%v, want computed-only",
			schedule.Optional,
			schedule.Computed,
		)
	}
	for name, attribute := range schedule.Attributes {
		switch field := attribute.(type) {
		case schema.StringAttribute:
			if !field.Computed || field.Optional {
				t.Errorf("%s is not computed-only", name)
			}
		case schema.BoolAttribute:
			if !field.Computed || field.Optional {
				t.Errorf("%s is not computed-only", name)
			}
		case schema.SetAttribute:
			if !field.Computed || field.Optional {
				t.Errorf("%s is not computed-only", name)
			}
		default:
			t.Errorf("unexpected schedule attribute type for %s: %T", name, attribute)
		}
	}
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
