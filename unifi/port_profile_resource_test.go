package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccPortProfileFramework_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortProfileFrameworkConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.test",
						"name",
						"Test Port Profile",
					),
					resource.TestCheckResourceAttr("unifi_port_profile.test", "autoneg", "true"),
				),
			},
			{
				ResourceName:      "unifi_port_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPortProfileFrameworkConfig_basic() string {
	return `
resource "unifi_port_profile" "test" {
	name     = "Test Port Profile"
	autoneg  = true
	op_mode  = "switch"
}
`
}

func TestAccPortProfileFramework_vlanFields(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPortProfileFrameworkConfig_vlanFields(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_port_profile.vlan", "id"),
					// native_networkconf_id is now persisted/computed (was previously a no-op).
					resource.TestCheckResourceAttrSet(
						"unifi_port_profile.vlan",
						"native_networkconf_id",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.vlan",
						"setting_preference",
						"manual",
					),
					resource.TestCheckResourceAttr(
						"unifi_port_profile.vlan",
						"port_keepalive_enabled",
						"true",
					),
				),
			},
			{
				ResourceName:      "unifi_port_profile.vlan",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPortProfileFrameworkConfig_vlanFields() string {
	return `
resource "unifi_port_profile" "vlan" {
	name                   = "Test Port Profile VLAN"
	op_mode                = "switch"
	setting_preference     = "manual"
	port_keepalive_enabled = true
}
`
}

func TestNewPortProfileFrameworkResource(t *testing.T) {
	got := NewPortProfileFrameworkResource()
	if got == nil {
		t.Fatal("NewPortProfileFrameworkResource() returned nil")
	}
	if _, ok := got.(fwresource.ResourceWithImportState); !ok {
		t.Errorf(
			"NewPortProfileFrameworkResource() does not implement fwresource.ResourceWithImportState",
		)
	}
	if _, ok := got.(fwresource.ResourceWithIdentity); !ok {
		t.Errorf(
			"NewPortProfileFrameworkResource() does not implement fwresource.ResourceWithIdentity",
		)
	}
	if _, ok := got.(fwresource.ResourceWithUpgradeState); !ok {
		t.Errorf(
			"NewPortProfileFrameworkResource() does not implement fwresource.ResourceWithUpgradeState",
		)
	}
}

func TestNewPortProfileListResource(t *testing.T) {
	got := NewPortProfileListResource()
	if got == nil {
		t.Fatal("NewPortProfileListResource() returned nil")
	}
	if _, ok := got.(fwlist.ListResourceWithConfigure); !ok {
		t.Errorf("NewPortProfileListResource() does not implement fwlist.ListResourceWithConfigure")
	}
}

func Test_portProfileResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name         string
		r            *portProfileResource
		args         args
		wantTypeName string
	}{
		{
			name: "type name with provider prefix",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: "unifi"},
				resp: &fwresource.MetadataResponse{},
			},
			wantTypeName: "unifi_port_profile",
		},
		{
			name: "type name with empty provider prefix",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.MetadataRequest{ProviderTypeName: ""},
				resp: &fwresource.MetadataResponse{},
			},
			wantTypeName: "_port_profile",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.args.resp.TypeName != tt.wantTypeName {
				t.Errorf(
					"Metadata() TypeName = %q, want %q",
					tt.args.resp.TypeName,
					tt.wantTypeName,
				)
			}
		})
	}
}

func Test_portProfileResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *portProfileResource
		args args
	}{
		{
			name: "does not panic",
			r:    &portProfileResource{},
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
		})
	}
}

func Test_portProfileResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name           string
		r              *portProfileResource
		args           args
		wantAttributes []string
	}{
		{
			name: "schema contains key attributes",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.SchemaRequest{},
				resp: &fwresource.SchemaResponse{},
			},
			wantAttributes: []string{"id", "name", "op_mode", "autoneg", "site"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
			for _, attr := range tt.wantAttributes {
				if _, ok := tt.args.resp.Schema.Attributes[attr]; !ok {
					t.Errorf("Schema() missing attribute %q", attr)
				}
			}
		})
	}
}

func Test_portProfileResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *portProfileResource
		args args
	}{
		{
			name: "returns non-nil map",
			r:    &portProfileResource{},
			args: args{ctx: context.Background()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.UpgradeState(tt.args.ctx)
			if got == nil {
				t.Error("portProfileResource.UpgradeState() returned nil")
			}
		})
	}
}

func Test_portProfileResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name       string
		r          *portProfileResource
		args       args
		wantErr    bool
		wantClient bool
	}{
		{
			name: "nil provider data produces no error",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: nil},
				resp: &fwresource.ConfigureResponse{},
			},
			wantErr:    false,
			wantClient: false,
		},
		{
			name: "wrong type produces error",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: "wrong-type"},
				resp: &fwresource.ConfigureResponse{},
			},
			wantErr:    true,
			wantClient: false,
		},
		{
			name: "correct Client type sets client",
			r:    &portProfileResource{},
			args: args{
				ctx:  context.Background(),
				req:  fwresource.ConfigureRequest{ProviderData: &Client{}},
				resp: &fwresource.ConfigureResponse{},
			},
			wantErr:    false,
			wantClient: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
			if tt.wantErr && !tt.args.resp.Diagnostics.HasError() {
				t.Error("Configure() expected error diagnostic, got none")
			}
			if !tt.wantErr && tt.args.resp.Diagnostics.HasError() {
				t.Errorf("Configure() unexpected error: %v", tt.args.resp.Diagnostics.Errors())
			}
			if tt.wantClient && tt.r.client == nil {
				t.Error("Configure() expected client to be set, got nil")
			}
		})
	}
}

