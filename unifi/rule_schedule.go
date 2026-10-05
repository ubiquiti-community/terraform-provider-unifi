package unifi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ruleScheduleModel is the schedule shared by unifi_qos_rule and
// unifi_content_filter. The controller uses the firewall policy schedule shape
// for both.
type ruleScheduleModel struct {
	Mode           types.String `tfsdk:"mode"`
	RepeatOnDays   types.Set    `tfsdk:"repeat_on_days"`
	TimeAllDay     types.Bool   `tfsdk:"time_all_day"`
	TimeRangeStart types.String `tfsdk:"time_range_start"`
	TimeRangeEnd   types.String `tfsdk:"time_range_end"`
	Date           types.String `tfsdk:"date"`
	DateStart      types.String `tfsdk:"date_start"`
	DateEnd        types.String `tfsdk:"date_end"`
}

// ruleSchedule is the wire form shared by the generated QOSRuleSchedule and
// ContentFilteringSchedule structs.
type ruleSchedule struct {
	Mode           string
	RepeatOnDays   []string
	TimeAllDay     bool
	TimeRangeStart string
	TimeRangeEnd   string
	Date           string
	DateStart      string
	DateEnd        string
}

var ruleScheduleAttrTypes = map[string]attr.Type{
	"mode":             types.StringType,
	"repeat_on_days":   types.SetType{ElemType: types.StringType},
	"time_all_day":     types.BoolType,
	"time_range_start": types.StringType,
	"time_range_end":   types.StringType,
	"date":             types.StringType,
	"date_start":       types.StringType,
	"date_end":         types.StringType,
}

func ruleScheduleSchemaAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "When the rule is active. Omit for always.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes: map[string]schema.Attribute{
			"mode": schema.StringAttribute{
				MarkdownDescription: "`ALWAYS`, `EVERY_DAY`, `EVERY_WEEK`, `ONE_TIME_ONLY` or `CUSTOM`.",
				Required:            true,
				Validators: []validator.String{stringvalidator.OneOf(
					"ALWAYS", "EVERY_DAY", "EVERY_WEEK", "ONE_TIME_ONLY", "CUSTOM")},
			},
			"repeat_on_days": schema.SetAttribute{
				MarkdownDescription: "Days for `EVERY_WEEK`/`CUSTOM`: `mon` … `sun`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"time_all_day":     schema.BoolAttribute{MarkdownDescription: "Active the whole day.", Optional: true},
			"time_range_start": schema.StringAttribute{MarkdownDescription: "Start time, `HH:MM`.", Optional: true},
			"time_range_end":   schema.StringAttribute{MarkdownDescription: "End time, `HH:MM`.", Optional: true},
			"date":             schema.StringAttribute{MarkdownDescription: "Date for `ONE_TIME_ONLY`, `YYYY-MM-DD`.", Optional: true},
			"date_start":       schema.StringAttribute{MarkdownDescription: "First date, `YYYY-MM-DD`.", Optional: true},
			"date_end":         schema.StringAttribute{MarkdownDescription: "Last date, `YYYY-MM-DD`.", Optional: true},
		},
	}
}

// scheduleFromObject converts the schedule attribute; null or unknown means always.
func scheduleFromObject(ctx context.Context, obj types.Object, diags *diag.Diagnostics) ruleSchedule {
	if !isKnown(obj) {
		return ruleSchedule{Mode: "ALWAYS"}
	}
	var m ruleScheduleModel
	diags.Append(obj.As(ctx, &m, basetypes.ObjectAsOptions{})...)
	s := ruleSchedule{
		Mode:           m.Mode.ValueString(),
		TimeAllDay:     m.TimeAllDay.ValueBool(),
		TimeRangeStart: m.TimeRangeStart.ValueString(),
		TimeRangeEnd:   m.TimeRangeEnd.ValueString(),
		Date:           m.Date.ValueString(),
		DateStart:      m.DateStart.ValueString(),
		DateEnd:        m.DateEnd.ValueString(),
	}
	s.RepeatOnDays = stringsFromSet(ctx, m.RepeatOnDays, diags)
	return s
}

func scheduleToObject(ctx context.Context, s ruleSchedule, diags *diag.Diagnostics) types.Object {
	mode := s.Mode
	if mode == "" {
		mode = "ALWAYS"
	}
	m := ruleScheduleModel{
		Mode:           types.StringValue(mode),
		RepeatOnDays:   setOrNull(ctx, s.RepeatOnDays, diags),
		TimeAllDay:     boolOrNull(s.TimeAllDay),
		TimeRangeStart: stringOrNull(s.TimeRangeStart),
		TimeRangeEnd:   stringOrNull(s.TimeRangeEnd),
		Date:           stringOrNull(s.Date),
		DateStart:      stringOrNull(s.DateStart),
		DateEnd:        stringOrNull(s.DateEnd),
	}
	obj, d := types.ObjectValueFrom(ctx, ruleScheduleAttrTypes, m)
	diags.Append(d...)
	return obj
}

// setOrNull returns a set of v, or null when v is empty: the controller omits
// or empties lists that are not in use, and config omits them.
func setOrNull(ctx context.Context, v []string, diags *diag.Diagnostics) types.Set {
	if len(v) == 0 {
		return types.SetNull(types.StringType)
	}
	s, d := types.SetValueFrom(ctx, types.StringType, v)
	diags.Append(d...)
	return s
}

func int64SetOrNull(ctx context.Context, v []int64, diags *diag.Diagnostics) types.Set {
	if len(v) == 0 {
		return types.SetNull(types.Int64Type)
	}
	s, d := types.SetValueFrom(ctx, types.Int64Type, v)
	diags.Append(d...)
	return s
}

func stringsFromSet(ctx context.Context, s types.Set, diags *diag.Diagnostics) []string {
	if !isKnown(s) {
		return nil
	}
	var v []string
	diags.Append(s.ElementsAs(ctx, &v, false)...)
	return v
}

func int64sFromSet(ctx context.Context, s types.Set, diags *diag.Diagnostics) []int64 {
	if !isKnown(s) {
		return nil
	}
	var v []int64
	diags.Append(s.ElementsAs(ctx, &v, false)...)
	return v
}

func boolOrNull(b bool) types.Bool {
	if !b {
		return types.BoolNull()
	}
	return types.BoolValue(true)
}
