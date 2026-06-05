package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runIPv4Validator(v validator.String, value types.String) bool {
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: value,
	}, resp)
	return !resp.Diagnostics.HasError()
}

func TestIPv4Validator(t *testing.T) {
	v := IPv4Validator()

	cases := []struct {
		name  string
		value types.String
		valid bool
	}{
		{"valid IPv4", types.StringValue("192.168.1.1"), true},
		{"any rejected", types.StringValue("any"), false},
		{"IPv6 rejected", types.StringValue("fe80::1"), false},
		{"garbage", types.StringValue("not-an-ip"), false},
		{"empty", types.StringValue(""), false},
		{"null skips", types.StringNull(), true},
		{"unknown skips", types.StringUnknown(), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runIPv4Validator(v, tc.value)
			if got != tc.valid {
				t.Fatalf("got valid=%v, want %v for %q", got, tc.valid, tc.value.ValueString())
			}
		})
	}
}

func TestIPv4OrAnyValidator(t *testing.T) {
	v := IPv4OrAnyValidator()

	cases := []struct {
		name  string
		value types.String
		valid bool
	}{
		{"valid IPv4", types.StringValue("203.0.113.4"), true},
		{"any accepted", types.StringValue("any"), true},
		// controller stores/returns lowercase "any"; uppercase must be rejected
		{"uppercase ANY rejected", types.StringValue("ANY"), false},
		{"IPv6 rejected", types.StringValue("fe80::1"), false},
		{"garbage", types.StringValue("not-an-ip"), false},
		{"null skips", types.StringNull(), true},
		{"unknown skips", types.StringUnknown(), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runIPv4Validator(v, tc.value)
			if got != tc.valid {
				t.Fatalf("got valid=%v, want %v for %q", got, tc.valid, tc.value.ValueString())
			}
		})
	}
}
