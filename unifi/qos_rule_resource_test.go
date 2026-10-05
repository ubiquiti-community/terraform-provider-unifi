package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func Test_qosRuleResource_Schema(t *testing.T) {
	resp := &fwresource.SchemaResponse{}
	(&qosRuleResource{}).Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() produced errors: %v", resp.Diagnostics)
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("schema invalid: %v", diags)
	}
}

func ptrI64(v int64) *int64 { return &v }

// Rules as UniFi Network 10.6 returns them.
func liveQOSRules() []unifi.QOSRule {
	return []unifi.QOSRule{
		{
			ID: "68766e5b6fd99c15fed6f604", Name: "Prioritize Online Gaming", Enabled: true,
			Index: ptrI64(10002), Objective: "PRIORITIZE", DownloadBurst: "OFF", UploadBurst: "OFF",
			Source: &unifi.QOSRuleSource{MatchingTarget: "ANY"},
			Destination: &unifi.QOSRuleDestination{
				MatchingTarget: "APP_CATEGORY", AppCategoryIDs: []int64{8}, PortMatchingType: "ANY",
			},
			Schedule: &unifi.QOSRuleSchedule{Mode: "ALWAYS", RepeatOnDays: []string{}},
		},
		{
			ID:                "68766e5b6fd99c15fed6f605",
			Name:              "limit",
			Enabled:           true,
			Index:             ptrI64(10003),
			Objective:         "LIMIT",
			DownloadLimitKbps: ptrI64(40000),
			UploadLimitKbps:   ptrI64(10000),
			DownloadBurst:     "OFF",
			UploadBurst:       "OFF",
			Source: &unifi.QOSRuleSource{
				MatchingTarget: "CLIENT",
				ClientMACs:     []string{"9c:6b:00:39:f7:a6"},
			},
			Destination: &unifi.QOSRuleDestination{
				MatchingTarget: "IP", MatchingTargetType: "SPECIFIC", IPs: []string{"10.0.0.0/8"},
				PortMatchingType: "SPECIFIC", Port: "80,443",
			},
			Schedule: &unifi.QOSRuleSchedule{
				Mode:           "EVERY_WEEK",
				RepeatOnDays:   []string{"mon"},
				TimeRangeStart: "18:00",
				TimeRangeEnd:   "23:00",
			},
		},
	}
}

// Reading a rule and writing it back must send what the controller has, so an
// imported rule is not changed by the first apply.
func Test_qosRuleResource_roundTrip(t *testing.T) {
	r, ctx := &qosRuleResource{}, context.Background()
	for _, live := range liveQOSRules() {
		var diags diag.Diagnostics
		var m qosRuleModel
		r.apiToModel(ctx, &live, &m, "default", &diags)
		got := r.modelToAPI(ctx, &m, &diags)
		if diags.HasError() {
			t.Fatalf("%s: unexpected diags: %v", live.Name, diags)
		}
		want := live
		want.ID, want.Index = "", nil // not part of the write body
		if len(want.Schedule.RepeatOnDays) == 0 {
			want.Schedule.RepeatOnDays = nil
		}
		want.Source.PortMatchingType = "ANY"
		if !reflect.DeepEqual(*got, want) {
			t.Errorf("%s: round trip =\n%+v\nwant\n%+v", live.Name, *got, want)
		}
	}
}

func Test_qosRuleResource_defaults(t *testing.T) {
	r, ctx := &qosRuleResource{}, context.Background()
	var diags diag.Diagnostics
	var m qosRuleModel
	r.apiToModel(ctx, &unifi.QOSRule{Objective: "PRIORITIZE"}, &m, "default", &diags)
	if !m.Destination.Attributes()["port_matching_type"].IsNull() {
		t.Error("an unmatched port should read as null")
	}
	got := r.modelToAPI(ctx, &m, &diags)
	if got.Source.MatchingTarget != "ANY" || got.Destination.PortMatchingType != "ANY" ||
		got.Schedule.Mode != "ALWAYS" {
		t.Errorf("defaults not applied: %+v %+v %+v", got.Source, got.Destination, got.Schedule)
	}
}
