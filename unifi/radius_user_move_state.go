package unifi

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure unifi_radius_user advertises move support so practitioners can migrate
// a deprecated unifi_account resource with a `moved` block.
var _ resource.ResourceWithMoveState = &radiusUserResource{}

// radiusUserV0Model is the flat (schema version 0) shape of unifi_account /
// unifi_radius_user state, before tunnel_type, tunnel_medium_type and
// tunnel_config_type moved under the nested `tunnel` object. It is only used to
// decode a v0 source state during a move.
type radiusUserV0Model struct {
	ID               types.String   `tfsdk:"id"`
	Site             types.String   `tfsdk:"site"`
	Name             types.String   `tfsdk:"name"`
	Password         types.String   `tfsdk:"password"`
	TunnelType       types.Int64    `tfsdk:"tunnel_type"`
	TunnelMediumType types.Int64    `tfsdk:"tunnel_medium_type"`
	NetworkID        types.String   `tfsdk:"network_id"`
	VLAN             types.Int64    `tfsdk:"vlan"`
	TunnelConfigType types.String   `tfsdk:"tunnel_config_type"`
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
}

// radiusUserSchemaV0 derives the v0 schema from the current one: the nested
// `tunnel` attribute is replaced by the three flat attributes it absorbed. Only
// the attribute types matter here (it decodes state, it never plans).
func radiusUserSchemaV0(current schema.Schema) schema.Schema {
	attrs := make(map[string]schema.Attribute, len(current.Attributes)+2)
	for name, a := range current.Attributes {
		if name != "tunnel" {
			attrs[name] = a
		}
	}
	attrs["tunnel_type"] = schema.Int64Attribute{Optional: true, Computed: true}
	attrs["tunnel_medium_type"] = schema.Int64Attribute{Optional: true, Computed: true}
	attrs["tunnel_config_type"] = schema.StringAttribute{Optional: true}

	return schema.Schema{
		Version:    0,
		Attributes: attrs,
		Blocks:     current.Blocks,
	}
}

// MoveState lets practitioners migrate the deprecated `unifi_account` resource to
// `unifi_radius_user` in place (via a `moved` block) instead of destroy/recreate.
//
//	moved {
//	  from = unifi_account.example
//	  to   = unifi_radius_user.example
//	}
//
// `unifi_account` is a deprecated alias backed by the same model and schema as
// this resource (see account_deprecated.go), so a source state at the current
// schema version is copied across verbatim. A v0 source state (written before
// the tunnel_* attributes were nested) is decoded against the flat v0 schema
// and re-shaped into the nested `tunnel` object. Each mover declares its source
// schema so the framework decodes MoveStateRequest.SourceState for us; the
// movers gate on SourceSchemaVersion themselves because the framework tolerates
// a schema/state mismatch when decoding.
func (r *radiusUserResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	v0 := radiusUserSchemaV0(schemaResp.Schema)

	return []resource.StateMover{
		{
			SourceSchema: &v0,
			StateMover:   r.moveFromAccountV0,
		},
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   r.moveFromAccount,
		},
	}
}

// accountMoveApplies reports whether req describes a move from this provider's
// unifi_account at the given schema version with a decoded source state. It is
// deliberately conservative: if the source provider, type or schema version is
// anything else, the mover returns without state so the framework treats it as
// skipped (and reports "implementation not found" rather than a bad move).
func accountMoveApplies(req resource.MoveStateRequest, version int64) bool {
	if req.SourceTypeName != "unifi_account" {
		return false
	}
	// Match on the provider type only (the last path segment), ignoring the host
	// and namespace. The address varies by context: the published provider is
	// registry.terraform.io/ubiquiti-community/unifi, but the acceptance-test
	// framework registers it as registry.terraform.io/hashicorp/unifi. Both are
	// this provider, so keying on namespace would wrongly skip the move.
	if seg := req.SourceProviderAddress; seg != "" {
		if idx := strings.LastIndex(seg, "/"); idx >= 0 {
			seg = seg[idx+1:]
		}
		if seg != "unifi" {
			return false
		}
	}
	if req.SourceSchemaVersion != version {
		return false
	}
	return req.SourceState != nil
}

// moveFromAccount handles a move from unifi_account state at the current
// schema version, which has exactly this resource's shape.
func (r *radiusUserResource) moveFromAccount(
	ctx context.Context,
	req resource.MoveStateRequest,
	resp *resource.MoveStateResponse,
) {
	if !accountMoveApplies(req, radiusUserSchemaVersion) {
		return
	}

	var data radiusUserResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &data)...)
}

// moveFromAccountV0 handles a move from unifi_account state written at schema
// version 0, re-shaping the flat tunnel_* attributes into the nested `tunnel`
// object.
func (r *radiusUserResource) moveFromAccountV0(
	ctx context.Context,
	req resource.MoveStateRequest,
	resp *resource.MoveStateResponse,
) {
	if !accountMoveApplies(req, 0) {
		return
	}

	var src radiusUserV0Model
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &src)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tunnel, d := types.ObjectValueFrom(ctx, radiusUserTunnelAttrTypes(), radiusUserTunnelModel{
		Type:       src.TunnelType,
		MediumType: src.TunnelMediumType,
		ConfigType: src.TunnelConfigType,
	})
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := radiusUserResourceModel{
		ID:        src.ID,
		Site:      src.Site,
		Name:      src.Name,
		Password:  src.Password,
		Tunnel:    tunnel,
		NetworkID: src.NetworkID,
		VLAN:      src.VLAN,
		Timeouts:  src.Timeouts,
	}
	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &data)...)
}
