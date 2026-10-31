package util

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ResolveUnknowns returns plan with every unknown value replaced by the value
// at the same path in prior. It returns an error when prior holds no known
// value at that path, for example for an unknown element of a set.
func ResolveUnknowns(plan, prior tftypes.Value) (tftypes.Value, error) {
	return tftypes.Transform(
		plan,
		func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			if v.IsKnown() {
				return v, nil
			}
			found, _, err := tftypes.WalkAttributePath(prior, p)
			if err != nil {
				return v, fmt.Errorf("unknown value has no prior value: %w", err)
			}
			resolved, ok := found.(tftypes.Value)
			if !ok || !resolved.IsKnown() {
				return v, errors.New("unknown value has no known prior value")
			}
			return resolved, nil
		},
	)
}
