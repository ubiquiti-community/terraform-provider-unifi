package planmodifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Int64DefaultOnCreate is StringDefaultOnCreate for integers: def for an
// unset attribute on create (and on update when the prior state holds no
// value), the prior state value otherwise.
func Int64DefaultOnCreate(def int64) planmodifier.Int64 {
	return int64DefaultOnCreate{def: def}
}

// Int64DefaultOnCreateKeepNull is Int64DefaultOnCreate for an attribute the
// controller may store no value for: on update it holds the prior state value
// even when that is null (see StringDefaultOnCreateKeepNull).
func Int64DefaultOnCreateKeepNull(def int64) planmodifier.Int64 {
	return int64DefaultOnCreate{def: def, keepNull: true}
}

type int64DefaultOnCreate struct {
	def      int64
	keepNull bool
}

func (m int64DefaultOnCreate) Description(_ context.Context) string {
	return fmt.Sprintf("defaults to %d on create; keeps the current value on update", m.def)
}

func (m int64DefaultOnCreate) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m int64DefaultOnCreate) PlanModifyInt64(
	_ context.Context,
	req planmodifier.Int64Request,
	resp *planmodifier.Int64Response,
) {
	switch planOnCreate(req.ConfigValue, req.State.Raw.IsNull(), req.StateValue, m.keepNull) {
	case planDefault:
		resp.PlanValue = types.Int64Value(m.def)
	case planPrior:
		resp.PlanValue = req.StateValue
	}
}
