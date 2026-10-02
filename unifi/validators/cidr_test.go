package validators

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func TestCIDRValidator(t *testing.T) {
	tests := []struct {
		name string
		want validator.String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CIDRValidator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CIDRValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cidrValidator_Description(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    cidrValidator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Description(tt.args.ctx); got != tt.want {
				t.Errorf("cidrValidator.Description() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cidrValidator_MarkdownDescription(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    cidrValidator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MarkdownDescription(tt.args.ctx); got != tt.want {
				t.Errorf("cidrValidator.MarkdownDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cidrValidator_ValidateString(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  validator.StringRequest
		resp *validator.StringResponse
	}
	tests := []struct {
		name string
		v    cidrValidator
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.v.ValidateString(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
