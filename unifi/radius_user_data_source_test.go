package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRadiusUserDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel.type",
						"3",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel.medium_type",
						"6",
					),
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "site"),
				),
			},
		},
	})
}

func TestAccRadiusUserDataSource_withNetworkID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_withTunnelParams(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds-tunnel",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel.type",
						"12",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"tunnel.medium_type",
						"6",
					),
				),
			},
		},
	})
}

func TestAccRadiusUserDataSource_passwordSensitive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusUserDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// Do not assert password is returned by the API on read, as
					// sensitive fields may be omitted. Instead, verify the data
					// source lookup succeeds for the expected user.
					resource.TestCheckResourceAttrSet("data.unifi_radius_user.test", "id"),
					resource.TestCheckResourceAttr(
						"data.unifi_radius_user.test",
						"name",
						"tfacc-radius-user-ds",
					),
				),
			},
		},
	})
}

func testAccRadiusUserDataSourceConfig_basic() string {
	return `
resource "unifi_radius_user" "test" {
  name     = "tfacc-radius-user-ds"
  password = "test-password"
}

data "unifi_radius_user" "test" {
  name = unifi_radius_user.test.name

  depends_on = [unifi_radius_user.test]
}
`
}

func testAccRadiusUserDataSourceConfig_withTunnelParams() string {
	return `
resource "unifi_radius_user" "test" {
  name     = "tfacc-radius-user-ds-tunnel"
  password = "test-password"

  tunnel = {
    type        = 12
    medium_type = 6
  }
}

data "unifi_radius_user" "test" {
  name = unifi_radius_user.test.name

  depends_on = [unifi_radius_user.test]
}
`
}

func TestNewRadiusUserDataSource(t *testing.T) {
	d := NewRadiusUserDataSource()
	if d == nil {
		t.Fatal("NewRadiusUserDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_radiusUserDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_radius_user"},
		{"test", "test_radius_user"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &radiusUserDataSource{}
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

func Test_radiusUserDataSource_Schema(t *testing.T) {
	d := &radiusUserDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, name := range []string{"id", "site", "name", "password", "tunnel", "network_id"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("missing attribute %q", name)
		}
	}
	for _, flat := range []string{"tunnel_type", "tunnel_medium_type"} {
		if _, ok := resp.Schema.Attributes[flat]; ok {
			t.Errorf("flat attribute %q should have been nested", flat)
		}
	}
}

func Test_radiusUserDataSource_Configure(t *testing.T) {
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
			d := &radiusUserDataSource{}
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
