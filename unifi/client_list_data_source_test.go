package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	gounifi "github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccClientListDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.test",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.test",
						"site",
						"default",
					),
				),
			},
		},
	})
}

func TestAccClientListDataSource_filtered(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClientListDataSourceConfig_wired(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.wired",
						"clients.#",
					),
					resource.TestCheckResourceAttr(
						"data.unifi_client_list.wired",
						"site",
						"default",
					),
				),
			},
			{
				Config: testAccClientListDataSourceConfig_blocked(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(
						"data.unifi_client_list.blocked",
						"clients.#",
					),
				),
			},
		},
	})
}

func testAccClientListDataSourceConfig_basic() string {
	return `
data "unifi_client_list" "test" {
}
`
}

func testAccClientListDataSourceConfig_wired() string {
	return `
data "unifi_client_list" "wired" {
  wired = true
}
`
}

func testAccClientListDataSourceConfig_blocked() string {
	return `
data "unifi_client_list" "blocked" {
  blocked = false
}
`
}

func TestNewClientListDataSource(t *testing.T) {
	tests := []struct {
		name string
		want fwdatasource.DataSource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewClientListDataSource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewClientListDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientListEntryAttrTypes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]attr.Type
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientListEntryAttrTypes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientListEntryAttrTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientListEntrySchemaAttributes(t *testing.T) {
	tests := []struct {
		name string
		want map[string]schema.Attribute
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientListEntrySchemaAttributes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientListEntrySchemaAttributes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientListDataSource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.MetadataRequest
		resp *fwdatasource.MetadataResponse
	}
	tests := []struct {
		name string
		d    *clientListDataSource
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

func Test_clientListDataSource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.SchemaRequest
		resp *fwdatasource.SchemaResponse
	}
	tests := []struct {
		name string
		d    *clientListDataSource
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

func Test_clientListDataSource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ConfigureRequest
		resp *fwdatasource.ConfigureResponse
	}
	tests := []struct {
		name string
		d    *clientListDataSource
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

func Test_clientListDataSource_resolveGroupID(t *testing.T) {
	type args struct {
		ctx       context.Context
		site      string
		groupName string
	}
	tests := []struct {
		name    string
		d       *clientListDataSource
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.d.resolveGroupID(tt.args.ctx, tt.args.site, tt.args.groupName)
			if (err != nil) != tt.wantErr {
				t.Errorf("clientListDataSource.resolveGroupID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("clientListDataSource.resolveGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clientListDataSource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwdatasource.ReadRequest
		resp *fwdatasource.ReadResponse
	}
	tests := []struct {
		name string
		d    *clientListDataSource
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

func Test_clientListEntryValues(t *testing.T) {
	type args struct {
		c    *gounifi.Client
		info *gounifi.ClientInfo
	}
	tests := []struct {
		name string
		args args
		want map[string]attr.Value
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientListEntryValues(tt.args.c, tt.args.info); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("clientListEntryValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_networkIDValue(t *testing.T) {
	type args struct {
		c *gounifi.Client
	}
	tests := []struct {
		name string
		args args
		want types.String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkIDValue(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("networkIDValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stringSliceToList(t *testing.T) {
	type args struct {
		s []string
	}
	tests := []struct {
		name string
		args args
		want basetypes.ListValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringSliceToList(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringSliceToList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_int64PointerValueOrNull(t *testing.T) {
	type args struct {
		v *int64
	}
	tests := []struct {
		name string
		args args
		want types.Int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := int64PointerValueOrNull(tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("int64PointerValueOrNull() = %v, want %v", got, tt.want)
			}
		})
	}
}
