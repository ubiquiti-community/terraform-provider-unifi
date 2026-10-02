package unifi

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClientInfoListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientInfoListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					// Verify the data source has clients attribute
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_info_list.test",
						"clients.#",
					),
					// Verify site is set
					resource.TestCheckResourceAttr(
						"data.unifi_client_info_list.test",
						"site",
						"default",
					),
				),
			},
		},
	})
}

func testAccClientInfoListDataSourceConfig_basic() string {
	return `
data "unifi_client_info_list" "test" {
}
`
}

func TestNewClientInfoListDataSource(t *testing.T) {
	d := NewClientInfoListDataSource()
	if d == nil {
		t.Fatal("NewClientInfoListDataSource() returned nil")
	}
	if _, ok := d.(fwdatasource.DataSourceWithConfigure); !ok {
		t.Error("expected DataSourceWithConfigure interface")
	}
}

func Test_clientInfoListDataSource_Metadata(t *testing.T) {
	tests := []struct {
		providerTypeName string
		wantTypeName     string
	}{
		{"unifi", "unifi_client_info_list"},
		{"test", "test_client_info_list"},
	}
	for _, tt := range tests {
		t.Run(tt.providerTypeName, func(t *testing.T) {
			d := &clientInfoListDataSource{}
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

func Test_clientInfoListDataSource_Schema(t *testing.T) {
	d := &clientInfoListDataSource{}
	resp := &fwdatasource.SchemaResponse{}
	d.Schema(context.Background(), fwdatasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() produced errors: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"site", "clients"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("missing attribute %q", attr)
		}
	}
}

func Test_clientInfoListDataSource_Configure(t *testing.T) {
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
			d := &clientInfoListDataSource{}
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
