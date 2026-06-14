package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccFirewallGroupFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallGroupFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"name",
						"Test Address Group",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"type",
						"address-group",
					),
					resource.TestCheckResourceAttr("unifi_firewall_group.test", "members.#", "2"),
				),
			},
			{
				ResourceName:      "unifi_firewall_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFirewallGroupFramework_portGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallGroupFrameworkConfig_portGroup(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_firewall_group.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"name",
						"Test Port Group",
					),
					resource.TestCheckResourceAttr(
						"unifi_firewall_group.test",
						"type",
						"port-group",
					),
					resource.TestCheckResourceAttr("unifi_firewall_group.test", "members.#", "3"),
				),
			},
		},
	})
}

func testAccFirewallGroupFrameworkConfig_basic() string {
	return `
resource "unifi_firewall_group" "test" {
	name = "Test Address Group"
	type = "address-group"
	members = [
		"192.168.1.10",
		"192.168.1.20"
	]
}
`
}

func testAccFirewallGroupFrameworkConfig_portGroup() string {
	return `
resource "unifi_firewall_group" "test" {
	name = "Test Port Group"
	type = "port-group"
	members = [
		"80",
		"443",
		"8080"
	]
}
`
}

func TestNewFirewallGroupFrameworkResource(t *testing.T) {
	got := NewFirewallGroupFrameworkResource()
	if got == nil {
		t.Fatal("NewFirewallGroupFrameworkResource() returned nil")
	}
	_ = got
	_ = got.(fwresource.ResourceWithImportState)
	_ = got.(fwresource.ResourceWithIdentity)
}

func TestNewFirewallGroupListResource(t *testing.T) {
	got := NewFirewallGroupListResource()
	if got == nil {
		t.Fatal("NewFirewallGroupListResource() returned nil")
	}
	_ = got
}

func Test_firewallGroupResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
	}{
		{
			name: "returns correct type name",
			r:    &firewallGroupResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != "unifi_firewall_group" {
				t.Errorf("TypeName = %q, want %q", tt.args.resp.TypeName, "unifi_firewall_group")
			}
		})
	}
}

func Test_firewallGroupResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
	}{
		{
			name: "has id attribute",
			r:    &firewallGroupResource{},
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
				t.Error("IdentitySchema missing 'id' attribute")
			}
		})
	}
}

func Test_firewallGroupResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
	}{
		{
			name: "has expected attributes",
			r:    &firewallGroupResource{},
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
			s := tt.args.resp.Schema
			for _, key := range []string{"id", "site", "name", "type", "members"} {
				if _, ok := s.Attributes[key]; !ok {
					t.Errorf("Schema missing attribute %q", key)
				}
			}
		})
	}
}

func Test_firewallGroupResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name       string
		r          *firewallGroupResource
		args       args
		wantErr    bool
		wantClient bool
	}{
		{
			name: "nil provider data",
			r:    &firewallGroupResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{},
				resp: &fwresource.ConfigureResponse{},
			},
		},
		{
			name: "wrong type",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: "wrong",
				},
				resp: &fwresource.ConfigureResponse{},
			},
			wantErr: true,
		},
		{
			name: "correct client",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				req: fwresource.ConfigureRequest{
					ProviderData: &Client{},
				},
				resp: &fwresource.ConfigureResponse{},
			},
			wantClient: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.wantErr && !tt.args.resp.Diagnostics.HasError() {
				t.Error("expected error diagnostic")
			}
			if !tt.wantErr && tt.args.resp.Diagnostics.HasError() {
				t.Errorf("unexpected error: %v", tt.args.resp.Diagnostics)
			}
			if tt.wantClient && tt.r.client == nil {
				t.Error("expected client to be set")
			}
		})
	}
}

