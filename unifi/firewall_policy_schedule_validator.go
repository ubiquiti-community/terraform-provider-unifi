package unifi

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// firewallPolicyScheduleValidator enforces only the fields required by the
// selected mode. It intentionally permits surplus fields because controllers
// can retain legacy schedule metadata under ALWAYS, and that metadata must be
// round-tripped during unrelated policy updates.
type firewallPolicyScheduleValidator struct{}

func (firewallPolicyScheduleValidator) Description(context.Context) string {
	return "validates the required date, weekday, and time fields for a firewall policy schedule mode"
}

func (firewallPolicyScheduleValidator) MarkdownDescription(ctx context.Context) string {
	return firewallPolicyScheduleValidator{}.Description(ctx)
}

func (firewallPolicyScheduleValidator) ValidateObject(
	ctx context.Context,
	req validator.ObjectRequest,
	resp *validator.ObjectResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var schedule firewallPolicyScheduleModel
	resp.Diagnostics.Append(
		req.ConfigValue.As(ctx, &schedule, basetypes.ObjectAsOptions{})...,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	if schedule.Mode.IsUnknown() {
		return
	}
	if schedule.Mode.IsNull() || schedule.Mode.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Missing Firewall Policy Schedule Mode",
			"A configured schedule must set mode.",
		)
		return
	}

	mode := schedule.Mode.ValueString()
	parseDate := func(value basetypes.StringValue, field string) (time.Time, bool) {
		if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
			return time.Time{}, false
		}
		parsed, err := time.Parse("2006-01-02", value.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Firewall Policy Schedule Date",
				fmt.Sprintf("%s must be a real calendar date in YYYY-MM-DD format: %v.", field, err),
			)
			return time.Time{}, false
		}
		return parsed, true
	}
	parseDate(schedule.Date, "date")
	dateStart, dateStartOK := parseDate(schedule.DateStart, "date_start")
	dateEnd, dateEndOK := parseDate(schedule.DateEnd, "date_end")
	if dateStartOK && dateEndOK && dateEnd.Before(dateStart) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Firewall Policy Schedule Date Range",
			"date_end must be on or after date_start.",
		)
	}
	normalizing := !schedule.Normalize.IsNull() && !schedule.Normalize.IsUnknown() &&
		schedule.Normalize.ValueBool()
	rejectStringWhenNormalizing := func(value basetypes.StringValue, field string) {
		if !normalizing || value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
			return
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Conflicting Normalized Firewall Policy Schedule",
			fmt.Sprintf("Schedule mode %s does not use %s; omit it when normalize is true.", mode, field),
		)
	}
	rejectDaysWhenNormalizing := func() {
		if !normalizing || schedule.RepeatOnDays.IsNull() || schedule.RepeatOnDays.IsUnknown() ||
			len(schedule.RepeatOnDays.Elements()) == 0 {
			return
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Conflicting Normalized Firewall Policy Schedule",
			fmt.Sprintf("Schedule mode %s does not use repeat_on_days; omit it when normalize is true.", mode),
		)
	}
	rejectTimeAllDayWhenNormalizing := func() {
		if !normalizing || schedule.TimeAllDay.IsNull() || schedule.TimeAllDay.IsUnknown() {
			return
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Conflicting Normalized Firewall Policy Schedule",
			fmt.Sprintf("Schedule mode %s does not use time_all_day; omit it when normalize is true.", mode),
		)
	}
	rejectTimeRangeForAllDay := func() {
		if !normalizing || schedule.TimeAllDay.IsNull() || schedule.TimeAllDay.IsUnknown() ||
			!schedule.TimeAllDay.ValueBool() {
			return
		}
		rejectStringWhenNormalizing(schedule.TimeRangeStart, "time_range_start with time_all_day")
		rejectStringWhenNormalizing(schedule.TimeRangeEnd, "time_range_end with time_all_day")
	}
	requireDate := func(value basetypes.StringValue, field string) bool {
		if value.IsUnknown() {
			return true
		}
		if value.IsNull() || value.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Firewall Policy Schedule",
				fmt.Sprintf("Schedule mode %s requires %s.", mode, field),
			)
			return false
		}
		return true
	}
	requireDays := func() bool {
		if schedule.RepeatOnDays.IsUnknown() {
			return true
		}
		if schedule.RepeatOnDays.IsNull() || len(schedule.RepeatOnDays.Elements()) == 0 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Firewall Policy Schedule",
				fmt.Sprintf("Schedule mode %s requires at least one repeat_on_days value.", mode),
			)
			return false
		}
		return true
	}
	requireTimeSelection := func() bool {
		if schedule.TimeAllDay.IsUnknown() {
			return true
		}
		if !schedule.TimeAllDay.IsNull() && schedule.TimeAllDay.ValueBool() {
			return true
		}
		if schedule.TimeRangeStart.IsUnknown() || schedule.TimeRangeEnd.IsUnknown() {
			return true
		}
		if schedule.TimeRangeStart.IsNull() || schedule.TimeRangeStart.ValueString() == "" ||
			schedule.TimeRangeEnd.IsNull() || schedule.TimeRangeEnd.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Firewall Policy Schedule",
				fmt.Sprintf(
					"Schedule mode %s requires time_all_day = true or both time_range_start and time_range_end.",
					mode,
				),
			)
			return false
		}
		return true
	}
	requireTimeAllDayFlag := func() bool {
		if schedule.TimeAllDay.IsUnknown() {
			return true
		}
		if schedule.TimeAllDay.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Firewall Policy Schedule",
				fmt.Sprintf("Schedule mode %s requires an explicit time_all_day value.", mode),
			)
			return false
		}
		return true
	}
	requireTimeRange := func() bool {
		if schedule.TimeRangeStart.IsUnknown() || schedule.TimeRangeEnd.IsUnknown() {
			return true
		}
		if schedule.TimeRangeStart.IsNull() || schedule.TimeRangeStart.ValueString() == "" ||
			schedule.TimeRangeEnd.IsNull() || schedule.TimeRangeEnd.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete Firewall Policy Schedule",
				fmt.Sprintf(
					"Schedule mode %s requires both time_range_start and time_range_end.",
					mode,
				),
			)
			return false
		}
		return true
	}
	rejectOneTimeAllDay := func() bool {
		if schedule.TimeAllDay.IsUnknown() {
			return true
		}
		if schedule.TimeAllDay.IsNull() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Incomplete One-Time Firewall Policy Schedule",
				"Schedule mode ONE_TIME_ONLY requires time_all_day = false and an explicit time range.",
			)
			return false
		}
		if !schedule.TimeAllDay.ValueBool() {
			return true
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Unsupported One-Time All-Day Schedule",
			"The controller requires an explicit time range for ONE_TIME_ONLY; time_all_day must be false.",
		)
		return false
	}

	switch mode {
	case "ALWAYS":
		rejectStringWhenNormalizing(schedule.Date, "date")
		rejectStringWhenNormalizing(schedule.DateStart, "date_start")
		rejectStringWhenNormalizing(schedule.DateEnd, "date_end")
		rejectDaysWhenNormalizing()
		rejectTimeAllDayWhenNormalizing()
		rejectStringWhenNormalizing(schedule.TimeRangeStart, "time_range_start")
		rejectStringWhenNormalizing(schedule.TimeRangeEnd, "time_range_end")
		return
	case "EVERY_DAY":
		rejectStringWhenNormalizing(schedule.Date, "date")
		rejectStringWhenNormalizing(schedule.DateStart, "date_start")
		rejectStringWhenNormalizing(schedule.DateEnd, "date_end")
		rejectDaysWhenNormalizing()
		rejectTimeRangeForAllDay()
		requireTimeAllDayFlag()
		requireTimeSelection()
	case "EVERY_WEEK":
		rejectStringWhenNormalizing(schedule.Date, "date")
		rejectStringWhenNormalizing(schedule.DateStart, "date_start")
		rejectStringWhenNormalizing(schedule.DateEnd, "date_end")
		rejectTimeRangeForAllDay()
		requireDays()
		requireTimeAllDayFlag()
		requireTimeSelection()
	case "ONE_TIME_ONLY":
		rejectStringWhenNormalizing(schedule.DateStart, "date_start")
		rejectStringWhenNormalizing(schedule.DateEnd, "date_end")
		rejectDaysWhenNormalizing()
		requireDate(schedule.Date, "date")
		if rejectOneTimeAllDay() {
			requireTimeRange()
		}
	case "CUSTOM":
		rejectStringWhenNormalizing(schedule.Date, "date")
		rejectTimeRangeForAllDay()
		requireDate(schedule.DateStart, "date_start")
		requireDate(schedule.DateEnd, "date_end")
		requireDays()
		requireTimeAllDayFlag()
		requireTimeSelection()
	}
}
