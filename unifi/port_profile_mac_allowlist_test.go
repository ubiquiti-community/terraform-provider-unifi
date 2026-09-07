package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// Port security with an empty allowlist is how the controller stores a port whose
// Port State is Disabled: security is on and no MAC may pass. A configuration that
// asks for that has to survive the read, or Create and Update both fail with
// "provider produced inconsistent result after apply".
//
// Unit rather than acceptance test: the conversion is what regresses, and it needs
// no controller.
func TestPortProfileEmptyMacAllowlistIsPreserved(t *testing.T) {
	ctx := context.Background()
	r := &portProfileResource{}

	emptySet, d := types.SetValueFrom(ctx, types.StringType, []string{})
	if d.HasError() {
		t.Fatalf("building the empty set: %v", d)
	}

	tests := []struct {
		name     string
		inModel  types.Set
		fromAPI  []string
		wantNull bool
		wantLen  int
	}{
		{"explicit empty is kept", emptySet, nil, false, 0},
		{"unset stays null", types.SetNull(types.StringType), nil, true, 0},
		{
			"api addresses are adopted",
			types.SetNull(types.StringType),
			[]string{"aa:bb:cc:dd:ee:ff"},
			false,
			1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			portSecurity, d := types.ObjectValueFrom(ctx, portProfilePortSecurityAttrTypes(),
				portProfilePortSecurityModel{
					Enabled:    types.BoolValue(true),
					MACAddress: tc.inModel,
				})
			if d.HasError() {
				t.Fatalf("building port_security: %v", d)
			}
			model := &portProfileResourceModel{
				Name:         types.StringValue("test"),
				PortSecurity: portSecurity,
			}
			api := &unifi.PortProfile{
				Name:                   "test",
				PortSecurityEnabled:    true,
				PortSecurityMACAddress: tc.fromAPI,
			}
			if d := r.portProfileToModel(ctx, api, model, "default"); d.HasError() {
				t.Fatalf("conversion: %v", d)
			}
			var ps portProfilePortSecurityModel
			if d := model.PortSecurity.As(ctx, &ps, basetypes.ObjectAsOptions{}); d.HasError() {
				t.Fatalf("extracting port_security: %v", d)
			}
			got := ps.MACAddress
			if got.IsNull() != tc.wantNull {
				t.Fatalf("port_security.mac_address null = %v, want %v", got.IsNull(), tc.wantNull)
			}
			if !tc.wantNull && len(got.Elements()) != tc.wantLen {
				t.Errorf(
					"port_security.mac_address has %d elements, want %d",
					len(got.Elements()),
					tc.wantLen,
				)
			}
		})
	}
}
