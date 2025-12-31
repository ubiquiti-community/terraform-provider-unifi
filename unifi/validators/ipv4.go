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

type ipv4Validator struct{}

func (v ipv4Validator) Description(ctx context.Context) string {
	return "value must be a valid IPv4 address"
}

func (v ipv4Validator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid IPv4 address"
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
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 Address",
			fmt.Sprintf("Value %q is not a valid IPv4 address.", value),
		)
	}
}
