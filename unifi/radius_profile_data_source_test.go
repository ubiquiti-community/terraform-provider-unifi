package unifi

import (
	"context"
	"reflect"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusProfileDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-ds",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"use_usg_acct_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"use_usg_auth_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"vlan_enabled",
						"false",
					),
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "site"),
					resource.TestCheckResourceAttrSet(
						"data.unifi_radius_profile.test",
						"interim_update_interval",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfileDataSource_withAuthServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_withAuthServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-ds-auth",
					),
					// Data source does not expose auth_server/acct_server blocks
				),
			},
		},
	})
}

func TestAccRadiusProfileDataSource_accountingEnabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileDataSourceConfig_accountingEnabled(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_profile.test",
						"interim_update_interval",
						"15m0s",
					),
				),
			},
		},
	})
}

func testAccRadiusProfileDataSourceConfig_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-ds"
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}

func testAccRadiusProfileDataSourceConfig_withAuthServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-ds-auth"

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-secret"
  }
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}

func testAccRadiusProfileDataSourceConfig_accountingEnabled() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-ds-acct"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = "15m0s"
}

data "unifi_radius_profile" "test" {
  name = unifi_radius_profile.test.name

  depends_on = [unifi_radius_profile.test]
}
`
}

func TestNewRadiusProfileDataSource(t *testing.T) {
	tests := []struct {
		name string
		want fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRadiusProfileDataSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRadiusProfileDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusProfileDataSource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.MetadataRequest
		resp *fwdatasource.MetadataResponse
	}
	tests := []struct {
		name string
		d    *radiusProfileDataSource
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

func Test_radiusProfileDataSource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.SchemaRequest
		resp *fwdatasource.SchemaResponse
	}
	tests := []struct {
		name string
		d    *radiusProfileDataSource
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

func Test_radiusProfileDataSource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ConfigureRequest
		resp *fwdatasource.ConfigureResponse
	}
	tests := []struct {
		name string
		d    *radiusProfileDataSource
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

func Test_radiusProfileDataSource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ReadRequest
		resp *fwdatasource.ReadResponse
	}
	tests := []struct {
		name string
		d    *radiusProfileDataSource
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
