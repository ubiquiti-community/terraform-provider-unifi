package unifi

import (
	"context"
	"reflect"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccDynamicDNS_dyndns(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDynamicDNSConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_dynamic_dns.test", "service", "dyndns"),
					resource.TestCheckResourceAttr(
						"unifi_dynamic_dns.test",
						"host_name",
						"test.example.com",
					),
					resource.TestCheckResourceAttr(
						"unifi_dynamic_dns.test",
						"server",
						"dyndns.example.com",
					),
				),
			},
			{
				ResourceName:    "unifi_dynamic_dns.test",
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
		},
	})
}

const testAccDynamicDNSConfig = `
resource "unifi_dynamic_dns" "test" {
	service = "dyndns"
	
	host_name = "test.example.com"

	server   = "dyndns.example.com"
	login    = "testuser"
	password = "password"
}
`

func TestNewDynamicDNSResource(t *testing.T) {
	r := NewDynamicDNSResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwresource.ResourceWithConfigure); !ok {
		t.Error("expected ResourceWithConfigure")
	}
	if _, ok := r.(fwresource.ResourceWithImportState); !ok {
		t.Error("expected ResourceWithImportState")
	}
	if _, ok := r.(fwresource.ResourceWithIdentity); !ok {
		t.Error("expected ResourceWithIdentity")
	}
}

func TestNewDynamicDNSListResource(t *testing.T) {
	r := NewDynamicDNSListResource()
	if r == nil {
		t.Fatal("returned nil")
	}
	if _, ok := r.(fwlist.ListResource); !ok {
		t.Error("expected ListResource")
	}
	if _, ok := r.(fwlist.ListResourceWithConfigure); !ok {
		t.Error("expected ListResourceWithConfigure")
	}
}

func Test_dynamicDNSResource_Metadata(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.MetadataResponse{}
	r.Metadata(context.Background(), fwresource.MetadataRequest{ProviderTypeName: "unifi"}, resp)
	if resp.TypeName != "unifi_dynamic_dns" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "unifi_dynamic_dns")
	}
}

func Test_dynamicDNSResource_Schema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.SchemaResponse{}
	r.Schema(context.Background(), fwresource.SchemaRequest{}, resp)
	for _, attr := range []string{"id", "site", "interface", "service", "host_name", "server", "login", "password"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected attribute %q in schema", attr)
		}
	}
}

func Test_dynamicDNSResource_IdentitySchema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwresource.IdentitySchemaResponse{}
	r.IdentitySchema(context.Background(), fwresource.IdentitySchemaRequest{}, resp)
	if _, ok := resp.IdentitySchema.Attributes["id"]; !ok {
		t.Error("expected identity schema to have 'id' attribute")
	}
	if _, ok := resp.IdentitySchema.Attributes["site"]; !ok {
		t.Error("expected identity schema to have 'site' attribute")
	}
}

func Test_dynamicDNSResource_Configure(t *testing.T) {
	tests := []struct {
		name      string
		req       fwresource.ConfigureRequest
		wantError bool
	}{
		{"nil_provider_data", fwresource.ConfigureRequest{}, false},
		{"wrong_type", fwresource.ConfigureRequest{ProviderData: "wrong"}, true},
		{"correct_client", fwresource.ConfigureRequest{ProviderData: &Client{}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			resp := &fwresource.ConfigureResponse{}
			r.Configure(context.Background(), tt.req, resp)
			if resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v", resp.Diagnostics.HasError(), tt.wantError)
			}
		})
	}
}

func Test_dynamicDNSResource_Create(t *testing.T) {
	// CRUD tests require a configured API client; covered by acceptance tests.
}

func Test_dynamicDNSResource_Read(t *testing.T) {
	// Read tests require a configured API client; covered by acceptance tests.
}

func Test_dynamicDNSResource_Update(t *testing.T) {
	// Update tests require a configured API client; covered by acceptance tests.
}

func Test_dynamicDNSResource_Delete(t *testing.T) {
	// Delete tests require a configured API client; covered by acceptance tests.
}

