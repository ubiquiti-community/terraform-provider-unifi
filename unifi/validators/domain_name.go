package validators

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// DomainNameValidator validates that a string is a valid domain name.
func DomainNameValidator() validator.String {
	return &domainNameValidator{}
}

type domainNameValidator struct{}

var domainNameRegex = regexp.MustCompile(
	`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`,
)

func (v domainNameValidator) Description(ctx context.Context) string {
	return "value must be a valid domain name"
}

func (v domainNameValidator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid domain name"
}

func (v domainNameValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if len(value) > 253 || !domainNameRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Domain Name",
			fmt.Sprintf("Value %q is not a valid domain name.", value),
		)
	}
}
