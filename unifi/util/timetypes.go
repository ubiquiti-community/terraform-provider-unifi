package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// This file bridges the UniFi API (which stores time values as integer counts of
// a fixed unit — almost always seconds, occasionally minutes) and the
// string-backed timetypes.GoDuration custom type used in the schema.
//
// It also provides the generic state-migration machinery used by resources that
// changed a numeric attribute (e.g. Int64 seconds) into a GoDuration string:
// the wire/state type changed from number to string, so prior state must be
// rewritten during a schema-version upgrade.

// DurationValue converts an integer count of unit into a GoDuration value
// (e.g. DurationValue(86400, time.Second) -> "24h0m0s").
func DurationValue(n int64, unit time.Duration) timetypes.GoDuration {
	return timetypes.NewGoDurationValue(time.Duration(n) * unit)
}

// DurationPtrValue is the *int64 form, mapping nil to a null GoDuration.
func DurationPtrValue(n *int64, unit time.Duration) timetypes.GoDuration {
	if n == nil {
		return timetypes.NewGoDurationNull()
	}
	return DurationValue(*n, unit)
}

// DurationUnits converts a GoDuration back to an integer count of unit for the
// API (e.g. DurationUnits(d, time.Second) -> whole seconds). Null/unknown -> 0.
func DurationUnits(d timetypes.GoDuration, unit time.Duration) int64 {
	if d.IsNull() || d.IsUnknown() {
		return 0
	}
	dur, diags := d.ValueGoDuration()
	if diags.HasError() {
		return 0
	}
	return int64(dur / unit)
}

// DurationUnitsPtr is the *int64 form, mapping null/unknown to nil.
func DurationUnitsPtr(d timetypes.GoDuration, unit time.Duration) *int64 {
	if d.IsNull() || d.IsUnknown() {
		return nil
	}
	v := DurationUnits(d, unit)
	return &v
}

// UpgradeDurationRawState rewrites prior raw state during a schema-version
// upgrade. It applies rewrite (which converts numeric duration fields to Go
// duration strings in place), reconciles the result against schemaType (filling
// attributes missing from older state with null and dropping unknown ones), then
// encodes it as a DynamicValue conforming to the current schema.
func UpgradeDurationRawState(
	schemaType tftypes.Type,
	rawJSON []byte,
	rewrite func(state map[string]any),
) (*tfprotov6.DynamicValue, error) {
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.UseNumber()

	var data map[string]any
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("decoding prior state: %w", err)
	}

	rewrite(data)

	reconciled := reconcileType(data, schemaType)

	newJSON, err := json.Marshal(reconciled)
	if err != nil {
		return nil, fmt.Errorf("encoding upgraded state: %w", err)
	}

	val, err := tftypes.ValueFromJSONWithOpts(newJSON, schemaType, tftypes.ValueFromJSONOpts{})
	if err != nil {
		return nil, fmt.Errorf("building upgraded value: %w", err)
	}

	dv, err := tfprotov6.NewDynamicValue(schemaType, val)
	if err != nil {
		return nil, fmt.Errorf("encoding dynamic value: %w", err)
	}
	return &dv, nil
}

// SetDurationField converts obj[key] from an integer count of unit into a Go
// duration string. It is a no-op when the key is absent or null, so it is safe
// to call against state written before the field existed.
func SetDurationField(obj map[string]any, key string, unit time.Duration) {
	raw, ok := obj[key]
	if !ok || raw == nil {
		return
	}
	n, ok := jsonToInt64(raw)
	if !ok {
		return
	}
	obj[key] = (time.Duration(n) * unit).String()
}

func jsonToInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, true
		}
		if f, err := x.Float64(); err == nil {
			return int64(f), true
		}
	case float64:
		return int64(x), true
	case int64:
		return x, true
	}
	return 0, false
}

// reconcileType walks a decoded JSON value against the target terraform type,
// keeping only attributes the type declares (older state may carry extras) and
// inserting null for attributes the type declares but the state lacks (older
// state may be missing newer attributes). Scalars pass through unchanged so the
// duration strings produced by rewrite survive.
func reconcileType(v any, typ tftypes.Type) any {
	switch t := typ.(type) {
	case tftypes.Object:
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		out := make(map[string]any, len(t.AttributeTypes))
		for name, at := range t.AttributeTypes {
			out[name] = reconcileType(m[name], at)
		}
		return out
	case tftypes.List:
		s, ok := v.([]any)
		if !ok {
			return v
		}
		for i := range s {
			s[i] = reconcileType(s[i], t.ElementType)
		}
		return s
	case tftypes.Set:
		s, ok := v.([]any)
		if !ok {
			return v
		}
		for i := range s {
			s[i] = reconcileType(s[i], t.ElementType)
		}
		return s
	case tftypes.Tuple:
		s, ok := v.([]any)
		if !ok {
			return v
		}
		for i := range s {
			if i < len(t.ElementTypes) {
				s[i] = reconcileType(s[i], t.ElementTypes[i])
			}
		}
		return s
	case tftypes.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return v
		}
		for k := range m {
			m[k] = reconcileType(m[k], t.ElementType)
		}
		return m
	default:
		return v
	}
}
