package validators

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func TestIPv6Validator(t *testing.T) {
	tests := []struct {
		name string
		want validator.String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IPv6Validator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IPv6Validator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ipv6Validator_Description(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ipv6Validator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.Description(tt.args.ctx); got != tt.want {
				t.Errorf("ipv6Validator.Description() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ipv6Validator_MarkdownDescription(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		v    ipv6Validator
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.MarkdownDescription(tt.args.ctx); got != tt.want {
				t.Errorf("ipv6Validator.MarkdownDescription() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_ipv6Validator_ValidateString(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  validator.StringRequest
		resp *validator.StringResponse
	}
	tests := []struct {
		name string
		v    ipv6Validator
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
