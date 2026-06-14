package models

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestClientInfoListType_ApplyTerraform5AttributePathStep(t *testing.T) {
	type args struct {
		in0 tftypes.AttributePathStep
	}
	tests := []struct {
		name    string
		tr      ClientInfoListType
		args    args
		want    any
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tr.ApplyTerraform5AttributePathStep(tt.args.in0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientInfoListType.ApplyTerraform5AttributePathStep() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListType.ApplyTerraform5AttributePathStep() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListType_Equal(t *testing.T) {
	type args struct {
		in0 attr.Type
	}
	tests := []struct {
		name string
		tr   ClientInfoListType
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Equal(tt.args.in0); got != tt.want {
				t.Errorf("ClientInfoListType.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListType_String(t *testing.T) {
	tests := []struct {
		name string
		tr   ClientInfoListType
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.String(); got != tt.want {
				t.Errorf("ClientInfoListType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListType_TerraformType(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name string
		tr   ClientInfoListType
		args args
		want tftypes.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.TerraformType(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListType.TerraformType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListType_ValueFromList(t *testing.T) {
	type args struct {
		in0 context.Context
		in1 basetypes.ListValue
	}
	tests := []struct {
		name  string
		tr    ClientInfoListType
		args  args
		want  basetypes.ListValuable
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.tr.ValueFromList(tt.args.in0, tt.args.in1)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListType.ValueFromList() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ClientInfoListType.ValueFromList() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestClientInfoListType_ValueFromTerraform(t *testing.T) {
	type args struct {
		in0 context.Context
		in1 tftypes.Value
	}
	tests := []struct {
		name    string
		tr      ClientInfoListType
		args    args
		want    attr.Value
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.tr.ValueFromTerraform(tt.args.in0, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientInfoListType.ValueFromTerraform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListType.ValueFromTerraform() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListType_ValueType(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name string
		tr   ClientInfoListType
		args args
		want attr.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.ValueType(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListType.ValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_Equal(t *testing.T) {
	type args struct {
		in0 attr.Value
	}
	tests := []struct {
		name string
		c    ClientInfoListValue
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Equal(tt.args.in0); got != tt.want {
				t.Errorf("ClientInfoListValue.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_IsNull(t *testing.T) {
	tests := []struct {
		name string
		c    ClientInfoListValue
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.IsNull(); got != tt.want {
				t.Errorf("ClientInfoListValue.IsNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_IsUnknown(t *testing.T) {
	tests := []struct {
		name string
		c    ClientInfoListValue
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.IsUnknown(); got != tt.want {
				t.Errorf("ClientInfoListValue.IsUnknown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_String(t *testing.T) {
	tests := []struct {
		name string
		c    ClientInfoListValue
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("ClientInfoListValue.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_ToListValue(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name  string
		c     ClientInfoListValue
		args  args
		want  basetypes.ListValue
		want1 diag.Diagnostics
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.c.ToListValue(tt.args.ctx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListValue.ToListValue() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("ClientInfoListValue.ToListValue() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestClientInfoListValue_ToTerraformValue(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name    string
		c       ClientInfoListValue
		args    args
		want    tftypes.Value
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.ToTerraformValue(tt.args.in0)
			if (err != nil) != tt.wantErr {
				t.Errorf("ClientInfoListValue.ToTerraformValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListValue.ToTerraformValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClientInfoListValue_Type(t *testing.T) {
	type args struct {
		in0 context.Context
	}
	tests := []struct {
		name string
		c    ClientInfoListValue
		args args
		want attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Type(tt.args.in0); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ClientInfoListValue.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}
