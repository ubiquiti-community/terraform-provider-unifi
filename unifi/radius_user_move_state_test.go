package unifi

import (
	"context"
	"reflect"
	"testing"

	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func Test_radiusUserResource_MoveState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *radiusUserResource
		args args
		want []fwresource.StateMover
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.MoveState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("radiusUserResource.MoveState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusUserResource_moveFromAccount(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MoveStateRequest
		resp *fwresource.MoveStateResponse
	}
	tests := []struct {
		name string
		r    *radiusUserResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.moveFromAccount(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
