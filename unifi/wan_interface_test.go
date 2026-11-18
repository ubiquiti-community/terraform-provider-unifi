package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The controller names WAN interfaces `wan` and `wan2` through `wan9`
// (`wan[2-9]?` in its field specs), and a gateway with three WAN ports has a
// `wan3`. An attribute that takes `wan` and `wan2` only cannot describe an
// entry on the third port.
func TestWANInterfaceAttributesAcceptEveryWAN(t *testing.T) {
	ctx := context.Background()

	var ddns, forward, tunnel fwresource.SchemaResponse
	(&dynamicDNSResource{}).Schema(ctx, fwresource.SchemaRequest{}, &ddns)
	(&portForwardResource{}).Schema(ctx, fwresource.SchemaRequest{}, &forward)
	(&siteToSiteVPNResource{}).Schema(ctx, fwresource.SchemaRequest{}, &tunnel)

	wan, ok := forward.Schema.Attributes["wan"].(schema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("unifi_port_forward.wan is %T", forward.Schema.Attributes["wan"])
	}
	destinations, ok := forward.Schema.Attributes["destination_ips"].(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf(
			"unifi_port_forward.destination_ips is %T",
			forward.Schema.Attributes["destination_ips"],
		)
	}

	for _, tc := range []struct {
		name      string
		attribute schema.Attribute
		// both is a value of unifi_port_forward.wan.interface only.
		both bool
	}{
		{"unifi_dynamic_dns.interface", ddns.Schema.Attributes["interface"], false},
		{"unifi_port_forward.wan.interface", wan.Attributes["interface"], true},
		{
			"unifi_port_forward.destination_ips.interface",
			destinations.NestedObject.Attributes["interface"],
			false,
		},
		{"unifi_site_to_site_vpn.interface", tunnel.Schema.Attributes["interface"], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			str, ok := tc.attribute.(schema.StringAttribute)
			if !ok {
				t.Fatalf("attribute is %T", tc.attribute)
			}
			for value, valid := range map[string]bool{
				"wan":   true,
				"wan2":  true,
				"wan3":  true,
				"wan9":  true,
				"both":  tc.both,
				"wan10": false,
				"lan":   false,
			} {
				var resp validator.StringResponse
				for _, v := range str.Validators {
					v.ValidateString(ctx, validator.StringRequest{
						Path:        path.Root("interface"),
						ConfigValue: types.StringValue(value),
					}, &resp)
				}
				if got := !resp.Diagnostics.HasError(); got != valid {
					t.Errorf("%q accepted = %v, want %v: %v", value, got, valid, resp.Diagnostics)
				}
			}
		})
	}
}
