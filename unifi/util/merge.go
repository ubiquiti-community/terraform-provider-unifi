package util

import (
	"context"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
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

// Client interface for merge operations.
// This allows the merger to work without a hard dependency on the unifi package.
type Client interface {
	GetSiteName() string
}

// ResourceMerger provides a generic merge pattern for UniFi resources.
// This ensures that all fields from existing resources are preserved
// when doing updates, as required by the UniFi API.
type ResourceMerger[T any] struct {
	site string
}

// NewResourceMerger creates a new ResourceMerger for the given type.
func NewResourceMerger[T any](client Client) *ResourceMerger[T] {
	return &ResourceMerger[T]{
		site: client.GetSiteName(),
	}
}

// MergeForUpdate performs a read-merge-update operation for UniFi resources.
// This is the core pattern that all UniFi resources should use.
func (rm *ResourceMerger[T]) MergeForUpdate(
	ctx context.Context,
	id string,
	planned *T,
	getFunc func(ctx context.Context, site, id string) (*T, error),
	updateFunc func(ctx context.Context, site string, resource *T) (*T, error),
) (*T, error) {
	// Get existing resource from API
	existing, err := getFunc(ctx, rm.site, id)
	if err != nil {
		return nil, err
	}

	// Merge planned changes with existing data
	merged := rm.mergeResources(existing, planned)

	// Update with merged data
	return updateFunc(ctx, rm.site, merged)
}

// mergeResources merges planned changes into existing resource data.
// This preserves all existing fields while applying planned changes.
func (rm *ResourceMerger[T]) mergeResources(existing, planned *T) *T {
	if existing == nil {
		return planned
	}
	if planned == nil {
		return existing
	}

	// Create a copy of existing to modify
	existingVal := reflect.ValueOf(existing).Elem()
	plannedVal := reflect.ValueOf(planned).Elem()

	// Create result as copy of existing
	resultType := existingVal.Type()
	result := reflect.New(resultType).Elem()
	result.Set(existingVal)

	// Merge fields from planned into result
	rm.mergeStructFields(result, plannedVal)

	if res, ok := result.Addr().Interface().(*T); ok {
		return res
	}
	return nil
}

// mergeStructFields recursively merges struct fields.
func (rm *ResourceMerger[T]) mergeStructFields(dest, source reflect.Value) {
	destType := dest.Type()

	for i := 0; i < dest.NumField(); i++ {
		destField := dest.Field(i)
		sourceField := source.Field(i)
		fieldType := destType.Field(i)

		if !destField.CanSet() {
			continue
		}

		// Skip internal/metadata fields that shouldn't be merged
		if rm.shouldSkipField(fieldType.Name) {
			continue
		}

		// Apply merge logic based on field type
		rm.mergeField(destField, sourceField)
	}
}

// mergeField applies merge logic for individual fields.
func (rm *ResourceMerger[T]) mergeField(dest, source reflect.Value) {
	switch source.Kind() {
	case reflect.String:
		// Only merge non-empty strings
		if str := source.String(); str != "" {
			dest.SetString(str)
		}
	case reflect.Bool:
		// Always merge boolean values as they have meaningful false states
		dest.SetBool(source.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Only merge non-zero integers (unless it's a meaningful zero)
		if val := source.Int(); val != 0 || rm.isZeroMeaningful(dest) {
			dest.SetInt(val)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val := source.Uint(); val != 0 || rm.isZeroMeaningful(dest) {
			dest.SetUint(val)
		}
	case reflect.Float32, reflect.Float64:
		if val := source.Float(); val != 0 || rm.isZeroMeaningful(dest) {
			dest.SetFloat(val)
		}
	case reflect.Slice:
		// For slices, replace if source has elements
		if !source.IsNil() && source.Len() > 0 {
			dest.Set(source)
		}
	case reflect.Map:
		// For maps, replace if source has elements
		if !source.IsNil() && source.Len() > 0 {
			dest.Set(source)
		}
	case reflect.Ptr:
		// For pointers, merge if source is not nil
		if !source.IsNil() {
			if dest.IsNil() {
				dest.Set(reflect.New(dest.Type().Elem()))
			}
			rm.mergeField(dest.Elem(), source.Elem())
		}
	case reflect.Struct:
		// Recursively merge struct fields
		rm.mergeStructFields(dest, source)
	default:
		// For other types, replace if not zero value
		if !source.IsZero() {
			dest.Set(source)
		}
	}
}

// shouldSkipField determines if a field should be skipped during merge.
func (rm *ResourceMerger[T]) shouldSkipField(fieldName string) bool {
	// Skip common metadata fields that shouldn't be merged
	skipFields := []string{
		"ID", "SiteID", "Rev", "CreateTime", "UpdateTime",
		"AttrHidden", "AttrNoDelete", "AttrNoEdit",
	}

	for _, skip := range skipFields {
		if strings.EqualFold(fieldName, skip) {
			return true
		}
	}

	return false
}

// isZeroMeaningful determines if a zero value is meaningful for a field.
// Some fields like VLAN ID=0 or Port=0 might be meaningful.
func (rm *ResourceMerger[T]) isZeroMeaningful(_ reflect.Value) bool {
	// For now, we'll be conservative and not merge zero values
	// This can be expanded based on specific field requirements
	return false
}

// WLAN-specific helper methods that use the generic merger.

// WLANMerger provides WLAN-specific merge operations.
type WLANMerger struct {
	*ResourceMerger[unifi.WLAN]
	client Client
}

// NewWLANMerger creates a new WLAN merger.
func NewWLANMerger(client Client) *WLANMerger {
	return &WLANMerger{
		ResourceMerger: NewResourceMerger[unifi.WLAN](client),
		client:         client,
	}
}

// UpdateWLAN performs the read-merge-update pattern for WLAN resources.
func (wm *WLANMerger) UpdateWLAN(
	ctx context.Context,
	id string,
	planned *unifi.WLAN,
	getFunc func(ctx context.Context, site, id string) (*unifi.WLAN, error),
	updateFunc func(ctx context.Context, site string, resource *unifi.WLAN) (*unifi.WLAN, error),
) (*unifi.WLAN, error) {
	return wm.MergeForUpdate(
		ctx,
		id,
		planned,
		getFunc,
		updateFunc,
	)
}

// Network-specific helper methods that use the generic merger.

// NetworkMerger provides Network-specific merge operations.
type NetworkMerger struct {
	*ResourceMerger[unifi.Network]
	client Client
}

// NewNetworkMerger creates a new Network merger.
func NewNetworkMerger(client Client) *NetworkMerger {
	return &NetworkMerger{
		ResourceMerger: NewResourceMerger[unifi.Network](client),
		client:         client,
	}
}

// UpdateNetwork performs the read-merge-update pattern for Network resources.
func (nm *NetworkMerger) UpdateNetwork(
	ctx context.Context,
	id string,
	planned *unifi.Network,
	getFunc func(ctx context.Context, site, id string) (*unifi.Network, error),
	updateFunc func(ctx context.Context, site string, resource *unifi.Network) (*unifi.Network, error),
) (*unifi.Network, error) {
	return nm.MergeForUpdate(
		ctx,
		id,
		planned,
		getFunc,
		updateFunc,
	)
}

// CreateMergerInterface defines the interface that resources can implement
// to use the generic merge pattern.
type CreateMergerInterface interface {
	GetResourceMerger() any
}

// UpdateMergerInterface defines the interface for resources that support
// the merge-update pattern.
type UpdateMergerInterface interface {
	MergeAndUpdate(ctx context.Context, id string, planned any) (any, error)
}
