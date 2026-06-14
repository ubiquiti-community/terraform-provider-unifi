package unifi

import (
	"context"
	"reflect"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewDeprecatedAccountResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeprecatedAccountResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeprecatedAccountResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deprecatedAccountResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *deprecatedAccountResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deprecatedAccountResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *deprecatedAccountResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func TestNewDeprecatedAccountDataSource(t *testing.T) {
	tests := []struct {
		name string
		want fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDeprecatedAccountDataSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDeprecatedAccountDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deprecatedAccountDataSource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.MetadataRequest
		resp *fwdatasource.MetadataResponse
	}
	tests := []struct {
		name string
		d    *deprecatedAccountDataSource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.d.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_deprecatedAccountDataSource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.SchemaRequest
		resp *fwdatasource.SchemaResponse
	}
	tests := []struct {
		name string
		d    *deprecatedAccountDataSource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.d.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
