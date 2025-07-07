package unifi

import (
	"context"
	"fmt"
	"net"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// IPv4Validator validates that a string is a valid IPv4 address
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

// IPv6Validator validates that a string is a valid IPv6 address
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

// CIDRValidator validates that a string is a valid CIDR notation
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

// MACAddressValidator validates that a string is a valid MAC address
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

// PortNumberValidator validates that an int64 is a valid port number (1-65535)
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

// DomainNameValidator validates that a string is a valid domain name
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