func Test_portProfileResource_modelToAPIPortProfile(t *testing.T) {
	type args struct {
		ctx   context.Context
		model *portProfileResourceModel
	}
	tests := []struct {
		name  string
		r     *portProfileResource
		args  args
		want  *unifi.PortProfile
		want1 diag.Diagnostics
	}{
		{
			name: "minimal model conversion",
			r:    &portProfileResource{},
			args: args{
				ctx: context.Background(),
				model: &portProfileResourceModel{
					ID:                         types.StringNull(),
					Site:                       types.StringNull(),
					Name:                       types.StringValue("test"),
					OpMode:                     types.StringValue("switch"),
					Autoneg:                    types.BoolValue(true),
					Dot1XCtrl:                  types.StringNull(),
					Dot1XIdleTimeout:           timetypes.NewGoDurationNull(),
					EgressRateLimitKbps:        types.Int64Null(),
					EgressRateLimitKbpsEnabled: types.BoolNull(),
					Forward:                    types.StringNull(),
					FullDuplex:                 types.BoolNull(),
					Isolation:                  types.BoolNull(),
					LLDPMedEnabled:             types.BoolNull(),
					LLDPMedNotifyEnabled:       types.BoolNull(),
					NativeNetworkConfID:        types.StringNull(),
					PoeMode:                    types.StringNull(),
					PortSecurityEnabled:        types.BoolNull(),
					PortSecurityMacAddress:     types.SetNull(types.StringType),
					PriorityQueue1Level:        types.Int64Null(),
					PriorityQueue2Level:        types.Int64Null(),
					PriorityQueue3Level:        types.Int64Null(),
					PriorityQueue4Level:        types.Int64Null(),
					Speed:                      types.Int64Null(),
					StormctrlBcastEnabled:      types.BoolNull(),
					StormctrlBcastLevel:        types.Int64Null(),
					StormctrlBcastRate:         types.Int64Null(),
					StormctrlMcastEnabled:      types.BoolNull(),
					StormctrlMcastLevel:        types.Int64Null(),
					StormctrlMcastRate:         types.Int64Null(),
					StormctrlType:              types.StringNull(),
					StormctrlUcastEnabled:      types.BoolNull(),
					StormctrlUcastLevel:        types.Int64Null(),
					StormctrlUcastRate:         types.Int64Null(),
					STPPortMode:                types.BoolNull(),
					TaggedNetworkConfIDs:       types.SetNull(types.StringType),
					VoiceNetworkConfID:         types.StringNull(),
					ExcludedNetworkConfIDs:     types.SetNull(types.StringType),
					MulticastRouterNetworkIDs:  types.SetNull(types.StringType),
					TaggedVLANMgmt:             types.StringNull(),
					FecMode:                    types.StringNull(),
					SettingPreference:          types.StringNull(),
					PortKeepaliveEnabled:       types.BoolNull(),
				},
			},
			want: &unifi.PortProfile{
				Name:    "test",
				OpMode:  "switch",
				Autoneg: true,
			},
			want1: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.r.modelToAPIPortProfile(tt.args.ctx, tt.args.model)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf(
					"portProfileResource.modelToAPIPortProfile() got = %+v, want %+v",
					got,
					tt.want,
				)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf(
					"portProfileResource.modelToAPIPortProfile() got1 = %v, want %v",
					got1,
					tt.want1,
				)
			}
		})
	}
}

func Test_portProfileResource_portProfileToModel(t *testing.T) {
	type args struct {
		ctx   context.Context
		api   *unifi.PortProfile
		model *portProfileResourceModel
		site  string
	}
	tests := []struct {
		name      string
		r         *portProfileResource
		args      args
		want      diag.Diagnostics
		checkFunc func(t *testing.T, model *portProfileResourceModel)
	}{
		{
			name: "minimal API to model conversion",
			r:    &portProfileResource{},
			args: args{
				ctx: context.Background(),
				api: &unifi.PortProfile{
					ID:     "abc123",
					Name:   "test-profile",
					OpMode: "switch",
				},
				model: &portProfileResourceModel{},
				site:  "default",
			},
			want: nil,
			checkFunc: func(t *testing.T, model *portProfileResourceModel) {
				if model.ID.ValueString() != "abc123" {
					t.Errorf("ID = %q, want %q", model.ID.ValueString(), "abc123")
				}
				if model.Name.ValueString() != "test-profile" {
					t.Errorf("Name = %q, want %q", model.Name.ValueString(), "test-profile")
				}
				if model.OpMode.ValueString() != "switch" {
					t.Errorf("OpMode = %q, want %q", model.OpMode.ValueString(), "switch")
				}
				if model.Site.ValueString() != "default" {
					t.Errorf("Site = %q, want %q", model.Site.ValueString(), "default")
				}
				if !model.PortSecurityMacAddress.IsNull() {
					t.Error("PortSecurityMacAddress should be null for empty API field")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.portProfileToModel(tt.args.ctx, tt.args.api, tt.args.model, tt.args.site)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("portProfileResource.portProfileToModel() = %v, want %v", got, tt.want)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.args.model)
			}
		})
	}
}

func Test_portProfileResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *portProfileResource
		args args
	}{
		{
			name: "does not panic",
			r:    &portProfileResource{},
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
		})
	}
}
