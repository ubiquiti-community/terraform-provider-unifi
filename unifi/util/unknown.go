package util

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// ResolveUnknowns returns plan with every unknown value replaced by the value
// at the same path in prior. An unknown reached only through attributes of an
// object that is null in prior becomes null, as on Create. It returns an error
// when prior holds no known value at that path otherwise, for example for an
// unknown element of a set.
func ResolveUnknowns(plan, prior tftypes.Value) (tftypes.Value, error) {
	return tftypes.Transform(
		plan,
		func(p *tftypes.AttributePath, v tftypes.Value) (tftypes.Value, error) {
			if v.IsKnown() {
				return v, nil
			}
			found, rest, err := tftypes.WalkAttributePath(prior, p)
			if errors.Is(err, tftypes.ErrInvalidStep) && underNullObject(found, rest) {
				return tftypes.NewValue(v.Type(), nil), nil
			}
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

// underNullObject reports whether ancestor is a null object and every step
// left from it names an attribute.
func underNullObject(ancestor any, rest *tftypes.AttributePath) bool {
	v, ok := ancestor.(tftypes.Value)
	if !ok || !v.IsNull() {
		return false
	}
	if _, ok := v.Type().(tftypes.Object); !ok {
		return false
	}
	for _, step := range rest.Steps() {
		if _, ok := step.(tftypes.AttributeName); !ok {
			return false
		}
	}
	return true
}
