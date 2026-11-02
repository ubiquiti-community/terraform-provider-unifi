package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func Test_hostOrIPValidator_ValidateString(t *testing.T) {
	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"ipv4", types.StringValue("203.0.113.10"), false},
		{"ipv6", types.StringValue("2001:db8::1"), false},
		{"fqdn", types.StringValue("vpn.example.com"), false},
		{"fqdn with digits and hyphens", types.StringValue("vpn-1.home2.example.com"), false},
		{"empty", types.StringValue(""), true},
		{"single label", types.StringValue("vpn"), true},
		{"invalid ipv4", types.StringValue("10.0.0.256"), true},
		{"ipv4 with port", types.StringValue("203.0.113.10:51820"), true},
		{"invalid ipv6", types.StringValue("2001:db8::g"), true},
		{"ipv6 cidr", types.StringValue("2001:db8::/64"), true},
		{"url", types.StringValue("https://vpn.example.com"), true},
		{"leading hyphen", types.StringValue("-vpn.example.com"), true},
		{"empty label", types.StringValue("vpn..example.com"), true},
		{"label too long", types.StringValue(
			"a234567890123456789012345678901234567890123456789012345678901234.example.com",
		), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{Path: path.Root("test"), ConfigValue: tt.value}
			resp := &validator.StringResponse{}
			HostOrIPValidator().ValidateString(context.Background(), req, resp)
			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("HasError() = %v, want %v (%v)", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}
