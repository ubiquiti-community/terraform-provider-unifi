package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// IPv6Validator validates that a string is a valid IPv6 address.
func IPv6Validator() validator.String {
	return &ipv6Validator{}
}

type ipv6Validator struct{}

func (v ipv6Validator) Description(ctx context.Context) string {
	return "value must be a valid IPv6 address"
}

func (v ipv6Validator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid IPv6 address"
}

func (v ipv6Validator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil { // To4() returns nil for IPv6 addresses
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv6 Address",
			fmt.Sprintf("Value %q is not a valid IPv6 address.", value),
		)
	}
}
