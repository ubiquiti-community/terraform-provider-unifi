package planmodifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The whole point of this modifier is the update case: a schema Default would
// overwrite the controller's value there, because the framework applies
// defaults before plan modifiers run. #544.
func TestStringDefaultOnCreate(t *testing.T) {
	ctx := context.Background()
	m := StringDefaultOnCreate("IPV4")

	nullState := tfsdk.State{Raw: tftypes.Value{}}
	existingState := tfsdk.State{
		Raw: tftypes.NewValue(tftypes.String, "BOTH"),
	}

	for _, tc := range []struct {
		name   string
		config types.String
		state  tfsdk.State
		prior  types.String
		want   types.String
	}{
		{
			name:   "create with nothing configured gets the default",
			config: types.StringNull(),
			state:  nullState,
			prior:  types.StringNull(),
			want:   types.StringValue("IPV4"),
		},
		{
			// The case that matters: an adopted dual-stack policy must not
			// narrow to IPv4 just because the attribute is undeclared.
			name:   "update keeps the controller value",
			config: types.StringNull(),
			state:  existingState,
			prior:  types.StringValue("BOTH"),
			want:   types.StringValue("BOTH"),
		},
		{
			name:   "a configured value is never overridden on create",
			config: types.StringValue("IPV6"),
			state:  nullState,
			prior:  types.StringNull(),
			want:   types.StringValue("IPV6"),
		},
		{
			name:   "a configured value is never overridden on update",
			config: types.StringValue("IPV6"),
			state:  existingState,
			prior:  types.StringValue("BOTH"),
			want:   types.StringValue("IPV6"),
		},
		{
			// An attribute added to the schema after the resource was created:
			// state exists but holds nothing, so the default applies rather
			// than leaving the plan unknown.
			name:   "update with no prior value falls back to the default",
			config: types.StringNull(),
			state:  existingState,
			prior:  types.StringNull(),
			want:   types.StringValue("IPV4"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				ConfigValue: tc.config,
				State:       tc.state,
				StateValue:  tc.prior,
				PlanValue:   tc.config,
			}
			if tc.config.IsNull() {
				req.PlanValue = types.StringUnknown()
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}

			m.PlanModifyString(ctx, req, resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("modifier: %v", resp.Diagnostics)
			}
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan = %v, want %v", resp.PlanValue, tc.want)
			}
		})
	}
}

func TestStringDefaultOnCreateDescription(t *testing.T) {
	d := StringDefaultOnCreate("IPV4").Description(context.Background())
	if d == "" {
		t.Error("Description must not be empty: it surfaces in plan output")
	}
}
