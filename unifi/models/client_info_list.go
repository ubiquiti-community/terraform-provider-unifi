package models

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.ListTypable  = ClientInfoListType{}
	_ basetypes.ListValuable = ClientInfoListValue{}
)

type ClientInfoListType struct{}

// ApplyTerraform5AttributePathStep implements [basetypes.ListTypable].
func (t ClientInfoListType) ApplyTerraform5AttributePathStep(
	tftypes.AttributePathStep,
) (any, error) {
	panic("unimplemented")
}

// Equal implements [basetypes.ListTypable].
func (t ClientInfoListType) Equal(attr.Type) bool {
	panic("unimplemented")
}

// String implements [basetypes.ListTypable].
func (t ClientInfoListType) String() string {
	panic("unimplemented")
}

// TerraformType implements [basetypes.ListTypable].
func (t ClientInfoListType) TerraformType(context.Context) tftypes.Type {
	panic("unimplemented")
}

// ValueFromList implements [basetypes.ListTypable].
func (t ClientInfoListType) ValueFromList(
	context.Context,
	basetypes.ListValue,
) (basetypes.ListValuable, diag.Diagnostics) {
	panic("unimplemented")
}

// ValueFromTerraform implements [basetypes.ListTypable].
func (t ClientInfoListType) ValueFromTerraform(context.Context, tftypes.Value) (attr.Value, error) {
	panic("unimplemented")
}

// ValueType implements [basetypes.ListTypable].
func (t ClientInfoListType) ValueType(context.Context) attr.Value {
	panic("unimplemented")
}

type ClientInfoListValue struct{}

// Equal implements [basetypes.ListValuable].
func (c ClientInfoListValue) Equal(attr.Value) bool {
	panic("unimplemented")
}

// IsNull implements [basetypes.ListValuable].
func (c ClientInfoListValue) IsNull() bool {
	panic("unimplemented")
}

// IsUnknown implements [basetypes.ListValuable].
func (c ClientInfoListValue) IsUnknown() bool {
	panic("unimplemented")
}

// String implements [basetypes.ListValuable].
func (c ClientInfoListValue) String() string {
	panic("unimplemented")
}

// ToListValue implements [basetypes.ListValuable].
func (c ClientInfoListValue) ToListValue(
	ctx context.Context,
) (basetypes.ListValue, diag.Diagnostics) {
	panic("unimplemented")
}

// ToTerraformValue implements [basetypes.ListValuable].
func (c ClientInfoListValue) ToTerraformValue(context.Context) (tftypes.Value, error) {
	panic("unimplemented")
}

// Type implements [basetypes.ListValuable].
func (c ClientInfoListValue) Type(context.Context) attr.Type {
	panic("unimplemented")
}
