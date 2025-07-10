package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// CIDRValidator validates that a string is a valid CIDR notation.
func CIDRValidator() validator.String {
	return &cidrValidator{}
}

type cidrValidator struct{}

func (v cidrValidator) Description(ctx context.Context) string {
	return "value must be a valid CIDR notation"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid CIDR notation"
}

func (v cidrValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR Notation",
			fmt.Sprintf("Value %q is not a valid CIDR notation: %s", value, err.Error()),
		)
	}
}
