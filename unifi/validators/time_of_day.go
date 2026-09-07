package validators

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// timeOfDayRegex matches a 24-hour clock time in HH:MM form, the wire format
// the controller uses for schedule boundaries and LCM night mode.
var timeOfDayRegex = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d$`)

// TimeOfDay validates that a string is a 24-hour clock time in `HH:MM` format
// (e.g. `07:30`, `22:00`). Null/unknown values are skipped.
func TimeOfDay() validator.String {
	return timeOfDayValidator{}
}

type timeOfDayValidator struct{}

func (timeOfDayValidator) Description(context.Context) string {
	return "value must be a 24-hour clock time in HH:MM format"
}

func (v timeOfDayValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a 24-hour clock time in `HH:MM` format"
}

func (timeOfDayValidator) ValidateString(
	_ context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if timeOfDayRegex.MatchString(req.ConfigValue.ValueString()) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Time Of Day",
		"Value "+req.ConfigValue.ValueString()+" must use 24-hour HH:MM format (e.g. 07:30, 22:00).",
	)
}
