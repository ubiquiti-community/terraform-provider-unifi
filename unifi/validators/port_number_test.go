package validators

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func TestPortNumberValidator(t *testing.T) {
	tests := []struct {
		name string
		want validator.Int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PortNumberValidator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PortNumberValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portValidator_Description(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    portValidator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Description(tt.args.ctx); got != tt.want {
				t.Errorf("portValidator.Description() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portValidator_MarkdownDescription(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    portValidator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MarkdownDescription(tt.args.ctx); got != tt.want {
				t.Errorf("portValidator.MarkdownDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portValidator_ValidateInt64(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  validator.Int64Request
		resp *validator.Int64Response
	}
	tests := []struct {
		name string
		v    portValidator
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.v.ValidateInt64(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
