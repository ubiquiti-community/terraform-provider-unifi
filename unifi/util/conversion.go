package util

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringValueOrNull handles the conversion from string to Framework types
// Following the pattern from the migration guide to handle null vs empty consistently
func StringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// BoolValueOrNull handles the conversion from bool pointer to Framework types
func BoolValueOrNull(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// Int64ValueOrNull handles the conversion from int pointer to Framework types
func Int64ValueOrNull(i *int) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*i))
}

// ListValueOrNull handles the conversion from string slice to Framework list
// Lists should be null when empty as per the migration guide
func ListValueOrNull(items []string) (types.List, error) {
	if len(items) == 0 {
		return types.ListNull(types.StringType), nil
	}
	
	itemValues := make([]attr.Value, len(items))
	for i, item := range items {
		itemValues[i] = types.StringValue(item)
	}
	
	return types.ListValue(types.StringType, itemValues)
}