func Test_firewallGroupResource_modelToAPIFirewallGroup(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *firewallGroupResourceModel
	}
	ctx := context.Background()
	membersSet, _ := types.SetValueFrom(ctx, types.StringType, []string{"10.0.0.1", "10.0.0.2"})
	portMembersSet, _ := types.SetValueFrom(ctx, types.StringType, []string{"80", "443"})

	tests := []struct {
		name    string
		r       *firewallGroupResource
		args    args
		want    *unifi.FirewallGroup
		wantErr bool
	}{
		{
			name: "basic address group",
			r:    &firewallGroupResource{},
			args: args{
				ctx: ctx,
				model: &firewallGroupResourceModel{
					Name:    types.StringValue("Test Group"),
					Type:    types.StringValue("address-group"),
					Members: membersSet,
				},
			},
			want: &unifi.FirewallGroup{
				Name:         "Test Group",
				GroupType:    "address-group",
				GroupMembers: []string{"10.0.0.1", "10.0.0.2"},
			},
		},
		{
			name: "empty members",
			r:    &firewallGroupResource{},
			args: args{
				ctx: ctx,
				model: &firewallGroupResourceModel{
					Name:    types.StringValue("Empty Group"),
					Type:    types.StringValue("address-group"),
					Members: types.SetNull(types.StringType),
				},
			},
			want: &unifi.FirewallGroup{
				Name:         "Empty Group",
				GroupType:    "address-group",
				GroupMembers: nil,
			},
		},
		{
			name: "port group",
			r:    &firewallGroupResource{},
			args: args{
				ctx: ctx,
				model: &firewallGroupResourceModel{
					Name:    types.StringValue("Port Group"),
					Type:    types.StringValue("port-group"),
					Members: portMembersSet,
				},
			},
			want: &unifi.FirewallGroup{
				Name:         "Port Group",
				GroupType:    "port-group",
				GroupMembers: []string{"80", "443"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.modelToAPIFirewallGroup(tt.args.ctx, tt.args.model)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"firewallGroupResource.modelToAPIFirewallGroup() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"firewallGroupResource.modelToAPIFirewallGroup() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func Test_firewallGroupResource_setResourceData(t *testing.T) {
	type args struct {
		ctx           context.Context
		firewallGroup *unifi.FirewallGroup
		model         *firewallGroupResourceModel
		site          string
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
	}{
		{
			name: "populates model from API",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				firewallGroup: &unifi.FirewallGroup{
					ID:           "fg1",
					Name:         "Test",
					GroupType:    "address-group",
					GroupMembers: []string{"10.0.0.1"},
				},
				model: &firewallGroupResourceModel{},
				site:  "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.setResourceData(tt.args.ctx, tt.args.firewallGroup, tt.args.model, tt.args.site)
			if tt.args.model.ID.ValueString() != "fg1" {
				t.Errorf("ID = %q, want %q", tt.args.model.ID.ValueString(), "fg1")
			}
			if tt.args.model.Name.ValueString() != "Test" {
				t.Errorf("Name = %q, want %q", tt.args.model.Name.ValueString(), "Test")
			}
		})
	}
}

func Test_firewallGroupResource_firewallGroupToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		api   *unifi.FirewallGroup
		model *firewallGroupResourceModel
		site  string
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
		want diag.Diagnostics
	}{
		{
			name: "basic API to model",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				api: &unifi.FirewallGroup{
					ID:           "fg1",
					Name:         "Test",
					GroupType:    "address-group",
					GroupMembers: []string{"10.0.0.1"},
				},
				model: &firewallGroupResourceModel{},
				site:  "default",
			},
			want: nil,
		},
		{
			name: "empty members produces null set",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				api: &unifi.FirewallGroup{
					ID:           "fg2",
					Name:         "Empty",
					GroupType:    "address-group",
					GroupMembers: []string{},
				},
				model: &firewallGroupResourceModel{},
				site:  "default",
			},
			want: nil,
		},
		{
			name: "empty name produces null",
			r:    &firewallGroupResource{},
			args: args{
				ctx: context.Background(),
				api: &unifi.FirewallGroup{
					ID:           "fg3",
					Name:         "",
					GroupType:    "address-group",
					GroupMembers: []string{},
				},
				model: &firewallGroupResourceModel{},
				site:  "default",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.firewallGroupToModel(tt.args.ctx, tt.args.api, tt.args.model, tt.args.site)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("firewallGroupResource.firewallGroupToModel() = %v, want %v", got, tt.want)
			}
			if tt.args.api.ID != "" && tt.args.model.ID.ValueString() != tt.args.api.ID {
				t.Errorf("ID = %q, want %q", tt.args.model.ID.ValueString(), tt.args.api.ID)
			}
			if tt.args.api.Name == "" && !tt.args.model.Name.IsNull() {
				t.Error("expected Name to be null for empty API name")
			}
			if len(tt.args.api.GroupMembers) == 0 && !tt.args.model.Members.IsNull() {
				t.Error("expected Members to be null for empty GroupMembers")
			}
		})
	}
}

func Test_firewallGroupResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *firewallGroupResource
		args args
	}{
		{
			name: "has site attribute",
			r:    &firewallGroupResource{},
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
			if _, ok := tt.args.resp.Schema.Attributes["site"]; !ok {
				t.Error("ListResourceConfigSchema missing 'site' attribute")
			}
		})
	}
}
