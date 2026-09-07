package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func Test_radiusUserResource_MoveState(t *testing.T) {
	r := &radiusUserResource{}
	got := r.MoveState(context.Background())
	if len(got) != 2 {
		t.Fatalf("MoveState() returned %d movers, want 2", len(got))
	}
	for i, m := range got {
		if m.SourceSchema == nil {
			t.Fatalf("mover %d: expected SourceSchema to be non-nil", i)
		}
	}

	// The first mover decodes the flat v0 shape ...
	v0 := got[0].SourceSchema
	if v0.Version != 0 {
		t.Errorf("v0 mover SourceSchema.Version = %d, want 0", v0.Version)
	}
	for _, flat := range []string{"tunnel_type", "tunnel_medium_type", "tunnel_config_type"} {
		if _, ok := v0.Attributes[flat]; !ok {
			t.Errorf("v0 SourceSchema missing flat attribute %q", flat)
		}
	}
	if _, ok := v0.Attributes["tunnel"]; ok {
		t.Error("v0 SourceSchema must not declare the nested tunnel attribute")
	}

	// ... and the second the current (nested) shape.
	cur := got[1].SourceSchema
	if cur.Version != radiusUserSchemaVersion {
		t.Errorf(
			"current mover SourceSchema.Version = %d, want %d",
			cur.Version,
			radiusUserSchemaVersion,
		)
	}
	if _, ok := cur.Attributes["tunnel"]; !ok {
		t.Error("current SourceSchema missing the nested tunnel attribute")
	}
}

// Test_radiusUserResource_moveFromAccount_guards checks that both movers skip
// (no state, no diagnostics) whenever the request is not a move from this
// provider's unifi_account at the schema version they handle.
func Test_radiusUserResource_moveFromAccount_guards(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	targetType := schemaResp.Schema.Type().TerraformType(ctx)

	movers := map[string]func(
		context.Context,
		fwresource.MoveStateRequest,
		*fwresource.MoveStateResponse,
	){
		"v0":      r.moveFromAccountV0,
		"current": r.moveFromAccount,
	}

	tests := []struct {
		name string
		req  fwresource.MoveStateRequest
	}{
		{
			name: "wrong_source_type_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:        "unifi_other",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
			},
		},
		{
			name: "wrong_provider_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:        "unifi_account",
				SourceProviderAddress: "registry.terraform.io/someone-else/other",
			},
		},
		{
			// SourceState is nil, so the mover returns early after passing the
			// provider guard.
			name: "correct_provider_address_with_slashes_nil_state",
			req: fwresource.MoveStateRequest{
				SourceTypeName:        "unifi_account",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
			},
		},
		{
			// An empty address passes the provider guard; the nil state guard
			// then triggers.
			name: "empty_provider_address_nil_state",
			req: fwresource.MoveStateRequest{
				SourceTypeName:        "unifi_account",
				SourceProviderAddress: "",
			},
		},
		{
			// Neither mover handles a version it does not own.
			name: "unknown_schema_version_skipped",
			req: fwresource.MoveStateRequest{
				SourceTypeName:        "unifi_account",
				SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
				SourceSchemaVersion:   99,
			},
		},
	}
	for moverName, mover := range movers {
		for _, tt := range tests {
			t.Run(moverName+"/"+tt.name, func(t *testing.T) {
				resp := &fwresource.MoveStateResponse{
					TargetState: tfsdk.State{
						Schema: schemaResp.Schema,
						Raw:    tftypes.NewValue(targetType, nil),
					},
				}
				mover(ctx, tt.req, resp)
				if resp.Diagnostics.HasError() {
					t.Errorf("unexpected diagnostics error: %v", resp.Diagnostics)
				}
				if !resp.TargetState.Raw.IsNull() {
					t.Errorf(
						"mover must be skipped, but set target state: %v",
						resp.TargetState.Raw,
					)
				}
			})
		}
	}
}

