package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func Test_ruleListPlanModifiers(t *testing.T) {
	ctx := context.Background()
	set := func(v ...string) types.Set {
		elems := make([]attr.Value, len(v))
		for i, s := range v {
			elems[i] = types.StringValue(s)
		}
		return types.SetValueMust(types.StringType, elems)
	}
	null := types.SetNull(types.StringType)
	// A non-null raw state/plan so the modifier is not treated as create or destroy.
	raw := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})

	cases := []struct {
		name        string
		state, plan types.Set
		want        bool
	}{
		{"removed", set("a"), null, true},
		{"emptied", set("a"), set(), true},
		{"changed", set("a"), set("b"), false},
		{"unchanged", set("a"), set("a"), false},
		{"added", null, set("a"), false},
		{"still null", null, null, false},
		{"unknown", set("a"), types.SetUnknown(types.StringType), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.SetRequest{
				StateValue: tc.state,
				PlanValue:  tc.plan,
				State:      tfsdk.State{Raw: raw},
				Plan:       tfsdk.Plan{Raw: raw},
			}
			resp := &planmodifier.SetResponse{PlanValue: tc.plan}
			for _, m := range ruleListPlanModifiers() {
				m.PlanModifySet(ctx, req, resp)
			}
			if resp.RequiresReplace != tc.want {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}
