package util

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// This file supports schemas that bundle formerly flat, prefixed attributes
// (e.g. lcm_brightness, lcm_idle_timeout) into nested objects (lcm = { ... }).
//
// Two halves:
//
//   - State upgrade: NestFields/EachObject rewrite decoded prior-state JSON so
//     the flat keys move under their new object, then UpgradeRawState encodes
//     the result against the current schema type.
//   - Runtime helpers: ObjectAs, ResolveUnknownObject and OverlayKnownObject
//     make nested types.Object values as easy to work with as the flat
//     attributes were.

// UpgradeRawState rewrites prior raw state during a schema-version upgrade. It
// applies rewrite in place, reconciles the result against schemaType (filling
// attributes missing from older state with null and dropping unknown ones),
// then encodes it as a DynamicValue conforming to the current schema.
//
// UpgradeDurationRawState is the same function under its historical name.
func UpgradeRawState(
	schemaType tftypes.Type,
	rawJSON []byte,
	rewrite func(state map[string]any),
) (*tfprotov6.DynamicValue, error) {
	return UpgradeDurationRawState(schemaType, rawJSON, rewrite)
}

// NestFields moves flat keys of obj into a nested map stored at obj[target].
// fields maps each old flat key to its new name inside the nested object.
// Keys absent from obj are skipped, so it is safe to call against state
// written before a field existed. When obj[target] already holds a map (from
// an earlier NestFields call building a deeper structure) the moved fields are
// merged into it. If none of the listed keys are present, obj[target] is left
// untouched so schema reconciliation fills it with null.
func NestFields(obj map[string]any, target string, fields map[string]string) {
	if obj == nil {
		return
	}
	nested, _ := obj[target].(map[string]any)
	moved := false
	for oldKey, newKey := range fields {
		raw, ok := obj[oldKey]
		if !ok {
			continue
		}
		if nested == nil {
			nested = map[string]any{}
		}
		nested[newKey] = raw
		delete(obj, oldKey)
		moved = true
	}
	if moved {
		obj[target] = nested
	}
}

// EachObject calls fn for every element of the JSON array stored at obj[key]
// that decodes as an object. Non-array values and non-object elements are
// ignored. It is the idiom for rewriting nested list/set blocks such as
// port_override during a state upgrade.
func EachObject(obj map[string]any, key string, fn func(map[string]any)) {
	list, ok := obj[key].([]any)
	if !ok {
		return
	}
	for _, elem := range list {
		if m, ok := elem.(map[string]any); ok {
			fn(m)
		}
	}
}

// WithObject calls fn with the object stored at obj[key] when it decodes as a
// map. It is the single-object counterpart of EachObject for rewriting fields
// inside an already-nested attribute (e.g. dhcp_server.dns_servers).
func WithObject(obj map[string]any, key string, fn func(map[string]any)) {
	if m, ok := obj[key].(map[string]any); ok {
		fn(m)
	}
}

// ObjectAs decodes a known, non-null types.Object into T. The boolean result
// is false (and T is the zero value) when the object is null or unknown, which
// lets write-side conversions skip an entire nested group with one check:
//
//	if lcm, ok, d := util.ObjectAs[deviceLcmModel](ctx, model.Lcm); ok { ... }
func ObjectAs[T any](
	ctx context.Context,
	obj types.Object,
) (T, bool, diag.Diagnostics) {
	var out T
	if obj.IsNull() || obj.IsUnknown() {
		return out, false, nil
	}
	diags := obj.As(ctx, &out, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return out, false, diags
	}
	return out, true, diags
}

// ResolveUnknownObject fills every unknown attribute of dst from src, recursing
// into nested objects. An unknown dst is replaced by src wholesale; a null dst
// is returned unchanged. It is the nested-object equivalent of "resolve
// Optional+Computed attributes the plan left unknown from the API response".
func ResolveUnknownObject(ctx context.Context, dst, src types.Object) types.Object {
	if dst.IsUnknown() {
		return src
	}
	if dst.IsNull() || src.IsNull() || src.IsUnknown() {
		return dst
	}
	attrs := dst.Attributes()
	srcAttrs := src.Attributes()
	out := make(map[string]attr.Value, len(attrs))
	changed := false
	for name, v := range attrs {
		s, ok := srcAttrs[name]
		switch {
		case !ok:
			out[name] = v
		case v.IsUnknown():
			out[name] = s
			changed = true
		default:
			if o, isObj := v.(types.Object); isObj {
				if so, isObj := s.(types.Object); isObj {
					r := ResolveUnknownObject(ctx, o, so)
					if !r.Equal(o) {
						changed = true
					}
					out[name] = r
					continue
				}
			}
			out[name] = v
		}
	}
	if !changed {
		return dst
	}
	return types.ObjectValueMust(dst.AttributeTypes(ctx), out)
}

// OverlayKnownObject returns applied with every attribute the practitioner
// set in planned (known and non-null) re-asserted on top, recursing into
// nested objects. A null or unknown planned object yields applied unchanged.
// It is used after a post-apply read to keep configured values that the
// controller applies asynchronously (e.g. LED overrides) from tripping the
// consistency check.
func OverlayKnownObject(ctx context.Context, planned, applied types.Object) types.Object {
	if planned.IsNull() || planned.IsUnknown() {
		return applied
	}
	if applied.IsNull() || applied.IsUnknown() {
		return planned
	}
	pAttrs := planned.Attributes()
	aAttrs := applied.Attributes()
	out := make(map[string]attr.Value, len(aAttrs))
	for name, a := range aAttrs {
		p, ok := pAttrs[name]
		if !ok || p.IsNull() || p.IsUnknown() {
			out[name] = a
			continue
		}
		if po, isObj := p.(types.Object); isObj {
			if ao, isObj := a.(types.Object); isObj {
				out[name] = OverlayKnownObject(ctx, po, ao)
				continue
			}
		}
		out[name] = p
	}
	return types.ObjectValueMust(applied.AttributeTypes(ctx), out)
}