// Test_radiusUserResource_moveFromAccountV0_nestsTunnel feeds a flat v0
// unifi_account state through the v0 mover and checks the target state carries
// the nested tunnel object.
func Test_radiusUserResource_moveFromAccountV0_nestsTunnel(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	current := schemaResp.Schema
	v0 := radiusUserSchemaV0(current)

	source := tfsdk.State{
		Schema: v0,
		Raw:    tftypes.NewValue(v0.Type().TerraformType(ctx), nil),
	}
	if d := source.Set(ctx, radiusUserV0Model{
		ID:               types.StringValue("acc-1"),
		Site:             types.StringValue("default"),
		Name:             types.StringValue("alice"),
		Password:         types.StringValue("secret"),
		TunnelType:       types.Int64Value(13),
		TunnelMediumType: types.Int64Value(6),
		NetworkID:        types.StringNull(),
		VLAN:             types.Int64Value(100),
		TunnelConfigType: types.StringValue("802.1x"),
		Timeouts:         timeoutsNullValue(),
	}); d.HasError() {
		t.Fatalf("building v0 source state: %v", d)
	}

	resp := &fwresource.MoveStateResponse{
		TargetState: tfsdk.State{
			Schema: current,
			Raw:    tftypes.NewValue(current.Type().TerraformType(ctx), nil),
		},
	}
	r.moveFromAccountV0(ctx, fwresource.MoveStateRequest{
		SourceTypeName:        "unifi_account",
		SourceProviderAddress: "registry.terraform.io/hashicorp/unifi",
		SourceSchemaVersion:   0,
		SourceState:           &source,
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("moveFromAccountV0: %v", resp.Diagnostics)
	}
	if resp.TargetState.Raw.IsNull() {
		t.Fatal("moveFromAccountV0 did not set target state")
	}

	var out radiusUserResourceModel
	if d := resp.TargetState.Get(ctx, &out); d.HasError() {
		t.Fatalf("decoding target state: %v", d)
	}
	if out.ID.ValueString() != "acc-1" || out.Name.ValueString() != "alice" ||
		out.VLAN.ValueInt64() != 100 || !out.NetworkID.IsNull() {
		t.Errorf("scalar attributes not carried across: %+v", out)
	}
	want := types.ObjectValueMust(radiusUserTunnelAttrTypes(), map[string]attr.Value{
		"type":        types.Int64Value(13),
		"medium_type": types.Int64Value(6),
		"config_type": types.StringValue("802.1x"),
	})
	if !out.Tunnel.Equal(want) {
		t.Errorf("tunnel = %v, want %v", out.Tunnel, want)
	}
}

// Test_radiusUserResource_moveFromAccount_copiesCurrentShape checks that a
// unifi_account state already at the current schema version is copied across
// verbatim, nested tunnel included.
func Test_radiusUserResource_moveFromAccount_copiesCurrentShape(t *testing.T) {
	ctx := context.Background()
	r := &radiusUserResource{}

	var schemaResp fwresource.SchemaResponse
	r.Schema(ctx, fwresource.SchemaRequest{}, &schemaResp)
	current := schemaResp.Schema
	currentType := current.Type().TerraformType(ctx)

	in := radiusUserResourceModel{
		ID:       types.StringValue("acc-2"),
		Site:     types.StringValue("default"),
		Name:     types.StringValue("bob"),
		Password: types.StringValue("secret"),
		Tunnel: types.ObjectValueMust(radiusUserTunnelAttrTypes(), map[string]attr.Value{
			"type":        types.Int64Value(3),
			"medium_type": types.Int64Value(6),
			"config_type": types.StringNull(),
		}),
		NetworkID: types.StringValue("net-1"),
		VLAN:      types.Int64Null(),
		Timeouts:  timeoutsNullValue(),
	}
	source := tfsdk.State{Schema: current, Raw: tftypes.NewValue(currentType, nil)}
	if d := source.Set(ctx, in); d.HasError() {
		t.Fatalf("building source state: %v", d)
	}

	resp := &fwresource.MoveStateResponse{
		TargetState: tfsdk.State{Schema: current, Raw: tftypes.NewValue(currentType, nil)},
	}
	r.moveFromAccount(ctx, fwresource.MoveStateRequest{
		SourceTypeName:        "unifi_account",
		SourceProviderAddress: "registry.terraform.io/ubiquiti-community/unifi",
		SourceSchemaVersion:   radiusUserSchemaVersion,
		SourceState:           &source,
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("moveFromAccount: %v", resp.Diagnostics)
	}
	if !resp.TargetState.Raw.Equal(source.Raw) {
		t.Errorf("target state = %v, want verbatim copy of %v", resp.TargetState.Raw, source.Raw)
	}
}
