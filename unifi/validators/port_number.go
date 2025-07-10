package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PortNumberValidator validates that an int64 is a valid port number (1-65535).
func PortNumberValidator() validator.Int64 {
	return &portValidator{}
}

type portValidator struct{}

func (v portValidator) Description(ctx context.Context) string {
	return "value must be a valid port number (1-65535)"
}

func (v portValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid port number (1-65535)"
}

func (v portValidator) ValidateInt64(
	ctx context.Context,
	req validator.Int64Request,
	resp *validator.Int64Response,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueInt64()
	if value < 1 || value > 65535 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Port Number",
			fmt.Sprintf("Value %d is not a valid port number. Must be between 1 and 65535.", value),
		)
	}
}
