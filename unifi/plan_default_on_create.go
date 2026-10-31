package unifi

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ModifyPlan helpers for Optional+Computed attributes that used to carry a
// static schema Default (#544).
//
// A schema Default cannot express "a default for new objects only": the
// framework applies defaults (TransformDefaults) before it runs plan
// modifiers, so for an attribute the configuration leaves out, the constant
// always replaces the controller's value, and UseStateForUnknown alongside it
// never fires. Every adopted object then planned the constant on its first
// apply. These helpers reproduce the old create-time value from ModifyPlan
// instead, and hold the prior state value on every later plan.
//
// planmodifiers.StringDefaultOnCreate covers string leaves the same way as an
// attribute plan modifier; these cover the bool, int64 and whole-object cases,
// and leaves of an object whose create-time shape is itself set here (an
// attribute modifier does not run under an unknown parent).

// planDefaultOnCreate plans an Optional+Computed attribute the configuration
// leaves unset: def when there is no prior value to keep (create, or a new
// element of a list block), otherwise the prior state value, even when that is
// null, since the controller may store no value at all. A configured value is
// never touched.
func planDefaultOnCreate[T attr.Value](
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	p path.Path,
	def T,
	hasPrior bool,
) {
	var configured T
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &configured)...)
	if resp.Diagnostics.HasError() || !configured.IsNull() {
		return
	}

	value := def
	if hasPrior {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, p, &value)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, value)...)
}

// planObjectOnCreate gives a nested group its create-time shape when the
// configuration omits it on create. On update the group carries no schema
// Default and its UseStateForUnknown keeps the controller's object.
func planObjectOnCreate(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
	p path.Path,
	shape types.Object,
) {
	if !req.State.Raw.IsNull() {
		return
	}
	var configured types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, p, &configured)...)
	if resp.Diagnostics.HasError() || !configured.IsNull() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, p, shape)...)
}
