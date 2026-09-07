package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type firewallPolicyScheduleValidator struct{}

func (firewallPolicyScheduleValidator) Description(context.Context) string {
	return "validates fields required by the selected firewall policy schedule mode"
}

func (v firewallPolicyScheduleValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (firewallPolicyScheduleValidator) ValidateObject(
	ctx context.Context,
	req validator.ObjectRequest,
	resp *validator.ObjectResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var s firewallPolicyScheduleModel
	resp.Diagnostics.Append(req.ConfigValue.As(ctx, &s, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() || s.Mode.IsUnknown() {
		return
	}

	mode := s.Mode.ValueString()
	if s.Mode.IsNull() || mode == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Schedule Mode",
			"A configured schedule must set mode.",
		)
		return
	}

	dates := map[string]types.String{
		"date":       s.Date,
		"date_start": s.DateStart,
		"date_end":   s.DateEnd,
	}
	parsed := map[string]time.Time{}
	for name, value := range dates {
		if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
			continue
		}
		date, err := time.Parse("2006-01-02", value.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Schedule Date",
				fmt.Sprintf("%s must be a real date in YYYY-MM-DD format.", name),
			)
			continue
		}
		parsed[name] = date
	}
	if start, ok := parsed["date_start"]; ok {
		if end, ok := parsed["date_end"]; ok && end.Before(start) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Schedule Date Range",
				"date_end must be on or after date_start.",
			)
		}
	}

	requireString := func(name string, value types.String) {
		if !value.IsUnknown() && (value.IsNull() || value.ValueString() == "") {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Schedule",
				fmt.Sprintf("Schedule mode %s requires %s.", mode, name),
			)
		}
	}
	requireDays := func() {
		if !s.RepeatOnDays.IsUnknown() &&
			(s.RepeatOnDays.IsNull() || len(s.RepeatOnDays.Elements()) == 0) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Schedule",
				fmt.Sprintf("Schedule mode %s requires at least one repeat_on_days value.", mode),
			)
		}
	}
	requireTime := func(oneTime bool) {
		allDay, rangeStart, rangeEnd := firewallPolicyScheduleTimeFields(s.Time)
		if allDay.IsUnknown() {
			return
		}
		if allDay.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Schedule",
				fmt.Sprintf("Schedule mode %s requires time.all_day.", mode),
			)
			return
		}
		if allDay.ValueBool() {
			if oneTime {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid One-Time Schedule",
					"ONE_TIME_ONLY requires time.all_day = false and an explicit time.range.",
				)
			}
			return
		}
		requireString("time.range.start", rangeStart)
		requireString("time.range.end", rangeEnd)
	}

	switch mode {
	case "EVERY_DAY":
		requireTime(false)
	case "EVERY_WEEK":
		requireDays()
		requireTime(false)
	case "ONE_TIME_ONLY":
		requireString("date", s.Date)
		requireTime(true)
	case "CUSTOM":
		requireString("date_start", s.DateStart)
		requireString("date_end", s.DateEnd)
		requireDays()
		requireTime(false)
	}
}
