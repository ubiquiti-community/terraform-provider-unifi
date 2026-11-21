package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/hwtypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// newClientReadResponse builds a read response for unifi_client with a
// non-null state and the given stored identity (nil for "no identity stored",
// which is what OpenTofu < 1.12 and Terraform < 1.12 send).
func newClientReadResponse(
	t *testing.T,
	storedIdentity map[string]tftypes.Value,
) *resource.ReadResponse {
	t.Helper()

	ctx := context.Background()
	r := &clientResource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	var identityResp resource.IdentitySchemaResponse
	r.IdentitySchema(ctx, resource.IdentitySchemaRequest{}, &identityResp)
	if identityResp.Diagnostics.HasError() {
		t.Fatalf("identity schema: %v", identityResp.Diagnostics)
	}

	identityType := identityResp.IdentitySchema.Type().TerraformType(ctx)
	identityRaw := tftypes.NewValue(identityType, nil)
	if storedIdentity != nil {
		identityRaw = tftypes.NewValue(identityType, storedIdentity)
	}

	return &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw: tftypes.NewValue(
				schemaResp.Schema.Type().TerraformType(ctx),
				tftypes.UnknownValue,
			),
		},
		Identity: &tfsdk.ResourceIdentity{
			Schema: identityResp.IdentitySchema,
			Raw:    identityRaw,
		},
	}
}

// TestRemoveResourceOnRead_noStoredIdentity reproduces the refresh of a client
// deleted on the controller from a state that carries no identity: the
// response must drop the state and still return an identity, otherwise the
// framework fails the read with "Missing Resource Identity After Read".
func TestRemoveResourceOnRead_noStoredIdentity(t *testing.T) {
	ctx := context.Background()
	resp := newClientReadResponse(t, nil)

	removeResourceOnRead(ctx, resp, clientIdentityModel{
		MAC: hwtypes.NewMACAddressValue("01:23:45:67:89:ab"),
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("state should have been removed, got %s", resp.State.Raw)
	}
	if resp.Identity.Raw.IsFullyNull() {
		t.Fatal("identity is still null: the framework would reject the read")
	}

	var identity clientIdentityModel
	if diags := resp.Identity.Get(ctx, &identity); diags.HasError() {
		t.Fatalf("reading identity back: %v", diags)
	}
	if got := identity.MAC.ValueString(); got != "01:23:45:67:89:ab" {
		t.Fatalf("identity mac = %q, want 01:23:45:67:89:ab", got)
	}
}

// TestRemoveResourceOnRead_keepsStoredIdentity checks that an identity already
// stored by the CLI is passed through untouched (the framework rejects any
// change of a non-null identity during refresh).
func TestRemoveResourceOnRead_keepsStoredIdentity(t *testing.T) {
	ctx := context.Background()
	resp := newClientReadResponse(t, map[string]tftypes.Value{
		"mac": tftypes.NewValue(tftypes.String, "aa:bb:cc:dd:ee:ff"),
	})
	before := resp.Identity.Raw.Copy()

	removeResourceOnRead(ctx, resp, clientIdentityModel{
		MAC: hwtypes.NewMACAddressValue("01:23:45:67:89:ab"),
	})

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("state should have been removed, got %s", resp.State.Raw)
	}
	if !resp.Identity.Raw.Equal(before) {
		t.Fatalf("stored identity changed: before %s, after %s", before, resp.Identity.Raw)
	}
}

// TestRemoveResourceOnRead_nilIdentity covers a read response without identity
// support (resp.Identity == nil): the helper must only remove the state.
func TestRemoveResourceOnRead_nilIdentity(t *testing.T) {
	ctx := context.Background()
	resp := newClientReadResponse(t, nil)
	resp.Identity = nil

	removeResourceOnRead(ctx, resp, clientIdentityModel{})

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Fatalf("state should have been removed, got %s", resp.State.Raw)
	}
}

// A refresh of an object deleted on the controller, such as every network,
// WAN and WLAN after a factory reset, must drop it from the state so the plan
// creates it again, with or without a stored identity (OpenTofu stores none).
// The controller answers a GET of a deleted ID with rc "ok" and no data.
func TestReadRemovesDeletedObject(t *testing.T) {
	ctx := context.Background()
	empty := `{"meta":{"rc":"ok"},"data":[]}`
	cases := map[string]string{
		"unifi_network":    "/api/s/default/rest/networkconf/gone",
		"unifi_wan":        "/api/s/default/rest/networkconf/gone",
		"unifi_vpn_client": "/api/s/default/rest/networkconf/gone",
		"unifi_vpn_server": "/api/s/default/rest/networkconf/gone",
		"unifi_wlan":       "/api/s/default/rest/wlanconf/gone",
	}
	responses := map[string]string{}
	for _, p := range cases {
		responses[p] = empty
	}
	srv := newConfiguredProviderServer(t, newFakeController(t, responses))
	schemas, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	identitySchemas, err := srv.GetResourceIdentitySchemas(
		ctx,
		&tfprotov6.GetResourceIdentitySchemasRequest{},
	)
	if err != nil {
		t.Fatalf("GetResourceIdentitySchemas: %v", err)
	}

	for typeName := range cases {
		for _, withIdentity := range []bool{false, true} {
			name := typeName + "/no identity"
			if withIdentity {
				name = typeName + "/stored identity"
			}
			t.Run(name, func(t *testing.T) {
				stateType := schemas.ResourceSchemas[typeName].ValueType()
				state, err := tfprotov6.NewDynamicValue(stateType, objectWithNulls(t, stateType,
					map[string]tftypes.Value{
						"id":   tftypes.NewValue(tftypes.String, "gone"),
						"site": tftypes.NewValue(tftypes.String, "default"),
						"name": tftypes.NewValue(tftypes.String, "deleted"),
					}))
				if err != nil {
					t.Fatalf("state: %v", err)
				}
				req := &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: &state}
				if withIdentity {
					identityType := identitySchemas.IdentitySchemas[typeName].ValueType()
					data, err := tfprotov6.NewDynamicValue(identityType, tftypes.NewValue(
						identityType, map[string]tftypes.Value{
							"id":   tftypes.NewValue(tftypes.String, "gone"),
							"site": tftypes.NewValue(tftypes.String, "default"),
						}))
					if err != nil {
						t.Fatalf("identity: %v", err)
					}
					req.CurrentIdentity = &tfprotov6.ResourceIdentityData{IdentityData: &data}
				}

				read, err := srv.ReadResource(ctx, req)
				if err != nil {
					t.Fatalf("ReadResource: %v", err)
				}
				failOnErrorDiagnostics(t, "read", read.Diagnostics)
				got, err := read.NewState.Unmarshal(stateType)
				if err != nil {
					t.Fatalf("new state: %v", err)
				}
				if !got.IsNull() {
					t.Errorf("new state = %v, want the resource removed", got)
				}
			})
		}
	}
}
