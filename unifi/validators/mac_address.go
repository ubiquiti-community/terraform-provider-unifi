package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// MACAddressValidator validates that a string is a valid MAC address.
func MACAddressValidator() validator.String {
	return &macAddressValidator{}
}

type macAddressValidator struct{}

func (v macAddressValidator) Description(ctx context.Context) string {
	return "value must be a valid MAC address"
}

func (v macAddressValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid MAC address"
}

func (v macAddressValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, err := net.ParseMAC(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid MAC Address",
			fmt.Sprintf("Value %q is not a valid MAC address: %s", value, err.Error()),
		)
	}
}
