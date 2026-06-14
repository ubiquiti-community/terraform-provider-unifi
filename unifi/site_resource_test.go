package unifi

import (
	"context"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccSiteFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("unifi_site.test", "description", "tfacc-test"),
					resource.TestCheckResourceAttrSet("unifi_site.test", "name"),
				),
				ResourceName:  "unifi_site.test",
				ImportState:   true,
				ImportStateId: "default",
			},
		},
	})
}

func testAccSiteFrameworkConfig_basic() string {
	return `
resource "unifi_site" "test" {
	name        = "default"
	description = "tfacc-test"
}
`
}

// TestSiteToModelNilDoesNotPanic guards #261: siteToModel must return an error
// for a nil site instead of dereferencing it (the read path used to fall
// through to a nil siteToModel on a not-found, panicking the provider).
func TestSiteToModelNilDoesNotPanic(t *testing.T) {
	r := &siteFrameworkResource{}
	var model siteFrameworkResourceModel
	diags := r.siteToModel(context.Background(), nil, &model)
	if !diags.HasError() {
		t.Fatal("expected an error diagnostic for a nil site, got none")
	}
}

func TestNewSiteFrameworkResource(t *testing.T) {
	r := NewSiteFrameworkResource()
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

func TestNewSiteListResource(t *testing.T) {
	r := NewSiteListResource()
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

func Test_siteFrameworkResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name         string
		r            *siteFrameworkResource
		args         args
		wantTypeName string
	}{
		{
			name: "type_name",
			r:    &siteFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
			wantTypeName: "unifi_site",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != tt.wantTypeName {
				t.Errorf("TypeName = %q, want %q", tt.args.resp.TypeName, tt.wantTypeName)
			}
		})
	}
}

func Test_siteFrameworkResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *siteFrameworkResource
		args args
	}{
		{
			name: "has_id",
			r:    &siteFrameworkResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwresource.IdentitySchemaRequest{},
				resp: &fwresource.IdentitySchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if _, ok := tt.args.resp.IdentitySchema.Attributes["id"]; !ok {
				t.Error("expected identity schema to have 'id' attribute")
			}
		})
	}
}

func Test_siteFrameworkResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *siteFrameworkResource
		args args
	}{
		{
			name: "has_key_attributes",
			r:    &siteFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			for _, attr := range []string{"id", "name", "description"} {
				if _, ok := tt.args.resp.Schema.Attributes[attr]; !ok {
					t.Errorf("expected attribute %q in schema", attr)
				}
			}
		})
	}
}

func Test_siteFrameworkResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name      string
		r         *siteFrameworkResource
		args      args
		wantError bool
	}{
		{
			name: "nil_provider_data",
			r:    &siteFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
		{
			name: "wrong_type",
			r:    &siteFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: "wrong"},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: true,
		},
		{
			name: "correct_client",
			r:    &siteFrameworkResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: &Client{}},
				resp: &fwresource.ConfigureResponse{},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.Diagnostics.HasError() != tt.wantError {
				t.Errorf(
					"hasError = %v, want %v",
					tt.args.resp.Diagnostics.HasError(),
					tt.wantError,
				)
			}
		})
	}
}

func Test_siteFrameworkResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *siteFrameworkResourceModel
		state *siteFrameworkResourceModel
	}
	tests := []struct {
		name string
		r    *siteFrameworkResource
		args args
	}{
		{
			name: "copies_description",
			r:    &siteFrameworkResource{},
			args: args{
				in0: context.Background(),
				plan: &siteFrameworkResourceModel{
					Description: types.StringValue("new-desc"),
				},
				state: &siteFrameworkResourceModel{
					Description: types.StringValue("old-desc"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
			if tt.args.state.Description.ValueString() != "new-desc" {
				t.Error("expected Description to be copied from plan")
			}
		})
	}
}

func Test_siteFrameworkResource_siteToModel(t *testing.T) {
	type args struct {
		in0   context.Context
		site  *unifi.Site
		model *siteFrameworkResourceModel
	}
	tests := []struct {
		name      string
		r         *siteFrameworkResource
		args      args
		wantError bool
	}{
		{
			name: "nil_site_returns_error",
			r:    &siteFrameworkResource{},
			args: args{
				in0:   context.Background(),
				site:  nil,
				model: &siteFrameworkResourceModel{},
			},
			wantError: true,
		},
		{
			name: "empty_id_and_name_returns_error",
			r:    &siteFrameworkResource{},
			args: args{
				in0:   context.Background(),
				site:  &unifi.Site{},
				model: &siteFrameworkResourceModel{},
			},
			wantError: true,
		},
		{
			name: "valid_site",
			r:    &siteFrameworkResource{},
			args: args{
				in0: context.Background(),
				site: &unifi.Site{
					ID:          "abc123",
					Name:        "default",
					Description: "Default site",
				},
				model: &siteFrameworkResourceModel{},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.siteToModel(tt.args.in0, tt.args.site, tt.args.model)
			if got.HasError() != tt.wantError {
				t.Errorf("hasError = %v, want %v, diags: %v", got.HasError(), tt.wantError, got)
			}
			if !tt.wantError && tt.args.site != nil {
				if tt.args.model.ID.ValueString() != tt.args.site.ID {
					t.Errorf("ID = %q, want %q", tt.args.model.ID.ValueString(), tt.args.site.ID)
				}
			}
		})
	}
}

func Test_siteFrameworkResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *siteFrameworkResource
		args args
	}{
		{
			name: "returns_schema",
			r:    &siteFrameworkResource{},
			args: args{
				in0:  context.Background(),
				in1:  fwlist.ListResourceSchemaRequest{},
				resp: &fwlist.ListResourceSchemaResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
			if len(tt.args.resp.Schema.Attributes) == 0 && len(tt.args.resp.Schema.Blocks) == 0 {
				t.Error("expected non-empty list resource schema")
			}
		})
	}
}
