package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// IPv4Validator validates that a string is a valid IPv4 address.
func IPv4Validator() validator.String {
	return &ipv4Validator{}
}

// IPv4OrAnyValidator validates that a string is either a valid IPv4 address or
// the literal "any". The UniFi controller uses "any" to represent "no specific
// IP filter" (e.g. a port forward applying to all WAN addresses), and returns
// it verbatim from the API, so it must be accepted as a valid value.
func IPv4OrAnyValidator() validator.String {
	return &ipv4Validator{allowAny: true}
}

type ipv4Validator struct {
	allowAny bool
}

func (v ipv4Validator) Description(ctx context.Context) string {
	if v.allowAny {
		return `value must be a valid IPv4 address or "any"`
	}
	return "value must be a valid IPv4 address"
}

func (v ipv4Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipv4Validator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if v.allowAny && value == "any" {
		return
	}

	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		detail := fmt.Sprintf("Value %q is not a valid IPv4 address.", value)
		if v.allowAny {
			detail = fmt.Sprintf("Value %q is not a valid IPv4 address or \"any\".", value)
		}
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 Address",
			detail,
		)
	}
}
