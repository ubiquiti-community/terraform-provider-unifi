package planmodifiers

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// onCreateCase is one row of the shared create-only default table: the
// configured value, whether prior state exists, the prior value, and what the
// default and keep-null variants must plan.
type onCreateCase[T any] struct {
	name     string
	config   T
	hasState bool
	prior    T
	want     T
	wantKeep T
}

func stateFor(hasState bool) tfsdk.State {
	if !hasState {
		return tfsdk.State{Raw: tftypes.Value{}}
	}
	return tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "existing")}
}

// The bool and int64 modifiers follow StringDefaultOnCreate exactly (#544):
// default on create, prior value on update, never over a configured value.
func TestBoolDefaultOnCreate(t *testing.T) {
	ctx := context.Background()
	null, unknown := types.BoolNull(), types.BoolUnknown()
	on, off := types.BoolValue(true), types.BoolValue(false)

	for _, tc := range []onCreateCase[types.Bool]{
		{"create with nothing configured gets the default", null, false, null, on, on},
		// The case that matters: an adopted object disabled on the controller
		// must not be re-enabled because `enabled` is undeclared.
		{"update keeps the controller value", null, true, off, off, off},
		{"a configured value is never overridden on create", off, false, null, off, off},
		{"a configured value is never overridden on update", off, true, on, off, off},
		{"an unknown configured value stays unknown", unknown, false, null, unknown, unknown},
		{"update with no prior value", null, true, null, on, null},
	} {
		for _, keep := range []bool{false, true} {
			m, want := BoolDefaultOnCreate(true), tc.want
			if keep {
				m, want = BoolDefaultOnCreateKeepNull(true), tc.wantKeep
			}
			t.Run(tc.name+keepSuffix(keep), func(t *testing.T) {
				req := planmodifier.BoolRequest{
					ConfigValue: tc.config,
					State:       stateFor(tc.hasState),
					StateValue:  tc.prior,
					PlanValue:   tc.config,
				}
				if tc.config.IsNull() {
					req.PlanValue = types.BoolUnknown()
				}
				resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
				m.PlanModifyBool(ctx, req, resp)
				if !resp.PlanValue.Equal(want) {
					t.Errorf("plan = %v, want %v", resp.PlanValue, want)
				}
			})
		}
	}
}

func TestInt64DefaultOnCreate(t *testing.T) {
	ctx := context.Background()
	null, unknown := types.Int64Null(), types.Int64Unknown()
	def, live, set := types.Int64Value(1812), types.Int64Value(11812), types.Int64Value(1645)

	for _, tc := range []onCreateCase[types.Int64]{
		{"create with nothing configured gets the default", null, false, null, def, def},
		{"update keeps the controller value", null, true, live, live, live},
		{"a configured value is never overridden on create", set, false, null, set, set},
		{"a configured value is never overridden on update", set, true, live, set, set},
		{"an unknown configured value stays unknown", unknown, false, null, unknown, unknown},
		{"update with no prior value", null, true, null, def, null},
	} {
		for _, keep := range []bool{false, true} {
			m, want := Int64DefaultOnCreate(1812), tc.want
			if keep {
				m, want = Int64DefaultOnCreateKeepNull(1812), tc.wantKeep
			}
			t.Run(tc.name+keepSuffix(keep), func(t *testing.T) {
				req := planmodifier.Int64Request{
					ConfigValue: tc.config,
					State:       stateFor(tc.hasState),
					StateValue:  tc.prior,
					PlanValue:   tc.config,
				}
				if tc.config.IsNull() {
					req.PlanValue = types.Int64Unknown()
				}
				resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}
				m.PlanModifyInt64(ctx, req, resp)
				if !resp.PlanValue.Equal(want) {
					t.Errorf("plan = %v, want %v", resp.PlanValue, want)
				}
			})
		}
	}
}

func keepSuffix(keep bool) string {
	if keep {
		return " (keep null)"
	}
	return ""
}

func TestTypedDefaultOnCreateDescriptions(t *testing.T) {
	ctx := context.Background()
	for name, d := range map[string]string{
		"bool":  BoolDefaultOnCreate(true).Description(ctx),
		"int64": Int64DefaultOnCreate(1812).Description(ctx),
	} {
		if d == "" {
			t.Errorf("%s: Description must not be empty: it surfaces in plan output", name)
		}
	}
}
