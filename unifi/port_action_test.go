package unifi

import (
	"context"
	"reflect"
	"testing"

	fwaction "github.com/hashicorp/terraform-plugin-framework/action"
)

func TestNewPortAction(t *testing.T) {
	tests := []struct {
		name string
		want fwaction.Action
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPortAction(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPortAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portAction_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwaction.MetadataRequest
		resp *fwaction.MetadataResponse
	}
	tests := []struct {
		name string
		a    *portAction
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_portAction_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwaction.SchemaRequest
		resp *fwaction.SchemaResponse
	}
	tests := []struct {
		name string
		a    *portAction
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_portAction_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwaction.ConfigureRequest
		resp *fwaction.ConfigureResponse
	}
	tests := []struct {
		name string
		a    *portAction
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_portAction_Invoke(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwaction.InvokeRequest
		resp *fwaction.InvokeResponse
	}
	tests := []struct {
		name string
		a    *portAction
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.a.Invoke(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
