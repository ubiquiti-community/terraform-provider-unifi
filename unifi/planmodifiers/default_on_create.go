// Package planmodifiers holds plan modifiers shared across the provider's
// resources.
package planmodifiers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringDefaultOnCreate supplies def for an unset attribute on create, and
// holds the prior state value on update.
//
// It exists because a schema `Default` cannot do this. The framework applies
// defaults before plan modifiers run (`TransformDefaults` precedes
// `SchemaModifyPlan` in server_planresourcechange.go), so by the time
// `UseStateForUnknown` is consulted the value is already the constant and no
// longer unknown. A `Default` therefore overwrites the controller's value on
// every resource the provider did not create, which is #544: an adopted
// firewall policy stored as `ip_version = "BOTH"` planned back to the schema's
// `"IPV4"`, silently dropping the IPv6 half of the rule.
//
// Dropping the default instead is the right answer whenever the controller
// supplies its own value (see #543). Use this modifier only where the
// controller *requires* the field, so create has to send something: omitting
// `ip_version` is rejected with `must not be null` on both POST and PUT.
func StringDefaultOnCreate(def string) planmodifier.String {
	return stringDefaultOnCreate{def: def}
}

type stringDefaultOnCreate struct {
	def string
}

func (m stringDefaultOnCreate) Description(_ context.Context) string {
	return fmt.Sprintf(
		"defaults to %q on create; keeps the current value on update", m.def,
	)
}

func (m stringDefaultOnCreate) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m stringDefaultOnCreate) PlanModifyString(
	_ context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	// The configuration asked for something: never override it.
	if !req.ConfigValue.IsNull() {
		return
	}

	// No prior state means this is a create: supply the default.
	if req.State.Raw.IsNull() {
		resp.PlanValue = types.StringValue(m.def)
		return
	}

	// On update, hold whatever the controller last reported, so an attribute
	// the configuration does not manage is not rewritten.
	if !req.StateValue.IsNull() {
		resp.PlanValue = req.StateValue
		return
	}

	// Prior state exists but holds no value (an attribute added to the schema
	// after the resource was created): fall back to the default rather than
	// leaving the plan unknown.
	resp.PlanValue = types.StringValue(m.def)
}
