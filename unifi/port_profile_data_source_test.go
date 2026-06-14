package unifi

import (
	"context"
	"reflect"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestNewPortProfileDataSource(t *testing.T) {
	tests := []struct {
		name string
		want fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewPortProfileDataSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPortProfileDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_portProfileDataSource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.MetadataRequest
		resp *fwdatasource.MetadataResponse
	}
	tests := []struct {
		name string
		d    *portProfileDataSource
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

func Test_portProfileDataSource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.SchemaRequest
		resp *fwdatasource.SchemaResponse
	}
	tests := []struct {
		name string
		d    *portProfileDataSource
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

func Test_portProfileDataSource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ConfigureRequest
		resp *fwdatasource.ConfigureResponse
	}
	tests := []struct {
		name string
		d    *portProfileDataSource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.d.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_portProfileDataSource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ReadRequest
		resp *fwdatasource.ReadResponse
	}
	tests := []struct {
		name string
		d    *portProfileDataSource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.d.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}
