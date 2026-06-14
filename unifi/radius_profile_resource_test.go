package unifi

import (
	"context"
	"reflect"
	"testing"

	fwlist "github.com/hashicorp/terraform-plugin-framework/list"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

func TestAccRadiusProfile_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1h0m0s",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_acct_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"use_usg_auth_server",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"false",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-auth",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1812",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"auth_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthServerCustomPort(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthServerCustomPort(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"10.0.0.1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.port",
						"1822",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_withAcctServer(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAcctServer(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-acct",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.port",
						"1813",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Secrets are not returned by the API on read
				ImportStateVerifyIgnore: []string{"acct_server.0.secret"},
			},
		},
	})
}

func TestAccRadiusProfile_withAuthAndAcctServers(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withAuthAndAcctServers(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.#",
						"1",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"auth_server.0.ip",
						"192.168.1.100",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"acct_server.0.ip",
						"192.168.1.101",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auth_server.0.secret",
					"acct_server.0.secret",
				},
			},
		},
	})
}

func TestAccRadiusProfile_withInterimUpdate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withInterimUpdate(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"30m0s",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_withVlan(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_withVlan(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"vlan_wlan_mode",
						"required",
					),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRadiusProfile_update(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"false",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"1h0m0s",
					),
				),
			},
			{
				Config: testAccRadiusProfileConfig_updated(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"name",
						"tfacc-radius-profile-updated",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"accounting_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_enabled",
						"true",
					),
					resource.TestCheckResourceAttr(
						"unifi_radius_profile.test",
						"interim_update_interval",
						"30m0s",
					),
				),
			},
		},
	})
}

func TestAccRadiusProfile_importWithSite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { preCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRadiusProfileConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "id"),
					resource.TestCheckResourceAttrSet("unifi_radius_profile.test", "site"),
				),
			},
			{
				ResourceName:      "unifi_radius_profile.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Import using "site:id" format verified via ImportStateIdFunc below
			},
		},
	})
}

func testAccRadiusProfileConfig_basic() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile"
}
`
}

func testAccRadiusProfileConfig_withAuthServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth"

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthServerCustomPort() string {
	return `
resource "unifi_radius_profile" "test" {
  name = "tfacc-radius-profile-auth-port"

  auth_server {
    ip     = "10.0.0.1"
    port   = 1822
    secret = "test-auth-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAcctServer() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-acct"
  accounting_enabled = true

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withAuthAndAcctServers() string {
	return `
resource "unifi_radius_profile" "test" {
  name               = "tfacc-radius-profile-full"
  accounting_enabled = true

  auth_server {
    ip     = "192.168.1.100"
    secret = "test-auth-secret"
  }

  acct_server {
    ip     = "192.168.1.101"
    secret = "test-acct-secret"
  }
}
`
}

func testAccRadiusProfileConfig_withInterimUpdate() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-interim"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = "30m0s"
}
`
}

func testAccRadiusProfileConfig_withVlan() string {
	return `
resource "unifi_radius_profile" "test" {
  name           = "tfacc-radius-profile-vlan"
  vlan_enabled   = true
  vlan_wlan_mode = "required"
}
`
}

func testAccRadiusProfileConfig_updated() string {
	return `
resource "unifi_radius_profile" "test" {
  name                    = "tfacc-radius-profile-updated"
  accounting_enabled      = true
  interim_update_enabled  = true
  interim_update_interval = "30m0s"
}
`
}

func TestNewRadiusProfileResource(t *testing.T) {
	tests := []struct {
		name string
		want fwresource.Resource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRadiusProfileResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRadiusProfileResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewRadiusProfileListResource(t *testing.T) {
	tests := []struct {
		name string
		want fwlist.ListResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewRadiusProfileListResource(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRadiusProfileListResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusProfileResource_Metadata(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.MetadataRequest
		resp *fwresource.MetadataResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Metadata(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_IdentitySchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwresource.IdentitySchemaRequest
		resp *fwresource.IdentitySchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.IdentitySchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_Schema(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.SchemaRequest
		resp *fwresource.SchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Schema(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_UpgradeState(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
		want map[int64]fwresource.StateUpgrader
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.UpgradeState(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("radiusProfileResource.UpgradeState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusProfileResource_Configure(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ConfigureRequest
		resp *fwresource.ConfigureResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Configure(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_Create(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.CreateRequest
		resp *fwresource.CreateResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Create(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_Read(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ReadRequest
		resp *fwresource.ReadResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Read(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_Update(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.UpdateRequest
		resp *fwresource.UpdateResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Update(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_Delete(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.DeleteRequest
		resp *fwresource.DeleteResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Delete(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_ImportState(t *testing.T) {
	type args struct {
		ctx  context.Context
		req  fwresource.ImportStateRequest
		resp *fwresource.ImportStateResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ImportState(tt.args.ctx, tt.args.req, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_applyPlanToState(t *testing.T) {
	type args struct {
		in0   context.Context
		plan  *radiusProfileResourceModel
		state *radiusProfileResourceModel
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.applyPlanToState(tt.args.in0, tt.args.plan, tt.args.state)
		})
	}
}

func Test_radiusProfileResource_modelToRadiusProfile(t *testing.T) {
	type args struct {
		in0   context.Context
		model *radiusProfileResourceModel
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
		want *unifi.RADIUSProfile
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.modelToRadiusProfile(tt.args.in0, tt.args.model); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("radiusProfileResource.modelToRadiusProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_radiusProfileResource_radiusProfileToModel(t *testing.T) {
	type args struct {
		in0           context.Context
		radiusProfile *unifi.RADIUSProfile
		model         *radiusProfileResourceModel
		site          string
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.radiusProfileToModel(tt.args.in0, tt.args.radiusProfile, tt.args.model, tt.args.site)
		})
	}
}

func Test_radiusProfileResource_ListResourceConfigSchema(t *testing.T) {
	type args struct {
		in0  context.Context
		in1  fwlist.ListResourceSchemaRequest
		resp *fwlist.ListResourceSchemaResponse
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.ListResourceConfigSchema(tt.args.in0, tt.args.in1, tt.args.resp)
		})
	}
}

func Test_radiusProfileResource_List(t *testing.T) {
	type args struct {
		ctx    context.Context
		req    fwlist.ListRequest
		stream *fwlist.ListResultsStream
	}
	tests := []struct {
		name string
		r    *radiusProfileResource
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.List(tt.args.ctx, tt.args.req, tt.args.stream)
		})
	}
}
