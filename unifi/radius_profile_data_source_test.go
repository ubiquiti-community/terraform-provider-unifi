package unifi

import (
	"context"
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
	d := NewRadiusProfileDataSource()
	if d == nil {
		t.Fatal("NewRadiusProfileDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_radiusProfileDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_radius_profile"},
		{"test", "test_radius_profile"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &radiusProfileDataSource{}
			resp := &fwdatasource.MetadataResponse{}
			d.Metadata(
				context.Background(),
				fwdatasource.MetadataRequest{ProviderTypeName: tt.providerTypeName},
				resp,
			)
			if resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_radiusProfileDataSource_Schema(t *testing.T) {
	d := &radiusProfileDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"id", "site", "name", "accounting_enabled", "interim_update_enabled"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_radiusProfileDataSource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		data      any
		wantError bool
	}{
		{"nil provider data", nil, false},
		{"wrong type", "wrong", true},
		{"correct client type", &Client{Site: "default"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &radiusProfileDataSource{}
			resp := &fwdatasource.ConfigureResponse{}
			d.Configure(
				context.Background(),
				fwdatasource.ConfigureRequest{ProviderData: tt.data},
				resp,
			)
			if tt.wantError && !resp.Diagnostics.HasError() {
				t.Error("expected error in diagnostics")
			}
			if !tt.wantError && resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", resp.Diagnostics)
			}
		})
	}
}
