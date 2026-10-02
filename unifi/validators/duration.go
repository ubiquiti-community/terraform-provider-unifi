package validators

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// GoDurationBetween validates that a timetypes.GoDuration value parses
// successfully and falls within the inclusive range [min, max]. Null/unknown
// values are skipped.
func GoDurationBetween(minDur, maxDur time.Duration) validator.String {
	return &goDurationBetweenValidator{min: minDur, max: maxDur}
}

type goDurationBetweenValidator struct {
	min, max time.Duration
}

func (v goDurationBetweenValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be a Go duration string between %s and %s", v.min, v.max)
}

func (v goDurationBetweenValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("value must be a Go duration string between `%s` and `%s`", v.min, v.max)
}

func (v goDurationBetweenValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	d, diags := timetypes.NewGoDurationValueFromString(req.ConfigValue.ValueString())
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	dur, dd := d.ValueGoDuration()
	resp.Diagnostics.Append(dd...)
	if dd.HasError() {
		return
	}
	if dur < v.min || dur > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf(
				"Duration %s is out of range. Must be between %s and %s.",
				dur, v.min, v.max,
			),
		)
	}
}

// GoDurationMultipleOf validates that a timetypes.GoDuration value is an exact
// multiple of unit (i.e. has no fractional component when expressed in unit).
// Null/unknown values are skipped.
func GoDurationMultipleOf(unit time.Duration) validator.String {
	return &goDurationMultipleOfValidator{unit: unit}
}

type goDurationMultipleOfValidator struct {
	unit time.Duration
}

func (v goDurationMultipleOfValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be a Go duration string that is a whole multiple of %s", v.unit)
}

func (v goDurationMultipleOfValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf(
		"value must be a Go duration string that is a whole multiple of `%s`",
		v.unit,
	)
}

func (v goDurationMultipleOfValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	d, diags := timetypes.NewGoDurationValueFromString(req.ConfigValue.ValueString())
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}
	dur, dd := d.ValueGoDuration()
	resp.Diagnostics.Append(dd...)
	if dd.HasError() {
		return
	}
	if v.unit > 0 && dur%v.unit != 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Duration",
			fmt.Sprintf(
				"Duration %s is not a whole multiple of %s. The controller stores this value with %s resolution, so smaller fractions would be silently truncated.",
				dur,
				v.unit,
				v.unit,
			),
		)
	}
}
