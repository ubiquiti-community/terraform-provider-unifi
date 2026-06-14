package unifi

import (
	"context"
	"reflect"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"data.unifi_client.test",
						"mac",
						"01:23:45:67:89:ae",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client.test",
						"name",
						"tfacc-data-user",
					),
					resource.TestCheckResourceAttrSet("data.unifi_client.test", "id"),
				),
			},
		},
	})
}

func testAccClientDataSourceConfig_basic() string {
	return `
resource "unifi_client" "test" {
	name = "tfacc-data-user"
	mac  = "01:23:45:67:89:ae"
}

data "unifi_client" "test" {
	mac = unifi_client.test.mac
}
`
}

func TestNewClientDataSource(t *testing.T) {
	tests := []struct {
		name string
		want fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientDataSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientDataSource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.MetadataRequest
		resp *fwdatasource.MetadataResponse
	}
	tests := []struct {
		name string
		d    *clientDataSource
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

func Test_clientDataSource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.SchemaRequest
		resp *fwdatasource.SchemaResponse
	}
	tests := []struct {
		name string
		d    *clientDataSource
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

func Test_clientDataSource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ConfigureRequest
		resp *fwdatasource.ConfigureResponse
	}
	tests := []struct {
		name string
		d    *clientDataSource
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

func Test_clientDataSource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ReadRequest
		resp *fwdatasource.ReadResponse
	}
	tests := []struct {
		name string
		d    *clientDataSource
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
