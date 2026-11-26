package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func Test_isControllerID(t *testing.T) {
	for id, want := range map[string]bool{
		"6650cc00000000000000a001":              true,  // ObjectID, before Network 11
		"864031f8-6747-498b-9509-7fb704a144a4":  true,  // UUID, Network 11
		"6650CC00000000000000A001":              false, // the controller writes lowercase
		"864031F8-6747-498B-9509-7FB704A144A4":  false,
		"6650cc00000000000000a00":               false,
		"864031f86747498b95097fb704a144a4":      false,
		"864031f8-6747-498b-9509-7fb704a144a4x": false,
		"Office LAN":                            false,
		"":                                      false,
	} {
		if got := isControllerID(id); got != want {
			t.Errorf("isControllerID(%q) = %v, want %v", id, got, want)
		}
	}
}

// Every resource that imports either by ID or by name must take a Network 11
// UUID as an ID, not look it up as a name (#715).
func Test_importState_takesControllerIDs(t *testing.T) {
	c := &Client{Site: "default"}
	resources := map[string]harnessResource{
		"unifi_network":    &networkResource{client: c},
		"unifi_vpn_client": &vpnClientResource{client: c},
		"unifi_vpn_server": &vpnServerResource{client: c},
		"unifi_wan":        &wanResource{client: c},
		"unifi_wlan":       &wlanFrameworkResource{client: c},
	}
	for name, r := range resources {
		for _, id := range []string{
			"864031f8-6747-498b-9509-7fb704a144a4",
			"6650cc00000000000000a001",
		} {
			t.Run(name+"/"+id, func(t *testing.T) {
				resp := importStateResponse(t, r)
				r.ImportState(t.Context(), fwresource.ImportStateRequest{ID: id}, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("ImportState: %v", resp.Diagnostics)
				}
				var got types.String
				resp.Diagnostics.Append(
					resp.State.GetAttribute(t.Context(), path.Root("id"), &got)...)
				if got.ValueString() != id {
					t.Errorf("state id = %v, want %s", got, id)
				}
				var identityID types.String
				resp.Diagnostics.Append(
					resp.Identity.GetAttribute(t.Context(), path.Root("id"), &identityID)...)
				if identityID.ValueString() != id {
					t.Errorf("identity id = %v, want %s", identityID, id)
				}
			})
		}
		t.Run(name+"/name", func(t *testing.T) {
			resp := importStateResponse(t, r)
			r.ImportState(t.Context(), fwresource.ImportStateRequest{ID: "Office LAN"}, resp)
			var got types.String
			resp.Diagnostics.Append(resp.State.GetAttribute(t.Context(), path.Root("id"), &got)...)
			if !got.IsNull() {
				t.Errorf("state id = %v after importing a name, want null", got)
			}
		})
	}
}

func importStateResponse(t *testing.T, r harnessResource) *fwresource.ImportStateResponse {
	t.Helper()
	ctx := context.Background()
	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	var identityResp fwresource.IdentitySchemaResponse
	r.IdentitySchema(ctx, fwresource.IdentitySchemaRequest{}, &identityResp)
	return &fwresource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
		Identity: &tfsdk.ResourceIdentity{
			Schema: identityResp.IdentitySchema,
			Raw: tftypes.NewValue(
				identityResp.IdentitySchema.Type().TerraformType(ctx), nil),
		},
	}
}