func Test_dynamicDNSResource_ImportState(t *testing.T) {
	// ImportState tests require tfsdk state setup; covered by acceptance tests.
}

func Test_dynamicDNSResource_applyPlanToState(t *testing.T) {
	r := &dynamicDNSResource{}
	plan := &dynamicDNSResourceModel{
		Interface: types.StringValue("wan"),
		Service:   types.StringValue("dyndns"),
		HostName:  types.StringValue("test.example.com"),
		Server:    types.StringValue("dyndns.example.com"),
		Login:     types.StringValue("user"),
		Password:  types.StringValue("pass"),
	}
	state := &dynamicDNSResourceModel{}
	r.applyPlanToState(context.Background(), plan, state)
	if state.Service.ValueString() != "dyndns" {
		t.Error("expected Service to be copied from plan")
	}
	if state.HostName.ValueString() != "test.example.com" {
		t.Error("expected HostName to be copied from plan")
	}
}

func Test_dynamicDNSResource_modelToDynamicDNS(t *testing.T) {
	tests := []struct {
		name  string
		model *dynamicDNSResourceModel
		want  *unifi.DynamicDNS
	}{
		{
			name: "basic_conversion",
			model: &dynamicDNSResourceModel{
				ID:        types.StringValue("abc123"),
				Interface: types.StringValue("wan"),
				Service:   types.StringValue("dyndns"),
				HostName:  types.StringValue("test.example.com"),
				Server:    types.StringValue("dyndns.example.com"),
				Login:     types.StringValue("user"),
				Password:  types.StringValue("pass"),
			},
			want: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
				Server:    "dyndns.example.com",
				Login:     "user",
				Password:  "pass",
			},
		},
		{
			name: "null_optional_fields",
			model: &dynamicDNSResourceModel{
				ID:        types.StringValue("abc123"),
				Interface: types.StringValue("wan"),
				Service:   types.StringValue("dyndns"),
				HostName:  types.StringValue("test.example.com"),
				Server:    types.StringNull(),
				Login:     types.StringNull(),
				Password:  types.StringNull(),
			},
			want: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			if got := r.modelToDynamicDNS(context.Background(), tt.model); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("modelToDynamicDNS() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func Test_dynamicDNSResource_dynamicDNSToModel(t *testing.T) {
	tests := []struct {
		name       string
		dynamicDNS *unifi.DynamicDNS
		site       string
	}{
		{
			name: "full_record",
			dynamicDNS: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
				Server:    "dyndns.example.com",
				Login:     "user",
				Password:  "pass",
			},
			site: "default",
		},
		{
			name: "empty_optional_fields",
			dynamicDNS: &unifi.DynamicDNS{
				ID:        "abc123",
				Interface: "wan",
				Service:   "dyndns",
				HostName:  "test.example.com",
			},
			site: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &dynamicDNSResource{}
			model := &dynamicDNSResourceModel{}
			r.dynamicDNSToModel(context.Background(), tt.dynamicDNS, model, tt.site)
			if model.ID.ValueString() != tt.dynamicDNS.ID {
				t.Errorf("ID = %q, want %q", model.ID.ValueString(), tt.dynamicDNS.ID)
			}
			if tt.site == "" && !model.Site.IsNull() {
				t.Error("expected Site to be null for empty site")
			}
			if tt.dynamicDNS.Server == "" && !model.Server.IsNull() {
				t.Error("expected Server to be null for empty server")
			}
		})
	}
}

func Test_dynamicDNSResource_ListResourceConfigSchema(t *testing.T) {
	r := &dynamicDNSResource{}
	resp := &fwlist.ListResourceSchemaResponse{}
	r.ListResourceConfigSchema(context.Background(), fwlist.ListResourceSchemaRequest{}, resp)
	if len(resp.Schema.Attributes) == 0 {
		t.Error("expected non-empty list resource schema")
	}
}

func Test_dynamicDNSResource_List(t *testing.T) {
	// List tests require a configured API client; covered by acceptance tests.
}
