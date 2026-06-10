package unifi

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure unifi_radius_user advertises move support so practitioners can migrate
// a deprecated unifi_account resource with a `moved` block.
var _ resource.ResourceWithMoveState = &radiusUserResource{}

// MoveState lets practitioners migrate the deprecated `unifi_account` resource to
// `unifi_radius_user` in place (via a `moved` block) instead of destroy/recreate.
//
//	moved {
//	  from = unifi_account.example
//	  to   = unifi_radius_user.example
//	}
//
// `unifi_account` is a deprecated alias backed by the same model and schema as
// this resource (see account_deprecated.go), so the source state can be copied
// across verbatim. We declare the source schema so the framework decodes it into
// MoveStateRequest.SourceState for us.
func (r *radiusUserResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	return []resource.StateMover{
		{
			SourceSchema: &schemaResp.Schema,
			StateMover:   r.moveFromAccount,
		},
	}
}

// moveFromAccount handles a move originating from this provider's unifi_account.
// It is deliberately conservative: if the source provider, type or schema version
// is anything else, it returns without state so the framework treats this mover
// as skipped (and reports "implementation not found" rather than a bad move).
func (r *radiusUserResource) moveFromAccount(
	ctx context.Context,
	req resource.MoveStateRequest,
	resp *resource.MoveStateResponse,
) {
	if req.SourceTypeName != "unifi_account" {
		return
	}
	// Ignore the hostname (registry.terraform.io vs registry.opentofu.org) and
	// match on namespace/type only.
	if !strings.HasSuffix(req.SourceProviderAddress, "ubiquiti-community/unifi") {
		return
	}
	if req.SourceSchemaVersion != 0 {
		return
	}
	if req.SourceState == nil {
		return
	}

	var data radiusUserResourceModel
	resp.Diagnostics.Append(req.SourceState.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.TargetState.Set(ctx, &data)...)
}
