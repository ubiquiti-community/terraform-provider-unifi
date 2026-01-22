package util

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConvertStringToAPIValue converts a Terraform types.String to a Go string pointer
// Returns nil if the value is null, empty string if unknown, and the actual value otherwise.
func ConvertStringToAPIValue(val types.String) *string {
	if val.IsNull() {
		return nil
	}
	if val.IsUnknown() {
		empty := ""
		return &empty
	}
	str := val.ValueString()
	return &str
}

// ConvertStringFromAPIValue converts an API string pointer to types.String.
func ConvertStringFromAPIValue(val *string) types.String {
	if val == nil {
		return types.StringNull()
	}
	return types.StringValue(*val)
}

// ConvertBoolToAPIValue converts a Terraform types.Bool to a Go bool pointer.
func ConvertBoolToAPIValue(val types.Bool) *bool {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}
	v := val.ValueBool()
	return &v
}

// ConvertBoolFromAPIValue converts an API bool pointer to types.Bool.
func ConvertBoolFromAPIValue(val *bool) types.Bool {
	if val == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*val)
}

// ConvertInt64ToAPIValue converts a Terraform types.Int64 to a Go int64 pointer.
func ConvertInt64ToAPIValue(val types.Int64) *int64 {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}
	v := val.ValueInt64()
	return &v
}

// ConvertInt64FromAPIValue converts an API int64 pointer to types.Int64.
func ConvertInt64FromAPIValue(val *int64) types.Int64 {
	return types.Int64PointerValue(val)
}

// ConvertStringSliceToAPIValue converts a Terraform types.List of strings to a Go string slice.
func ConvertStringSliceToAPIValue(ctx context.Context, val types.List) []string {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}

	elements := val.Elements()
	result := make([]string, len(elements))

	for i, elem := range elements {
		if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
			result[i] = str.ValueString()
		}
	}

	return result
}

// ConvertStringSliceFromAPIValue converts an API string slice to types.List.
func ConvertStringSliceFromAPIValue(ctx context.Context, vals []string) types.List {
	if vals == nil {
		return types.ListNull(types.StringType)
	}

	elements := make([]attr.Value, len(vals))
	for i, val := range vals {
		elements[i] = types.StringValue(val)
	}

	list, _ := types.ListValue(types.StringType, elements)
	return list
}

// ConvertMapStringToAPIValue converts a Terraform types.Map to a Go map[string]string.
func ConvertMapStringToAPIValue(ctx context.Context, val types.Map) map[string]string {
	if val.IsNull() || val.IsUnknown() {
		return nil
	}

	elements := val.Elements()
	result := make(map[string]string, len(elements))

	for key, elem := range elements {
		if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
			result[key] = str.ValueString()
		}
	}

	return result
}

// ConvertMapStringFromAPIValue converts an API map[string]string to types.Map.
func ConvertMapStringFromAPIValue(ctx context.Context, vals map[string]string) types.Map {
	if vals == nil {
		return types.MapNull(types.StringType)
	}

	elements := make(map[string]attr.Value, len(vals))
	for key, val := range vals {
		elements[key] = types.StringValue(val)
	}

	mapVal, _ := types.MapValue(types.StringType, elements)
	return mapVal
}

// SafeStringValue returns the string value or empty string if null/unknown.
func SafeStringValue(val types.String) string {
	if val.IsNull() || val.IsUnknown() {
		return ""
	}
	return val.ValueString()
}

// SafeBoolValue returns the bool value or false if null/unknown.
func SafeBoolValue(val types.Bool) bool {
	if val.IsNull() || val.IsUnknown() {
		return false
	}
	return val.ValueBool()
}

// SafeInt64Value returns the int64 value or 0 if null/unknown.
func SafeInt64Value(val types.Int64) int64 {
	if val.IsNull() || val.IsUnknown() {
		return 0
	}
	return val.ValueInt64()
}

// StringValueOrNull returns types.StringValue if not empty, otherwise types.StringNull.
func StringValueOrNull(val string) types.String {
	if val == "" {
		return types.StringNull()
	}
	return types.StringValue(val)
}

func Ptr[T any](in T) *T {
	return &in
}
