package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestUpgradeDeviceIdentityV0 guards #502: version 0 identities were written
// as {"id"} by v0.55.0 and as {"mac"} by v0.56.0, and both must upgrade.
func TestUpgradeDeviceIdentityV0(t *testing.T) {
	ctx := context.Background()

	r := &deviceResource{}
	var schemaResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &schemaResp)

	if got := schemaResp.IdentitySchema.GetVersion(); got != 1 {
		t.Fatalf("identity schema version = %d, want 1", got)
	}
	if _, ok := r.UpgradeIdentity(ctx)[0]; !ok {
		t.Fatal("missing identity upgrader for version 0")
	}

	tests := []struct {
		name    string
		raw     string
		wantMAC hwtypes.MACAddress
	}{
		{
			name:    "v0.55.0 id-keyed identity",
			raw:     `{"id":"6ab5392029cec83d66d7de1a"}`,
			wantMAC: hwtypes.NewMACAddressNull(),
		},
		{
			name:    "v0.56.0 mac-keyed identity",
			raw:     `{"mac":"00:27:22:00:00:02"}`,
			wantMAC: hwtypes.NewMACAddressValue("00:27:22:00:00:02"),
		},
		{
			name:    "empty mac",
			raw:     `{"mac":""}`,
			wantMAC: hwtypes.NewMACAddressNull(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := fwresource.UpgradeIdentityRequest{
				RawIdentity: &tfprotov6.RawState{JSON: []byte(tt.raw)},
			}
			resp := fwresource.UpgradeIdentityResponse{
				Identity: &tfsdk.ResourceIdentity{Schema: schemaResp.IdentitySchema},
			}

			upgradeDeviceIdentityV0(ctx, req, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.Identity.Raw.IsNull() {
				t.Fatal("upgraded identity is null")
			}

			var got deviceIdentityModel
			if diags := resp.Identity.Get(ctx, &got); diags.HasError() {
				t.Fatalf("reading upgraded identity: %v", diags)
			}
			if !got.MAC.Equal(tt.wantMAC) {
				t.Errorf("mac = %s, want %s", got.MAC, tt.wantMAC)
			}
		})
	}
}
