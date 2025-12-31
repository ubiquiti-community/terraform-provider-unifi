package util

import (
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// MergeResourceData merges planned changes with existing API data
// This is necessary because UniFi APIs require complete objects on updates
// and don't support partial updates.
func MergeResourceData(existing, planned any) any {
	existingVal := reflect.ValueOf(existing)
	plannedVal := reflect.ValueOf(planned)

	// If either is nil, return the other
	if !existingVal.IsValid() {
		return planned
	}
	if !plannedVal.IsValid() {
		return existing
	}

	// Handle pointers
	if existingVal.Kind() == reflect.Ptr {
		if existingVal.IsNil() {
			return planned
		}
		existingVal = existingVal.Elem()
	}
	if plannedVal.Kind() == reflect.Ptr {
		if plannedVal.IsNil() {
			return existing
		}
		plannedVal = plannedVal.Elem()
	}

	// Create a copy of existing to modify
	result := reflect.New(existingVal.Type()).Elem()
	result.Set(existingVal)

	// Merge fields from planned into result
	mergeFields(result, plannedVal)

	return result.Interface()
}

// mergeFields recursively merges fields from source into dest.
func mergeFields(dest, source reflect.Value) {
	if dest.Type() != source.Type() {
		return
	}

	switch dest.Kind() {
	case reflect.Struct:
		for i := 0; i < dest.NumField(); i++ {
			destField := dest.Field(i)
			sourceField := source.Field(i)

			if !destField.CanSet() {
				continue
			}

			// Handle special Terraform types
			if isNullOrUnknown(sourceField) {
				continue // Don't merge null/unknown values
			}

			if sourceField.Type() == destField.Type() {
				if destField.Kind() == reflect.Struct {
					mergeFields(destField, sourceField)
				} else {
					// Check if this is a meaningful value to merge
					if shouldMergeValue(sourceField) {
						destField.Set(sourceField)
					}
				}
			}
		}
	case reflect.Slice:
		if !source.IsNil() && source.Len() > 0 {
			dest.Set(source)
		}
	case reflect.Map:
		if !source.IsNil() && source.Len() > 0 {
			dest.Set(source)
		}
	default:
		if shouldMergeValue(source) {
			dest.Set(source)
		}
	}
}

// shouldMergeValue determines if a value should be merged.
func shouldMergeValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() != ""
	case reflect.Bool:
		return true // Always merge boolean values
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return val.Float() != 0
	case reflect.Slice, reflect.Map:
		return !val.IsNil() && val.Len() > 0
	default:
		return !val.IsZero()
	}
}

// isNullOrUnknown checks if a Terraform Framework value is null or unknown.
func isNullOrUnknown(val reflect.Value) bool {
	// Check for types.String, types.Bool, types.Int64, etc.
	if val.Type().String() == "types.String" {
		if str, ok := val.Interface().(types.String); ok {
			return str.IsNull() || str.IsUnknown()
		}
	}
	if val.Type().String() == "types.Bool" {
		if b, ok := val.Interface().(types.Bool); ok {
			return b.IsNull() || b.IsUnknown()
		}
	}
	if val.Type().String() == "types.Int64" {
		if i, ok := val.Interface().(types.Int64); ok {
			return i.IsNull() || i.IsUnknown()
		}
	}
	if val.Type().String() == "types.Float64" {
		if f, ok := val.Interface().(types.Float64); ok {
			return f.IsNull() || f.IsUnknown()
		}
	}
	if val.Type().String() == "types.List" {
		if l, ok := val.Interface().(types.List); ok {
			return l.IsNull() || l.IsUnknown()
		}
	}
	if val.Type().String() == "types.Set" {
		if s, ok := val.Interface().(types.Set); ok {
			return s.IsNull() || s.IsUnknown()
		}
	}
	if val.Type().String() == "types.Map" {
		if m, ok := val.Interface().(types.Map); ok {
			return m.IsNull() || m.IsUnknown()
		}
	}

	return false
}
