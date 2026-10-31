package planmodifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BoolDefaultOnCreate is StringDefaultOnCreate for booleans: def for an unset
// attribute on create (and on update when the prior state holds no value),
// the prior state value otherwise.
//
// Use it for a bool go-unifi always serializes (no omitempty): leaving such an
// attribute unknown on create would send false, which is not what a new
// object has been created with (an `enabled` flag, typically).
func BoolDefaultOnCreate(def bool) planmodifier.Bool {
	return boolDefaultOnCreate{def: def}
}

// BoolDefaultOnCreateKeepNull is BoolDefaultOnCreate for an attribute the
// controller may store no value for: on update it holds the prior state value
// even when that is null (see StringDefaultOnCreateKeepNull).
func BoolDefaultOnCreateKeepNull(def bool) planmodifier.Bool {
	return boolDefaultOnCreate{def: def, keepNull: true}
}

type boolDefaultOnCreate struct {
	def      bool
	keepNull bool
}

func (m boolDefaultOnCreate) Description(_ context.Context) string {
	return fmt.Sprintf("defaults to %t on create; keeps the current value on update", m.def)
}

func (m boolDefaultOnCreate) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m boolDefaultOnCreate) PlanModifyBool(
	_ context.Context,
	req planmodifier.BoolRequest,
	resp *planmodifier.BoolResponse,
) {
	switch planOnCreate(req.ConfigValue, req.State.Raw.IsNull(), req.StateValue, m.keepNull) {
	case planDefault:
		resp.PlanValue = types.BoolValue(m.def)
	case planPrior:
		resp.PlanValue = req.StateValue
	}
}
