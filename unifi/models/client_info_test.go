package models

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestClientInfoObjectType_Equal(t *testing.T) {
	type args struct {
		o attr.Type
	}
	tests := []struct {
		name string
		tr   ClientInfoObjectType
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Equal(tt.args.o); got != tt.want {
				t.Errorf("ClientInfoObjectType.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectType_String(t *testing.T) {
	tests := []struct {
		name string
		tr   ClientInfoObjectType
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.String(); got != tt.want {
				t.Errorf("ClientInfoObjectType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectType_ValueFromObject(t *testing.T) {
	type args struct {
		ctx context.Context
		in  basetypes.ObjectValue
	}
	tests := []struct {
		name  string
		tr    ClientInfoObjectType
		args  args
		want  basetypes.ObjectValuable
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.tr.ValueFromObject(tt.args.ctx, tt.args.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectType.ValueFromObject() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ClientInfoObjectType.ValueFromObject() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestClientInfoObjectValue_Type(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Type(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectValue.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectValue_ValueType(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.ValueType(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoObjectValue.ValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoObjectValue_Equal(t *testing.T) {
	type args struct {
		o attr.Value
	}
	tests := []struct {
		name string
		v    ClientInfoObjectValue
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Equal(tt.args.o); got != tt.want {
				t.Errorf("ClientInfoObjectValue.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientInfoObjectType(t *testing.T) {
	tests := []struct {
		name string
		want ClientInfoObjectType
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientInfoObjectType(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientInfoObjectType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClientInfoObjectTypeFromData(t *testing.T) {
	type args struct {
		data unifi.ClientInfo
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			NewClientInfoObjectTypeFromData(tt.args.data)
		})
	}
}

func TestAttributeTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AttributeTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AttributeTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Attributes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Attributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoDataSourceSchema(t *testing.T) {
	tests := []struct {
		name string
		want map[string]schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoDataSourceSchema(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoDataSourceSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListAttribute(t *testing.T) {
	tests := []struct {
		name string
		want schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoListAttribute(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientListValue(t *testing.T) {
	type args struct {
		ctx            context.Context
		clientInfoList unifi.ClientList
		target         *types.List
	}
	tests := []struct {
		name string
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientListValue(tt.args.ctx, tt.args.clientInfoList, tt.args.target); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientListValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoAttrValues(t *testing.T) {
	type args struct {
		ctx        context.Context
		clientInfo *unifi.ClientInfo
	}
	tests := []struct {
		name string
		args args
		want map[string]attr.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoAttrValues(tt.args.ctx, tt.args.clientInfo); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoAttrValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoValue(t *testing.T) {
	type args struct {
		ctx        context.Context
		clientInfo *unifi.ClientInfo
		target     *ClientInfoObjectValue
	}
	tests := []struct {
		name string
		args args
		want diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClientInfoValue(tt.args.ctx, tt.args.clientInfo, tt.args.target); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
