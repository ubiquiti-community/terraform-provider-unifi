package validators

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// HostOrIPValidator validates that a string is an IPv4 address, an IPv6
// address or a fully qualified domain name (at least two labels, a top-level
// label that is not all digits, so a mistyped address such as "10.0.0.256" is
// rejected instead of passing as a host name).
func HostOrIPValidator() validator.String {
	return &hostOrIPValidator{}
}

type hostOrIPValidator struct{}

var numericLabelRegex = regexp.MustCompile(`^[0-9]+$`)

func (v hostOrIPValidator) Description(ctx context.Context) string {
	return "value must be a valid IPv4 address, IPv6 address or fully qualified domain name"
}

func (v hostOrIPValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v hostOrIPValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()
	if isIPAddress(value) || isFQDN(value) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Host",
		fmt.Sprintf(
			"Value %q is not a valid IPv4 address, IPv6 address or fully qualified domain name.",
			value,
		),
	)
}

func isIPAddress(value string) bool {
	return net.ParseIP(value) != nil
}

func isFQDN(value string) bool {
	if len(value) > 253 || !domainNameRegex.MatchString(value) {
		return false
	}
	labels := strings.Split(value, ".")
	return len(labels) >= 2 && !numericLabelRegex.MatchString(labels[len(labels)-1])
}
