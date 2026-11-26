package unifi

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// listInPriorOrder returns got as a list, in prior's order when prior holds
// the same elements. The controller does not keep the order of ID lists it
// treats as sets, so a reorder alone would plan a change or fail an apply with
// an inconsistent result.
func listInPriorOrder(
	ctx context.Context,
	prior types.List,
	got []string,
) (types.List, diag.Diagnostics) {
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorElems []string
		diags := prior.ElementsAs(ctx, &priorElems, false)
		if !diags.HasError() && sameElements(priorElems, got) {
			return prior, nil
		}
	}
	return types.ListValueFrom(ctx, types.StringType, got)
}

// sameElements reports whether a and b hold the same strings the same number
// of times, in any order.
func sameElements(a, b []string) bool {
	return slices.Equal(slices.Sorted(slices.Values(a)), slices.Sorted(slices.Values(b)))
}
